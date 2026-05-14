// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/auth"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/service"
	objstore "github.com/ultravioletrs/cube/internal/embedder/storage"
)

const (
	maxUploadBytes         = 128 << 20
	directUploadSourceName = "Direct Uploads"
	directUploadSourceKind = "direct_upload"
)

type directUploadSourceConfig struct {
	Kind string `json:"kind"`
}

func uploadRecord(
	recordsSvc domain.RecordService,
	sourcesSvc domain.SourceService,
	store objstore.Store,
	objectKeyPrefix string,
	trigger func(),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeJSON(w, http.StatusInternalServerError, errBody("object storage is not configured"))
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1024)
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "too large") {
				writeJSON(w, http.StatusRequestEntityTooLarge, errBody("file is too large (max 128MB)"))
				return
			}
			writeJSON(w, http.StatusBadRequest, errBody("invalid multipart form"))
			return
		}

		src, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("missing file field"))
			return
		}
		defer src.Close()

		userID := auth.UserID(r.Context())
		domainID := auth.DomainID(r.Context())
		recordName := strings.TrimSpace(r.FormValue("name"))
		sourceID := strings.TrimSpace(r.FormValue("source_id"))
		if recordName == "" {
			recordName = header.Filename
		}
		if recordName == "" {
			writeJSON(w, http.StatusBadRequest, errBody("file name is required"))
			return
		}

		mimeType, stream, err := detectUploadMIME(src, header)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("failed to read uploaded file"))
			return
		}

		recordFormat := service.DetectRecordFormat(header.Filename, mimeType)
		switch recordFormat {
		case domain.RecordFormatText, domain.RecordFormatMD, domain.RecordFormatPDF, domain.RecordFormatDOCX, domain.RecordFormatCode, domain.RecordFormatImage:
		default:
			writeJSON(w, http.StatusUnsupportedMediaType, errBody("uploaded file type is not supported for direct upload"))
			return
		}

		source, err := resolveUploadSource(r.Context(), sourcesSvc, domainID, userID, sourceID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrNotFound):
				writeJSON(w, http.StatusNotFound, errBody("source not found"))
			default:
				writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			}
			return
		}

		objectKey, err := prepareUploadObjectKey(objectKeyPrefix, userID, header.Filename)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("failed to prepare upload storage"))
			return
		}

		size := header.Size
		if size < 0 {
			size = -1
		}
		normalizedMIME := normalizeUploadMIME(recordFormat, mimeType)
		if err := store.Put(r.Context(), objectKey, normalizedMIME, size, stream); err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("failed to store uploaded file"))
			return
		}

		rec, err := recordsSvc.Create(r.Context(), domain.Record{
			DomainID:   domainID,
			UserID:     userID,
			SourceID:   source.ID,
			Name:       recordName,
			Format:     recordFormat,
			Status:     domain.RecordStatusQueued,
			ExternalID: objectKey,
			MimeType:   normalizedMIME,
		})
		if err != nil {
			_ = store.Delete(r.Context(), objectKey)
			switch {
			case errors.Is(err, domain.ErrConflict):
				writeJSON(w, http.StatusConflict, errBody(err.Error()))
			default:
				writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			}
			return
		}

		if trigger != nil {
			trigger()
		}

		slog.Info("record upload queued", "record_id", rec.ID, "user_id", userID, "format", rec.Format, "source_id", rec.SourceID)
		writeJSON(w, http.StatusCreated, toRecordResponse(rec))
	}
}

func detectUploadMIME(src multipart.File, header *multipart.FileHeader) (string, io.Reader, error) {
	sniff := make([]byte, 512)
	n, err := io.ReadFull(src, sniff)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", nil, err
	}
	sniff = sniff[:n]

	mimeType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if mimeType == "" {
		mimeType = http.DetectContentType(sniff)
	}
	return mimeType, io.MultiReader(bytes.NewReader(sniff), src), nil
}

func prepareUploadObjectKey(prefix, userID, originalName string) (string, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(originalName)))
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	storedName := fmt.Sprintf("%d-%x%s", time.Now().UTC().UnixNano(), random, ext)

	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		prefix = "uploads"
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", fmt.Errorf("user id is required")
	}

	return path.Join(prefix, userID, storedName), nil
}

func resolveUploadSource(
	ctx context.Context,
	sourcesSvc domain.SourceService,
	domainID, userID, sourceID string,
) (domain.Source, error) {
	if sourceID != "" {
		src, err := sourcesSvc.GetByID(ctx, sourceID, domainID)
		if err != nil {
			return domain.Source{}, err
		}
		if src.Type != domain.SourceTypeLocalFS {
			return domain.Source{}, fmt.Errorf("source %q is not a direct upload source", sourceID)
		}
		return src, nil
	}
	return ensureDirectUploadSource(ctx, sourcesSvc, domainID, userID)
}

func ensureDirectUploadSource(
	ctx context.Context,
	sourcesSvc domain.SourceService,
	domainID, userID string,
) (domain.Source, error) {
	const pageLimit uint64 = 100
	var offset uint64

	for {
		page, err := sourcesSvc.List(ctx, domainID, domain.Page{Limit: pageLimit, Offset: offset})
		if err != nil {
			return domain.Source{}, err
		}

		for _, src := range page.Sources {
			if isDirectUploadSource(src) {
				return src, nil
			}
		}

		if page.Total == 0 || offset+pageLimit >= page.Total {
			break
		}
		offset += pageLimit
	}

	cfg, err := json.Marshal(directUploadSourceConfig{Kind: directUploadSourceKind})
	if err != nil {
		return domain.Source{}, err
	}

	created, err := sourcesSvc.Create(ctx, domain.Source{
		DomainID:         domainID,
		UserID:           userID,
		Type:             domain.SourceTypeLocalFS,
		Name:             directUploadSourceName,
		Config:           cfg,
		Status:           domain.SourceStatusActive,
		SyncEnabled:      false,
		AutoSyncInterval: 0,
	})
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, domain.ErrConflict) {
		return domain.Source{}, err
	}

	page, listErr := sourcesSvc.List(ctx, domainID, domain.Page{Limit: pageLimit, Offset: 0})
	if listErr != nil {
		return domain.Source{}, listErr
	}
	for _, src := range page.Sources {
		if isDirectUploadSource(src) {
			return src, nil
		}
	}
	return domain.Source{}, err
}

func isDirectUploadSource(src domain.Source) bool {
	if src.Type != domain.SourceTypeLocalFS {
		return false
	}
	if src.Name == directUploadSourceName {
		return true
	}

	var cfg directUploadSourceConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return false
	}
	return strings.EqualFold(cfg.Kind, directUploadSourceKind)
}

func normalizeUploadMIME(format domain.RecordFormat, detected string) string {
	switch format {
	case domain.RecordFormatPDF:
		return "application/pdf"
	case domain.RecordFormatDOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case domain.RecordFormatMD:
		return "text/markdown"
	case domain.RecordFormatCode, domain.RecordFormatText:
		return "text/plain"
	default:
		return detected
	}
}
