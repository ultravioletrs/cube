// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

// Package ingest implements the background embedding pipeline.
// It polls for queued records, downloads their content from the source,
// chunks and embeds the text, then stores chunks in the vector store.
package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/embedding"
	"github.com/ultravioletrs/cube/internal/embedder/imageembedding"
	embedmetrics "github.com/ultravioletrs/cube/internal/embedder/metrics"
	"github.com/ultravioletrs/cube/internal/embedder/postgres"
	objstore "github.com/ultravioletrs/cube/internal/embedder/storage"
)

// Worker polls for queued records and runs the embedding pipeline.
type Worker struct {
	records         domain.RecordRepository
	sources         domain.SourceRepository
	chunks          *postgres.ChunksRepository
	embeddings      *embedding.Registry
	imageEmbeddings *postgres.ImageEmbeddingsRepository
	imageEmbedder   *imageembedding.Client
	store           objstore.Store
	sourceProviders *SourceProviderRegistry
	chunkSize       int
	overlap         int
	batchSize       int
	maxConcurrent   int
	embedBatchSize  int
	recordTimeout   time.Duration
	maxChunks       int
	pollInterval    time.Duration
	trigger         chan struct{}
}

const (
	imageEmbeddingMaxAttempts = 3
	imageEmbeddingRetryDelay  = 2 * time.Second
)

var errRecordCancelled = errors.New("record ingest cancelled")

// NewWorker creates an ingestion worker. chunkSize and overlap are in words.
func NewWorker(
	records domain.RecordRepository,
	sources domain.SourceRepository,
	chunks *postgres.ChunksRepository,
	embeddings *embedding.Registry,
	store objstore.Store,
	sourceProviders *SourceProviderRegistry,
	chunkSize, overlap int,
) *Worker {
	return &Worker{
		records:         records,
		sources:         sources,
		chunks:          chunks,
		embeddings:      embeddings,
		store:           store,
		sourceProviders: sourceProviders,
		chunkSize:       chunkSize,
		overlap:         overlap,
		batchSize:       10,
		maxConcurrent:   2,
		embedBatchSize:  16,
		recordTimeout:   15 * time.Minute,
		maxChunks:       0,
		pollInterval:    10 * time.Second,
		trigger:         make(chan struct{}, 1),
	}
}

// SetBatchSize adjusts how many queued records are fetched each iteration.
func (w *Worker) SetBatchSize(size int) {
	if size > 0 {
		w.batchSize = size
	}
}

// SetMaxConcurrent adjusts how many records are ingested in parallel.
func (w *Worker) SetMaxConcurrent(n int) {
	if n > 0 {
		w.maxConcurrent = n
	}
}

// SetEmbedBatchSize adjusts how many chunks are sent in one embedding request.
func (w *Worker) SetEmbedBatchSize(n int) {
	if n > 0 {
		w.embedBatchSize = n
	}
}

// SetRecordTimeout adjusts the maximum wall-clock time for one record ingest.
func (w *Worker) SetRecordTimeout(d time.Duration) {
	if d > 0 {
		w.recordTimeout = d
	}
}

// SetMaxChunks adjusts the maximum number of chunks one record may produce.
func (w *Worker) SetMaxChunks(n int) {
	if n >= 0 {
		w.maxChunks = n
	}
}

// SetPollInterval adjusts how often the worker checks for queued records.
func (w *Worker) SetPollInterval(d time.Duration) {
	if d > 0 {
		w.pollInterval = d
	}
}

// SetImageEmbedding enables optional visual embeddings for image records.
func (w *Worker) SetImageEmbedding(repo *postgres.ImageEmbeddingsRepository, client *imageembedding.Client) {
	w.imageEmbeddings = repo
	w.imageEmbedder = client
}

// Run starts the worker poll loop. It blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processQueued(ctx)
		case <-w.trigger:
			w.processQueued(ctx)
		}
	}
}

// Trigger signals the worker to process queued records immediately.
// It is non-blocking: if the worker is busy the tick will fire again shortly.
func (w *Worker) Trigger() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *Worker) processQueued(ctx context.Context) {
	records, err := w.records.ListQueued(ctx, w.batchSize)
	if err != nil {
		slog.Error("ingest: list queued records", "err", err)
		return
	}
	if len(records) == 0 {
		return
	}

	sem := make(chan struct{}, w.maxConcurrent)
	var wg sync.WaitGroup

	for _, rec := range records {
		if ctx.Err() != nil {
			break
		}
		rec := rec
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			w.processRecord(ctx, rec)
		}()
	}
	wg.Wait()
}

func (w *Worker) processRecord(ctx context.Context, rec domain.Record) {
	logger := slog.With("record_id", rec.ID, "record_name", rec.Name)

	if err := w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusProcessing, ""); err != nil {
		logger.Error("ingest: mark processing", "err", err)
		return
	}

	recordCtx := ctx
	cancel := func() {}
	if w.recordTimeout > 0 {
		recordCtx, cancel = context.WithTimeout(ctx, w.recordTimeout)
	}
	defer cancel()

	if rec.Format == domain.RecordFormatImage {
		w.processImageRecord(recordCtx, ctx, rec, logger)
		return
	}

	if w.isCancelled(ctx, rec) {
		w.cleanupCancelledRecord(ctx, rec, logger)
		return
	}

	text, pageCount, err := w.downloadContent(recordCtx, rec)
	if err != nil {
		logger.Warn("ingest: download failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}
	if w.isCancelled(ctx, rec) {
		w.cleanupCancelledRecord(ctx, rec, logger)
		return
	}

	chunks, plan := adaptiveChunk(text, w.chunkSize, w.overlap)
	if len(chunks) == 0 {
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, "document produced zero chunks")
		return
	}
	logger.Info("ingest: chunked document", "words", plan.Words, "chunk_size", plan.Size, "chunk_overlap", plan.Overlap, "chunks", len(chunks))
	if w.maxChunks > 0 && len(chunks) > w.maxChunks {
		err := fmt.Errorf("document produced %d chunks, above limit %d", len(chunks), w.maxChunks)
		logger.Warn("ingest: too many chunks", "chunks", len(chunks), "max_chunks", w.maxChunks)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	embedder, err := w.embeddings.ForRecord(rec)
	if err != nil {
		logger.Warn("ingest: select embedding model failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	indexedChunks, err := w.embedAndStoreBatched(recordCtx, ctx, rec, embedder, chunks)
	if err != nil {
		if errors.Is(err, errRecordCancelled) {
			w.cleanupCancelledRecord(ctx, rec, logger)
			return
		}
		logger.Warn("ingest: embed failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}
	if w.isCancelled(ctx, rec) {
		w.cleanupCancelledRecord(ctx, rec, logger)
		return
	}

	_ = w.records.UpdateAfterIngest(ctx, rec.ID, domain.IngestResult{
		ChunkCount: indexedChunks,
		SizeBytes:  int64(len(text)),
		PageCount:  pageCount,
	})
	logger.Info("ingest: indexed", "chunks", indexedChunks)
}

func (w *Worker) processImageRecord(ctx, statusCtx context.Context, rec domain.Record, logger *slog.Logger) {
	if w.isCancelled(statusCtx, rec) {
		w.cleanupCancelledRecord(ctx, rec, logger)
		return
	}

	content, err := w.downloadRawContent(ctx, rec)
	if err != nil {
		logger.Warn("ingest: image download failed", "err", err)
		_ = w.records.UpdateStatus(statusCtx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}
	if w.isCancelled(ctx, rec) {
		w.cleanupCancelledRecord(ctx, rec, logger)
		return
	}

	doc, err := ExtractText(DriveFile{
		Name:     rec.Name,
		MimeType: rec.MimeType,
	}, content)
	if err != nil {
		logger.Warn("ingest: image extraction failed", "err", err)
		_ = w.records.UpdateStatus(statusCtx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}
	if w.isCancelled(ctx, rec) {
		w.cleanupCancelledRecord(ctx, rec, logger)
		return
	}
	if doc.ImageMode == ImageIngestModeNone {
		doc.ImageMode = ImageIngestModeImage
	}

	chunks, plan := adaptiveChunk(doc.Text, w.chunkSize, w.overlap)
	if len(chunks) == 0 {
		_ = w.records.UpdateStatus(statusCtx, rec.ID, domain.RecordStatusFailed, "image produced zero chunks")
		return
	}
	logger.Info("ingest: chunked image text", "words", plan.Words, "chunk_size", plan.Size, "chunk_overlap", plan.Overlap, "chunks", len(chunks))
	if w.maxChunks > 0 && len(chunks) > w.maxChunks {
		err := fmt.Errorf("image produced %d chunks, above limit %d", len(chunks), w.maxChunks)
		logger.Warn("ingest: image too many chunks", "chunks", len(chunks), "max_chunks", w.maxChunks)
		_ = w.records.UpdateStatus(statusCtx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	// Text chunks from images are always embedded with the text profile. The
	// visual signal is stored separately in image_embeddings when selected below.
	embedder, err := w.embeddings.ForRecord(domain.Record{Format: domain.RecordFormatText})
	if err != nil {
		logger.Warn("ingest: select text embedding model failed", "err", err)
		_ = w.records.UpdateStatus(statusCtx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	indexedChunks, err := w.embedAndStoreBatched(ctx, statusCtx, rec, embedder, chunks)
	if err != nil {
		if errors.Is(err, errRecordCancelled) {
			w.cleanupCancelledRecord(ctx, rec, logger)
			return
		}
		logger.Warn("ingest: embed image text failed", "err", err)
		_ = w.records.UpdateStatus(statusCtx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}
	if w.isCancelled(ctx, rec) {
		w.cleanupCancelledRecord(ctx, rec, logger)
		return
	}

	if w.imageEmbeddings != nil {
		if err := w.imageEmbeddings.DeleteByRecord(ctx, rec.ID); err != nil {
			logger.Warn("ingest: delete previous image embedding failed", "err", err)
		}
	}
	if doc.ImageMode != ImageIngestModeOCR {
		if w.imageEmbeddings == nil || w.imageEmbedder == nil {
			err := fmt.Errorf("visual image embedding is required for image ingest mode %q but is not configured", doc.ImageMode)
			logger.Warn("ingest: visual image embedding failed", "err", err)
			_ = w.records.UpdateStatus(statusCtx, rec.ID, domain.RecordStatusFailed, err.Error())
			return
		}
		if err := w.storeImageEmbeddingContentWithRetry(ctx, rec, content, logger); err != nil {
			if errors.Is(err, errRecordCancelled) {
				w.cleanupCancelledRecord(ctx, rec, logger)
				return
			}
			logger.Warn("ingest: visual image embedding failed", "err", err)
			_ = w.records.UpdateStatus(statusCtx, rec.ID, domain.RecordStatusFailed, err.Error())
			return
		}
	}
	if w.isCancelled(ctx, rec) {
		w.cleanupCancelledRecord(ctx, rec, logger)
		return
	}

	_ = w.records.UpdateAfterIngest(statusCtx, rec.ID, domain.IngestResult{
		ChunkCount:  indexedChunks,
		SizeBytes:   int64(len(content)),
		PageCount:   doc.PageCount,
		Description: imageIngestDescription(doc),
	})
	logger.Info("ingest: image indexed", "chunks", indexedChunks, "mode", doc.ImageMode, "ocr_chars", doc.OCRTextCharCount)
}

func imageIngestDescription(doc ExtractedDocument) string {
	switch doc.ImageMode {
	case ImageIngestModeOCR:
		return "Image ingest mode: OCR only"
	case ImageIngestModeHybrid:
		return "Image ingest mode: hybrid OCR + visual embedding"
	case ImageIngestModeImage:
		return "Image ingest mode: visual embedding"
	default:
		return ""
	}
}

func (w *Worker) isCancelled(ctx context.Context, rec domain.Record) bool {
	current, err := w.records.GetByID(ctx, rec.ID, rec.DomainID)
	if err != nil {
		return false
	}
	return current.Status == domain.RecordStatusCancelled
}

func (w *Worker) cleanupCancelledRecord(ctx context.Context, rec domain.Record, logger *slog.Logger) {
	if err := w.chunks.DeleteByRecord(ctx, rec.ID); err != nil {
		logger.Warn("ingest: cleanup cancelled chunks failed", "err", err)
	}
	if w.imageEmbeddings != nil {
		if err := w.imageEmbeddings.DeleteByRecord(ctx, rec.ID); err != nil {
			logger.Warn("ingest: cleanup cancelled image embedding failed", "err", err)
		}
	}
	logger.Info("ingest: cancelled")
}

func (w *Worker) embedAndStoreBatched(
	ctx context.Context,
	statusCtx context.Context,
	rec domain.Record,
	embedder embedding.Embedder,
	chunks []string,
) (int, error) {
	batchSize := w.embedBatchSize
	if batchSize <= 0 {
		batchSize = 16
	}
	if w.isCancelled(statusCtx, rec) {
		return 0, errRecordCancelled
	}
	if err := w.chunks.DeleteByRecord(statusCtx, rec.ID); err != nil {
		return 0, err
	}
	totalChunks := len(chunks)
	indexedChunks := 0
	if err := w.records.UpdateIngestProgress(statusCtx, rec.ID, 0, totalChunks); err != nil {
		return 0, err
	}

	for i := 0; i < len(chunks); i += batchSize {
		if w.isCancelled(statusCtx, rec) {
			return 0, errRecordCancelled
		}
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		vecs, err := embedder.Embed(ctx, chunks[i:end])
		if err != nil {
			if !isContextLengthError(err) {
				_ = w.chunks.DeleteByRecord(statusCtx, rec.ID)
				return 0, err
			}
			added, err := w.embedAndStoreSplitFallback(ctx, statusCtx, rec, embedder, chunks[i:end], indexedChunks, &totalChunks)
			if err != nil {
				_ = w.chunks.DeleteByRecord(statusCtx, rec.ID)
				return 0, err
			}
			indexedChunks += added
			continue
		}
		if len(vecs) != end-i {
			_ = w.chunks.DeleteByRecord(statusCtx, rec.ID)
			return 0, fmt.Errorf("embed: got %d embeddings for %d chunks", len(vecs), end-i)
		}

		chunkObjs := make([]postgres.Chunk, len(vecs))
		for j, vec := range vecs {
			chunkObjs[j] = postgres.Chunk{Content: chunks[i+j], Embedding: vec}
		}
		if err := w.chunks.AppendChunks(ctx, rec.DomainID, rec.UserID, rec.ID, indexedChunks, chunkObjs); err != nil {
			_ = w.chunks.DeleteByRecord(statusCtx, rec.ID)
			return 0, err
		}
		indexedChunks += len(chunkObjs)
		if err := w.records.UpdateIngestProgress(statusCtx, rec.ID, indexedChunks, totalChunks); err != nil {
			_ = w.chunks.DeleteByRecord(statusCtx, rec.ID)
			return 0, err
		}
	}
	return indexedChunks, nil
}

func (w *Worker) embedAndStoreSplitFallback(
	ctx context.Context,
	statusCtx context.Context,
	rec domain.Record,
	embedder embedding.Embedder,
	chunks []string,
	startIndex int,
	totalChunks *int,
) (int, error) {
	indexed := 0
	for _, text := range chunks {
		if w.isCancelled(statusCtx, rec) {
			return 0, errRecordCancelled
		}
		parts, vecs, err := embedWithContextSplit(ctx, embedder, text)
		if err != nil {
			return 0, err
		}
		if len(parts) != len(vecs) {
			return 0, fmt.Errorf("embed split: got %d embeddings for %d chunks", len(vecs), len(parts))
		}
		*totalChunks += len(parts) - 1
		chunkObjs := make([]postgres.Chunk, len(parts))
		for i, part := range parts {
			chunkObjs[i] = postgres.Chunk{Content: part, Embedding: vecs[i]}
		}
		if err := w.chunks.AppendChunks(ctx, rec.DomainID, rec.UserID, rec.ID, startIndex+indexed, chunkObjs); err != nil {
			return 0, err
		}
		indexed += len(chunkObjs)
		if err := w.records.UpdateIngestProgress(statusCtx, rec.ID, startIndex+indexed, *totalChunks); err != nil {
			return 0, err
		}
	}
	return indexed, nil
}

func embedWithContextSplit(ctx context.Context, embedder embedding.Embedder, text string) ([]string, [][]float32, error) {
	vecs, err := embedder.Embed(ctx, []string{text})
	if err == nil {
		return []string{text}, vecs, nil
	}
	if !isContextLengthError(err) {
		return nil, nil, err
	}
	left, right, ok := splitTextForContext(text)
	if !ok {
		return nil, nil, err
	}
	leftParts, leftVecs, err := embedWithContextSplit(ctx, embedder, left)
	if err != nil {
		return nil, nil, err
	}
	rightParts, rightVecs, err := embedWithContextSplit(ctx, embedder, right)
	if err != nil {
		return nil, nil, err
	}
	return append(leftParts, rightParts...), append(leftVecs, rightVecs...), nil
}

func splitTextForContext(text string) (string, string, bool) {
	words := strings.Fields(text)
	if len(words) < 2 {
		return "", "", false
	}
	mid := len(words) / 2
	return strings.Join(words[:mid], " "), strings.Join(words[mid:], " "), true
}

func isContextLengthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context length") || strings.Contains(msg, "input length exceeds")
}

func (w *Worker) storeImageEmbedding(ctx context.Context, rec domain.Record, logger *slog.Logger) error {
	content, err := w.downloadRawContent(ctx, rec)
	if err != nil {
		return err
	}
	return w.storeImageEmbeddingContentWithRetry(ctx, rec, content, logger)
}

func (w *Worker) storeImageEmbeddingContentWithRetry(ctx context.Context, rec domain.Record, content []byte, logger *slog.Logger) error {
	var lastErr error
	for attempt := 1; attempt <= imageEmbeddingMaxAttempts; attempt++ {
		if w.isCancelled(ctx, rec) {
			return errRecordCancelled
		}
		if err := w.storeImageEmbeddingContent(ctx, rec, content, logger); err != nil {
			lastErr = err
			if attempt == imageEmbeddingMaxAttempts {
				break
			}
			logger.Warn("ingest: visual image embedding retrying", "attempt", attempt, "max_attempts", imageEmbeddingMaxAttempts, "err", err)
			timer := time.NewTimer(imageEmbeddingRetryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("visual image embedding cancelled: %w", ctx.Err())
			case <-timer.C:
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("visual image embedding failed after %d attempts: %w", imageEmbeddingMaxAttempts, lastErr)
}

func (w *Worker) storeImageEmbeddingContent(ctx context.Context, rec domain.Record, content []byte, logger *slog.Logger) error {
	result, err := w.imageEmbedder.EmbedImage(ctx, rec.Name, rec.MimeType, content)
	if err != nil {
		return err
	}
	if err := w.imageEmbeddings.Store(ctx, postgres.ImageEmbedding{
		DomainID:   rec.DomainID,
		UserID:     rec.UserID,
		RecordID:   rec.ID,
		Model:      result.Model,
		Dimensions: result.Dimensions,
		Embedding:  result.Embedding,
	}); err != nil {
		return err
	}
	logger.Info("ingest: stored image embedding", "model", result.Model, "dimensions", result.Dimensions)
	return nil
}

// downloadContent fetches the raw text for a record.
// For Drive sources it reads via the Drive API; for direct uploads it reads
// from the configured object storage backend.
func (w *Worker) downloadContent(ctx context.Context, rec domain.Record) (string, *int, error) {
	if rec.SourceID == "" {
		return "", nil, fmt.Errorf("record %s is missing source_id", rec.ID)
	}

	src, err := w.sources.GetByID(ctx, rec.SourceID, rec.DomainID)
	if err != nil {
		return "", nil, err
	}

	switch src.Type {
	case domain.SourceTypeLocalFS:
		started := time.Now().UTC()
		text, pageCount, err := w.downloadFromLocalSource(ctx, rec, src)
		embedmetrics.ObserveSourceDownload(
			string(src.Type),
			string(domain.SourceTypeLocalFS),
			time.Since(started),
			err,
		)
		return text, pageCount, err
	default:
		provider, ok := w.sourceProviders.Provider(src.Type)
		if !ok {
			return "", nil, fmt.Errorf("unsupported source type %q for ingestion", src.Type)
		}
		started := time.Now().UTC()
		text, pageCount, err := provider.DownloadRecord(ctx, rec, src)
		embedmetrics.ObserveSourceDownload(
			string(src.Type),
			string(provider.Type()),
			time.Since(started),
			err,
		)
		return text, pageCount, err
	}
}

func (w *Worker) downloadRawContent(ctx context.Context, rec domain.Record) ([]byte, error) {
	if rec.SourceID == "" {
		return nil, fmt.Errorf("record %s is missing source_id", rec.ID)
	}

	src, err := w.sources.GetByID(ctx, rec.SourceID, rec.DomainID)
	if err != nil {
		return nil, err
	}

	if src.Type == domain.SourceTypeLocalFS {
		return w.downloadRawFromLocalSource(ctx, rec, src)
	}

	provider, ok := w.sourceProviders.Provider(src.Type)
	if !ok {
		return nil, fmt.Errorf("unsupported source type %q for raw image download", src.Type)
	}
	rawProvider, ok := provider.(RawContentProvider)
	if !ok {
		return nil, fmt.Errorf("source type %q does not support raw content download", src.Type)
	}
	return rawProvider.DownloadRecordContent(ctx, rec, src)
}

type localUploadConfig struct {
	Kind      string `json:"kind,omitempty"`
	UploadDir string `json:"upload_dir"`
}

func (w *Worker) downloadFromLocalSource(ctx context.Context, rec domain.Record, src domain.Source) (string, *int, error) {
	body, err := w.downloadRawFromLocalSource(ctx, rec, src)
	if err != nil {
		return "", nil, err
	}

	doc, err := ExtractText(DriveFile{
		Name:     rec.Name,
		MimeType: rec.MimeType,
	}, body)
	if err != nil {
		return "", nil, err
	}
	return doc.Text, doc.PageCount, nil
}

func (w *Worker) downloadRawFromLocalSource(ctx context.Context, rec domain.Record, src domain.Source) ([]byte, error) {
	if rec.ExternalID == "" {
		return nil, fmt.Errorf("record %s is missing external_id", rec.ID)
	}

	// New direct uploads store external_id as an object key.
	if strings.Contains(rec.ExternalID, "/") {
		if w.store == nil {
			return nil, fmt.Errorf("object storage is not configured")
		}
		body, err := w.store.Get(ctx, rec.ExternalID)
		if err != nil {
			return nil, err
		}
		return body, nil
	}

	// Legacy local_fs records might still point to upload_dir + file name.
	var cfg localUploadConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.UploadDir) == "" {
		return nil, fmt.Errorf("local source %s is missing upload_dir config", src.ID)
	}

	fileName := filepath.Base(rec.ExternalID)
	if fileName != rec.ExternalID {
		return nil, fmt.Errorf("invalid local upload file id")
	}

	path := filepath.Join(cfg.UploadDir, rec.UserID, fileName)
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return body, nil
}
