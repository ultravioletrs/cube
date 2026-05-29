// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ImageEmbedding is one visual embedding for an image record.
type ImageEmbedding struct {
	DomainID   string
	UserID     string
	RecordID   string
	Model      string
	Dimensions int
	Embedding  []float32
}

// ImageEmbeddingsRepository stores visual embeddings separately from text chunks.
type ImageEmbeddingsRepository struct {
	db dbExecutor
}

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// NewImageEmbeddingsRepository creates an image embeddings repository.
func NewImageEmbeddingsRepository(db dbExecutor) *ImageEmbeddingsRepository {
	return &ImageEmbeddingsRepository{db: db}
}

// Store replaces the visual embedding for a record.
func (r *ImageEmbeddingsRepository) Store(ctx context.Context, emb ImageEmbedding) error {
	vec := float32SliceToPGVector(emb.Embedding)
	_, err := r.db.Exec(ctx, `
		INSERT INTO image_embeddings (domain_id, user_id, record_id, model, dimensions, embedding)
		VALUES ($1, $2, $3, $4, $5, $6::vector)
		ON CONFLICT (record_id) DO UPDATE
		SET domain_id = EXCLUDED.domain_id,
		    user_id = EXCLUDED.user_id,
		    model = EXCLUDED.model,
		    dimensions = EXCLUDED.dimensions,
		    embedding = EXCLUDED.embedding,
		    created_at = now()`,
		emb.DomainID,
		emb.UserID,
		emb.RecordID,
		emb.Model,
		emb.Dimensions,
		vec,
	)
	if err != nil {
		return fmt.Errorf("store image embedding: %w", err)
	}
	return nil
}

// DeleteByRecord removes any visual embedding for recordID.
func (r *ImageEmbeddingsRepository) DeleteByRecord(ctx context.Context, recordID string) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM image_embeddings WHERE record_id = $1`, recordID); err != nil {
		return fmt.Errorf("delete image embedding: %w", err)
	}
	return nil
}

// Search returns records whose visual image vector is closest to queryVec.
func (r *ImageEmbeddingsRepository) Search(
	ctx context.Context,
	domainID string,
	queryVec []float32,
	limit int,
	recordIDs []string,
) ([]ChunkSearchResult, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}

	vec := float32SliceToPGVector(queryVec)
	args := []any{domainID, vec}
	next := 3

	var recordIDFilter string
	if len(recordIDs) > 0 {
		phs := make([]string, 0, len(recordIDs))
		for _, id := range recordIDs {
			if id = strings.TrimSpace(id); id == "" {
				continue
			}
			phs = append(phs, fmt.Sprintf("$%d", next))
			args = append(args, id)
			next++
		}
		if len(phs) > 0 {
			recordIDFilter = " AND rec.id IN (" + strings.Join(phs, ",") + ")"
		}
	}

	args = append(args, limit)
	limitPH := fmt.Sprintf("$%d", next)

	query := `
		SELECT rec.id,
		       rec.name,
		       COALESCE(rec.external_url, ''),
		       COALESCE(rec.description, '')
		FROM image_embeddings ie
		JOIN records rec ON rec.id = ie.record_id
		WHERE ie.domain_id = $1 AND rec.status = 'indexed'` + recordIDFilter + `
		ORDER BY ie.embedding <-> $2::vector
		LIMIT ` + limitPH

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search image embeddings: %w", err)
	}
	defer rows.Close()

	results := make([]ChunkSearchResult, 0, limit)
	for rows.Next() {
		var res ChunkSearchResult
		var description string
		if err := rows.Scan(&res.RecordID, &res.RecordName, &res.ExternalURL, &description); err != nil {
			return nil, fmt.Errorf("scan image embedding result: %w", err)
		}
		res.ChunkIndex = -1
		res.Content = "Visual image match: " + res.RecordName
		if strings.TrimSpace(description) != "" {
			res.Content += ". " + description
		}
		results = append(results, res)
	}
	return results, rows.Err()
}
