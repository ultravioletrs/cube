// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

type retrievalRepoStub struct {
	called bool
	userID string
	query  domain.RetrievalQuery
	out    []domain.ChunkMatch
	err    error
}

func (s *retrievalRepoStub) KeywordSearchChunks(
	_ context.Context,
	userID string,
	q domain.RetrievalQuery,
) ([]domain.ChunkMatch, error) {
	s.called = true
	s.userID = userID
	s.query = q
	return s.out, s.err
}

func TestRetrieveValidatesAndScopesUser(t *testing.T) {
	repo := &retrievalRepoStub{
		out: []domain.ChunkMatch{
			{ChunkID: "c1", RecordID: "r1", RecordName: "doc.md", RecordFormat: domain.RecordFormatMD, Content: "hello"},
		},
	}
	svc := NewRetrievalService(repo)

	result, err := svc.Retrieve(context.Background(), "u1", domain.RetrievalQuery{
		Query:     "hello world",
		RecordIDs: []string{"r1"},
		TopK:      99,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called {
		t.Fatalf("expected repository call")
	}
	if repo.userID != "u1" {
		t.Fatalf("expected user_id u1, got %q", repo.userID)
	}
	if repo.query.TopK != 20 {
		t.Fatalf("expected top_k clamped to 20, got %d", repo.query.TopK)
	}
	if result.Total != 1 || len(result.Matches) != 1 {
		t.Fatalf("expected 1 match, got total=%d len=%d", result.Total, len(result.Matches))
	}
}

func TestRetrieveReturnsEmptyMatches(t *testing.T) {
	repo := &retrievalRepoStub{out: []domain.ChunkMatch{}}
	svc := NewRetrievalService(repo)

	result, err := svc.Retrieve(context.Background(), "u1", domain.RetrievalQuery{
		Query: "nohits",
		TopK:  0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 || len(result.Matches) != 0 {
		t.Fatalf("expected empty result, got total=%d len=%d", result.Total, len(result.Matches))
	}
	if repo.query.TopK != 5 {
		t.Fatalf("expected default top_k=5, got %d", repo.query.TopK)
	}
}
