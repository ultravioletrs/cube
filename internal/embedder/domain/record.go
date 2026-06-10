// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"time"
)

// RecordFormat identifies the content type of a record.
type RecordFormat string

const (
	RecordFormatText  RecordFormat = "text"
	RecordFormatPDF   RecordFormat = "pdf"
	RecordFormatMD    RecordFormat = "md"
	RecordFormatDOCX  RecordFormat = "docx"
	RecordFormatCode  RecordFormat = "code"
	RecordFormatImage RecordFormat = "image"
	RecordFormatLink  RecordFormat = "link"
)

// RecordStatus represents the processing state of a record.
type RecordStatus string

const (
	RecordStatusQueued     RecordStatus = "queued"
	RecordStatusProcessing RecordStatus = "processing"
	RecordStatusIndexed    RecordStatus = "indexed"
	RecordStatusFailed     RecordStatus = "failed"
	RecordStatusCancelled  RecordStatus = "cancelled"
)

// Record represents a single indexed item (document, image, link, etc.) in the
// vector store.  It always preserves a reference to the original external document
// so that vectorized content can be traced back to its origin at any time.
type Record struct {
	ID string
	// DomainID scopes this record to a domain (workspace); all domain members share it.
	DomainID string
	// UserID records the creating user for audit purposes.
	UserID string
	// SourceID links back to the Source this record was ingested from.
	SourceID string
	Name     string
	Format   RecordFormat
	Status   RecordStatus

	// ExternalID is the source-system identifier for the original item.
	// For Google Drive: the file ID. For a link: the URL. For a filesystem: the path.
	ExternalID string
	// ExternalURL is the browser-accessible URL of the original document.
	ExternalURL string
	// ExternalRef carries additional location context from the source system
	// (e.g. Google Drive parent folder ID, SharePoint site URL, git commit SHA).
	ExternalRef string
	MimeType    string

	// FolderPath is the human-readable containing-folder path within the source
	// (e.g. /Docs/2024/Q3); FolderID is the immediate parent folder ID. Both are
	// populated for folder-tree ingests (Google Drive); nil otherwise.
	FolderPath *string
	FolderID   *string

	// Content metadata populated after successful ingestion.
	Description         string
	ChunkCount          *int
	SizeBytes           *int64
	PageCount           *int
	IngestTotalChunks   *int
	IngestIndexedChunks *int
	// IngestStage is the current (or last) ingest phase: extracting, chunking,
	// embedding. Nil once queued or after a successful index; on failure it
	// persists so the UI can show where ingest stopped.
	IngestStage *string

	// SourceVersion and SourceModifiedAt enable idempotent re-sync:
	// a record is only re-ingested when the source version changes.
	SourceVersion    string
	SourceModifiedAt *time.Time

	Error     *string
	CreatedAt time.Time
	UpdatedAt time.Time
	Source    *RecordSourceLink
}

// RecordSourceLink carries the original source linkage for a record list/detail response.
type RecordSourceLink struct {
	ID     string
	Name   string
	Type   SourceType
	Status SourceStatus
}

// RecordPage is a paginated result set of records.
type RecordPage struct {
	Records []Record
	Total   uint64
}

// RecordFilter constrains a list query.  All fields are optional.
type RecordFilter struct {
	SourceID *string
	Status   *RecordStatus
	Format   *RecordFormat
	// Name is a case-insensitive substring matched against the record name.
	Name *string
	// FolderPrefix matches records whose folder_path equals or is nested under
	// the given path (prefix match), e.g. "/Docs/2024".
	FolderPrefix *string
}

// IngestResult holds post-ingestion metadata written back to the record.
type IngestResult struct {
	ChunkCount  int
	SizeBytes   int64
	PageCount   *int
	Description string
}

// RecordUpsertState describes what a sync operation changed.
type RecordUpsertState string

const (
	RecordUpsertCreated   RecordUpsertState = "created"
	RecordUpsertUpdated   RecordUpsertState = "updated"
	RecordUpsertUnchanged RecordUpsertState = "unchanged"
)

// RecordUpsertResult is returned when syncing source items into records.
type RecordUpsertResult struct {
	Record Record
	State  RecordUpsertState
}

// RecordRepository defines the persistence contract for records.
type RecordRepository interface {
	Create(ctx context.Context, r Record) (Record, error)
	GetByID(ctx context.Context, id, domainID string) (Record, error)
	List(ctx context.Context, domainID string, f RecordFilter, p Page) (RecordPage, error)
	Delete(ctx context.Context, id, domainID string) error
	DeleteBySourceExternalIDs(ctx context.Context, domainID, sourceID string, externalIDs []string) (int, error)
	UpsertFromSource(ctx context.Context, r Record) (RecordUpsertResult, error)
	// ListQueued returns up to limit records in "queued" status, across all domains.
	ListQueued(ctx context.Context, limit int) ([]Record, error)
	// UpdateStatus transitions a record to the given status (and clears/sets error).
	UpdateStatus(ctx context.Context, id string, s RecordStatus, errMsg string) error
	UpdateIngestProgress(ctx context.Context, id string, indexedChunks, totalChunks int) error
	// UpdateIngestStage records the current ingest phase (extracting/chunking/embedding).
	UpdateIngestStage(ctx context.Context, id, stage string) error
	// UpdateAfterIngest writes chunk_count and size_bytes and marks the record indexed.
	UpdateAfterIngest(ctx context.Context, id string, res IngestResult) error
}

// RecordService defines the business-logic contract for records.
type RecordService interface {
	Create(ctx context.Context, r Record) (Record, error)
	GetByID(ctx context.Context, id, domainID string) (Record, error)
	List(ctx context.Context, domainID string, f RecordFilter, p Page) (RecordPage, error)
	Delete(ctx context.Context, id, domainID string) error
	RetryIngest(ctx context.Context, id, domainID string) error
	CancelIngest(ctx context.Context, id, domainID string) error
}
