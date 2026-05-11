package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

type retrievalService struct {
	repo domain.RetrievalRepository
}

// NewRetrievalService creates a retrieval service over stored chunks.
func NewRetrievalService(repo domain.RetrievalRepository) domain.RetrievalService {
	return &retrievalService{repo: repo}
}

func (s *retrievalService) Retrieve(
	ctx context.Context,
	userID string,
	q domain.RetrievalQuery,
) (domain.RetrievalResult, error) {
	if strings.TrimSpace(userID) == "" {
		return domain.RetrievalResult{}, fmt.Errorf("user_id is required")
	}
	if strings.TrimSpace(q.Query) == "" {
		return domain.RetrievalResult{}, fmt.Errorf("query is required")
	}
	if q.TopK <= 0 {
		q.TopK = 5
	}
	if q.TopK > 20 {
		q.TopK = 20
	}

	matches, err := s.repo.KeywordSearchChunks(ctx, userID, q)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return domain.RetrievalResult{
		Query:   q.Query,
		Matches: matches,
		Total:   len(matches),
	}, nil
}
