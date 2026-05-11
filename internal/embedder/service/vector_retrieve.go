package service

import (
	"context"
	"fmt"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/embedding"
	"github.com/ultravioletrs/cube/internal/embedder/postgres"
)

type vectorRetrieveService struct {
	chunks   *postgres.ChunksRepository
	embedder *embedding.Registry
}

// NewVectorRetrieveService creates a service for vector-based chunk retrieval.
func NewVectorRetrieveService(chunks *postgres.ChunksRepository, embedder *embedding.Registry) domain.VectorRetrieveService {
	return &vectorRetrieveService{chunks: chunks, embedder: embedder}
}

func (s *vectorRetrieveService) Retrieve(ctx context.Context, userID, query string, recordIDs []string, topK int) ([]domain.VectorChunk, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if topK <= 0 {
		topK = 5
	}

	emb, err := s.embedder.ForRecord(domain.Record{Format: domain.RecordFormatText})
	if err != nil {
		return nil, fmt.Errorf("get embedder: %w", err)
	}

	vecs, err := emb.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	results, err := s.chunks.SearchChunks(ctx, userID, vecs[0], topK, recordIDs)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}

	chunks := make([]domain.VectorChunk, len(results))
	for i, r := range results {
		chunks[i] = domain.VectorChunk{
			RecordID:    r.RecordID,
			RecordName:  r.RecordName,
			ExternalURL: r.ExternalURL,
			ChunkIndex:  r.ChunkIndex,
			Content:     r.Content,
		}
	}
	return chunks, nil
}
