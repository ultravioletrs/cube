// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"

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
