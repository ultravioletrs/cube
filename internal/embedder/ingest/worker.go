// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

// Package ingest implements the background embedding pipeline.
// It polls for queued records, downloads their content from the source,
// chunks and embeds the text, then stores chunks in the vector store.
package ingest

import (
	"context"
	"encoding/json"
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
	pollInterval    time.Duration
	trigger         chan struct{}
}

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

	if rec.Format == domain.RecordFormatImage {
		w.processImageRecord(ctx, rec, logger)
		return
	}

	text, pageCount, err := w.downloadContent(ctx, rec)
	if err != nil {
		logger.Warn("ingest: download failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	chunks := chunk(text, w.chunkSize, w.overlap)
	if len(chunks) == 0 {
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, "document produced zero chunks")
		return
	}

	embedder, err := w.embeddings.ForRecord(rec)
	if err != nil {
		logger.Warn("ingest: select embedding model failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	embeddings, err := embedBatched(ctx, embedder, chunks, w.embedBatchSize)
	if err != nil {
		logger.Warn("ingest: embed failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	chunkObjs := make([]postgres.Chunk, len(chunks))
	for i, c := range chunks {
		chunkObjs[i] = postgres.Chunk{Content: c, Embedding: embeddings[i]}
	}

	if err := w.chunks.StoreChunks(ctx, rec.DomainID, rec.UserID, rec.ID, chunkObjs); err != nil {
		logger.Error("ingest: store chunks", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	_ = w.records.UpdateAfterIngest(ctx, rec.ID, domain.IngestResult{
		ChunkCount: len(chunks),
		SizeBytes:  int64(len(text)),
		PageCount:  pageCount,
	})
	logger.Info("ingest: indexed", "chunks", len(chunks))
}

func (w *Worker) processImageRecord(ctx context.Context, rec domain.Record, logger *slog.Logger) {
	content, err := w.downloadRawContent(ctx, rec)
	if err != nil {
		logger.Warn("ingest: image download failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	doc, err := ExtractText(DriveFile{
		Name:     rec.Name,
		MimeType: rec.MimeType,
	}, content)
	if err != nil {
		logger.Warn("ingest: image extraction failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}
	if doc.ImageMode == ImageIngestModeNone {
		doc.ImageMode = ImageIngestModeImage
	}

	chunks := chunk(doc.Text, w.chunkSize, w.overlap)
	if len(chunks) == 0 {
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, "image produced zero chunks")
		return
	}

	// Text chunks from images are always embedded with the text profile. The
	// visual signal is stored separately in image_embeddings when selected below.
	embedder, err := w.embeddings.ForRecord(domain.Record{Format: domain.RecordFormatText})
	if err != nil {
		logger.Warn("ingest: select text embedding model failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	embeddings, err := embedBatched(ctx, embedder, chunks, w.embedBatchSize)
	if err != nil {
		logger.Warn("ingest: embed image text failed", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	chunkObjs := make([]postgres.Chunk, len(chunks))
	for i, c := range chunks {
		chunkObjs[i] = postgres.Chunk{Content: c, Embedding: embeddings[i]}
	}

	if err := w.chunks.StoreChunks(ctx, rec.DomainID, rec.UserID, rec.ID, chunkObjs); err != nil {
		logger.Error("ingest: store image text chunks", "err", err)
		_ = w.records.UpdateStatus(ctx, rec.ID, domain.RecordStatusFailed, err.Error())
		return
	}

	if w.imageEmbeddings != nil {
		if err := w.imageEmbeddings.DeleteByRecord(ctx, rec.ID); err != nil {
			logger.Warn("ingest: delete previous image embedding failed", "err", err)
		}
	}
	if doc.ImageMode != ImageIngestModeOCR && w.imageEmbeddings != nil && w.imageEmbedder != nil {
		if err := w.storeImageEmbeddingContent(ctx, rec, content, logger); err != nil {
			logger.Warn("ingest: visual image embedding skipped", "err", err)
		}
	}

	_ = w.records.UpdateAfterIngest(ctx, rec.ID, domain.IngestResult{
		ChunkCount:  len(chunks),
		SizeBytes:   int64(len(content)),
		PageCount:   doc.PageCount,
		Description: imageIngestDescription(doc),
	})
	logger.Info("ingest: image indexed", "chunks", len(chunks), "mode", doc.ImageMode, "ocr_chars", doc.OCRTextCharCount)
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

// embedBatched sends chunks to the embedder in groups so a single large
// document does not produce one enormous HTTP request that times out.
func embedBatched(ctx context.Context, embedder embedding.Embedder, chunks []string, batchSize int) ([][]float32, error) {
	if batchSize <= 0 {
		batchSize = 16
	}
	all := make([][]float32, 0, len(chunks))
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		vecs, err := embedder.Embed(ctx, chunks[i:end])
		if err != nil {
			return nil, err
		}
		all = append(all, vecs...)
	}
	return all, nil
}

func (w *Worker) storeImageEmbedding(ctx context.Context, rec domain.Record, logger *slog.Logger) error {
	content, err := w.downloadRawContent(ctx, rec)
	if err != nil {
		return err
	}
	return w.storeImageEmbeddingContent(ctx, rec, content, logger)
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
