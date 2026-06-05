// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/embedding"
	"github.com/ultravioletrs/cube/internal/embedder/imageembedding"
	"github.com/ultravioletrs/cube/internal/embedder/postgres"
)

type vectorRetrieveService struct {
	chunks          *postgres.ChunksRepository
	imageEmbeddings *postgres.ImageEmbeddingsRepository
	embedder        *embedding.Registry
	imageEmbedder   *imageembedding.Client
}

// NewVectorRetrieveService creates a service for vector-based chunk retrieval.
func NewVectorRetrieveService(chunks *postgres.ChunksRepository, embedder *embedding.Registry) domain.VectorRetrieveService {
	return &vectorRetrieveService{chunks: chunks, embedder: embedder}
}

// NewMultimodalRetrieveService creates a retriever that searches text chunks
// plus optional visual image embeddings.
func NewMultimodalRetrieveService(
	chunks *postgres.ChunksRepository,
	imageEmbeddings *postgres.ImageEmbeddingsRepository,
	embedder *embedding.Registry,
	imageEmbedder *imageembedding.Client,
) domain.VectorRetrieveService {
	return &vectorRetrieveService{
		chunks:          chunks,
		imageEmbeddings: imageEmbeddings,
		embedder:        embedder,
		imageEmbedder:   imageEmbedder,
	}
}

func (s *vectorRetrieveService) Retrieve(ctx context.Context, domainID, query string, recordIDs []string, topK int) ([]domain.VectorChunk, error) {
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

	results, err := s.chunks.HybridSearchChunks(ctx, domainID, vecs[0], domain.RetrievalQuery{
		Query:     query,
		RecordIDs: recordIDs,
		TopK:      topK,
	})
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}

	if s.imageEmbeddings != nil && s.imageEmbedder != nil {
		imageQuery, err := s.imageEmbedder.EmbedText(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("embed visual query: %w", err)
		}
		imageResults, err := s.imageEmbeddings.Search(ctx, domainID, imageQuery.Embedding, topK, recordIDs)
		if err != nil {
			return nil, fmt.Errorf("search image embeddings: %w", err)
		}
		results = interleaveChunkResults(results, imageResults, topK)
	}

	chunks := make([]domain.VectorChunk, len(results))
	for i, r := range results {
		chunks[i] = domain.VectorChunk{
			RecordID:    r.RecordID,
			RecordName:  r.RecordName,
			ExternalURL: r.ExternalURL,
			ChunkIndex:  r.ChunkIndex,
			Content:     r.Content,
			Score:       r.Score,
		}
	}
	return chunks, nil
}

func interleaveChunkResults(textResults, imageResults []postgres.ChunkSearchResult, topK int) []postgres.ChunkSearchResult {
	if topK <= 0 {
		topK = 5
	}
	out := make([]postgres.ChunkSearchResult, 0, topK)
	seen := make(map[string]struct{}, topK)
	maxLen := len(textResults)
	if len(imageResults) > maxLen {
		maxLen = len(imageResults)
	}
	for i := 0; i < maxLen && len(out) < topK; i++ {
		if i < len(textResults) {
			addChunkResult(&out, seen, textResults[i], topK)
		}
		if i < len(imageResults) {
			addChunkResult(&out, seen, imageResults[i], topK)
		}
	}
	return out
}

func addChunkResult(out *[]postgres.ChunkSearchResult, seen map[string]struct{}, result postgres.ChunkSearchResult, topK int) {
	if len(*out) >= topK {
		return
	}
	key := fmt.Sprintf("%s:%d:%s", result.RecordID, result.ChunkIndex, result.Content)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*out = append(*out, result)
}
