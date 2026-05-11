// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

// RetrievalQuery defines a user-scoped chunk retrieval request.
type RetrievalQuery struct {
	Query     string
	RecordIDs []string
	SourceIDs []string
	TopK      int
}

// ChunkMatch is a single retrieved chunk with source metadata.
type ChunkMatch struct {
	ChunkID      string
	RecordID     string
	RecordName   string
	RecordFormat RecordFormat
	SourceID     string
	SourceName   string
	ChunkIndex   int
	PageNumber   *int
	Content      string
}

// RetrievalResult holds chunk retrieval output for one query.
type RetrievalResult struct {
	Query   string
	Matches []ChunkMatch
	Total   int
}

// RetrievalRepository provides low-level retrieval over stored chunks.
type RetrievalRepository interface {
	KeywordSearchChunks(ctx context.Context, userID string, q RetrievalQuery) ([]ChunkMatch, error)
}

// RetrievalService provides user-scoped retrieval business logic.
type RetrievalService interface {
	Retrieve(ctx context.Context, userID string, q RetrievalQuery) (RetrievalResult, error)
}
