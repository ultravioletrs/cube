// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ChunksRepository writes text chunks and their embedding vectors to the
// chunks table, linking them to a record by record_id.
type ChunksRepository struct {
	db *pgxpool.Pool
}

// NewChunksRepository creates a ChunksRepository backed by pool.
func NewChunksRepository(db *pgxpool.Pool) *ChunksRepository {
	return &ChunksRepository{db: db}
}

// Chunk is a single chunk of text with its embedding vector.
type Chunk struct {
	Content   string
	Embedding []float32
}

// StoreChunks deletes existing chunks for recordID then inserts the new batch.
// The embedding column is written as a pgvector literal: '[f1,f2,...]'.
func (r *ChunksRepository) StoreChunks(ctx context.Context, userID, recordID string, chunks []Chunk) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM chunks WHERE record_id = $1`, recordID,
	); err != nil {
		return fmt.Errorf("delete old chunks: %w", err)
	}

	for i, c := range chunks {
		vec := float32SliceToPGVector(c.Embedding)
		_, err := tx.Exec(ctx,
			`INSERT INTO chunks (user_id, record_id, content, embedding, chunk_index)
			 VALUES ($1, $2, $3, $4::vector, $5)`,
			userID, recordID, c.Content, vec, i,
		)
		if err != nil {
			return fmt.Errorf("insert chunk %d: %w", i, err)
		}
	}

	return tx.Commit(ctx)
}

// ChunkSearchResult is a retrieved chunk with the record metadata needed for
// building citations.
type ChunkSearchResult struct {
	Content     string
	RecordID    string
	RecordName  string
	ExternalURL string
	ChunkIndex  int
}

// SearchChunks performs a cosine-distance vector similarity search over the
// authenticated user's chunks.  If recordIDs is non-empty the search is
// scoped to those records only.  Results are ordered by ascending distance
// (most similar first).
func (r *ChunksRepository) SearchChunks(ctx context.Context, userID string, queryVec []float32, limit int, recordIDs []string) ([]ChunkSearchResult, error) {
	vec := float32SliceToPGVector(queryVec)

	var sb strings.Builder
	args := []any{userID, vec, limit} // $1 $2 $3

	sb.WriteString(`
		SELECT c.content, c.record_id, rec.name, COALESCE(rec.external_url, ''), c.chunk_index
		FROM chunks c
		JOIN records rec ON rec.id = c.record_id
		WHERE c.user_id = $1
		  AND c.embedding IS NOT NULL`)

	if len(recordIDs) > 0 {
		ph := make([]string, len(recordIDs))
		for i, id := range recordIDs {
			args = append(args, id)
			ph[i] = fmt.Sprintf("$%d", len(args))
		}
		sb.WriteString(fmt.Sprintf(" AND c.record_id IN (%s)", strings.Join(ph, ",")))
	}

	sb.WriteString(` ORDER BY c.embedding <-> $2::vector LIMIT $3`)

	rows, err := r.db.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()

	var results []ChunkSearchResult
	for rows.Next() {
		var res ChunkSearchResult
		if err := rows.Scan(&res.Content, &res.RecordID, &res.RecordName, &res.ExternalURL, &res.ChunkIndex); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		results = append(results, res)
	}
	return results, rows.Err()
}

// float32SliceToPGVector formats a float32 slice as a pgvector literal string.
func float32SliceToPGVector(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", f)
	}
	b.WriteByte(']')
	return b.String()
}
