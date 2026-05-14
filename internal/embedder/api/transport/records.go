// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ultravioletrs/cube/internal/embedder/auth"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
	objstore "github.com/ultravioletrs/cube/internal/embedder/storage"
)

// MountRecords registers Record routes on the given router.
// All routes require an authenticated user (auth.Middleware must run first).
func MountRecords(
	r chi.Router,
	recordsSvc domain.RecordService,
	sourcesSvc domain.SourceService,
	store objstore.Store,
	objectKeyPrefix string,
	trigger func(),
) {
	r.Post("/api/v1/records/upload", uploadRecord(recordsSvc, sourcesSvc, store, objectKeyPrefix, trigger))
	r.Get("/api/v1/records", listRecords(recordsSvc))
	r.Get("/api/v1/records/{id}", getRecord(recordsSvc))
	r.Post("/api/v1/records/{id}/retry", retryRecord(recordsSvc, trigger))
	r.Delete("/api/v1/records/{id}", deleteRecord(recordsSvc))
	r.Get("/api/v1/sources/{source_id}/records", listRecordsBySource(recordsSvc))
}

// recordResponse is the JSON shape returned by record endpoints.
// ExternalID / ExternalURL / ExternalRef carry the full traceability chain back
// to the original document in the source system.
type recordResponse struct {
	ID          string                `json:"id"`
	UserID      string                `json:"user_id"`
	SourceID    string                `json:"source_id"`
	Name        string                `json:"name"`
	Format      string                `json:"format"`
	Status      string                `json:"status"`
	ExternalID  string                `json:"external_id"`
	ExternalURL string                `json:"external_url"`
	ExternalRef string                `json:"external_ref,omitempty"`
	MimeType    string                `json:"mime_type,omitempty"`
	Description string                `json:"description,omitempty"`
	ChunkCount  *int                  `json:"chunks,omitempty"`
	SizeBytes   *int64                `json:"size_bytes,omitempty"`
	PageCount   *int                  `json:"pages,omitempty"`
	Error       *string               `json:"error,omitempty"`
	CreatedAt   string                `json:"created_at"`
	UpdatedAt   string                `json:"updated_at"`
	Source      *recordSourceResponse `json:"source,omitempty"`
}

type recordSourceResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	Status     string `json:"status"`
}

func toRecordResponse(rec domain.Record) recordResponse {
	resp := recordResponse{
		ID:          rec.ID,
		UserID:      rec.UserID,
		SourceID:    rec.SourceID,
		Name:        rec.Name,
		Format:      string(rec.Format),
		Status:      string(rec.Status),
		ExternalID:  rec.ExternalID,
		ExternalURL: rec.ExternalURL,
		ExternalRef: rec.ExternalRef,
		MimeType:    rec.MimeType,
		Description: rec.Description,
		ChunkCount:  rec.ChunkCount,
		SizeBytes:   rec.SizeBytes,
		PageCount:   rec.PageCount,
		Error:       rec.Error,
		CreatedAt:   rec.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   rec.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if rec.Source != nil {
		resp.Source = &recordSourceResponse{
			ID:         rec.Source.ID,
			Name:       rec.Source.Name,
			SourceType: string(rec.Source.Type),
			Status:     string(rec.Source.Status),
		}
	}
	return resp
}

type recordListResponse struct {
	Records []recordResponse `json:"records"`
	Total   uint64           `json:"total"`
	Offset  uint64           `json:"offset"`
	Limit   uint64           `json:"limit"`
}

func getRecord(svc domain.RecordService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		domainID := auth.DomainID(r.Context())

		rec, err := svc.GetByID(r.Context(), id, domainID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errBody("record not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		writeJSON(w, http.StatusOK, toRecordResponse(rec))
	}
}

func listRecords(svc domain.RecordService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domainID := auth.DomainID(r.Context())
		p := parsePage(r)
		f := parseRecordFilter(r)

		page, err := svc.List(r.Context(), domainID, f, p)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		writeJSON(w, http.StatusOK, toRecordListResponse(page, p))
	}
}

func listRecordsBySource(svc domain.RecordService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domainID := auth.DomainID(r.Context())
		sourceID := chi.URLParam(r, "source_id")
		p := parsePage(r)
		f := parseRecordFilter(r)
		f.SourceID = &sourceID

		page, err := svc.List(r.Context(), domainID, f, p)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		writeJSON(w, http.StatusOK, toRecordListResponse(page, p))
	}
}

func deleteRecord(svc domain.RecordService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		domainID := auth.DomainID(r.Context())

		if err := svc.Delete(r.Context(), id, domainID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errBody("record not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func retryRecord(svc domain.RecordService, trigger func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		domainID := auth.DomainID(r.Context())

		if err := svc.RetryIngest(r.Context(), id, domainID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errBody("record not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		trigger()
		w.WriteHeader(http.StatusAccepted)
	}
}

func parseRecordFilter(r *http.Request) domain.RecordFilter {
	var f domain.RecordFilter
	if s := r.URL.Query().Get("source_id"); s != "" {
		f.SourceID = &s
	}
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.RecordStatus(s)
		f.Status = &st
	}
	if s := r.URL.Query().Get("format"); s != "" {
		fmt := domain.RecordFormat(s)
		f.Format = &fmt
	}
	return f
}

func toRecordListResponse(page domain.RecordPage, p domain.Page) recordListResponse {
	resp := recordListResponse{
		Total:  page.Total,
		Offset: p.Offset,
		Limit:  p.Limit,
	}
	for _, rec := range page.Records {
		resp.Records = append(resp.Records, toRecordResponse(rec))
	}
	if resp.Records == nil {
		resp.Records = []recordResponse{}
	}
	return resp
}
