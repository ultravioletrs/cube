// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Reranker calls the Ollama /api/rerank endpoint with a cross-encoder model.
type Reranker struct {
	baseURL string
	model   string
	http    *http.Client
}

// NewReranker returns a reranker backed by Ollama's /api/rerank endpoint.
// model should be a cross-encoder such as "bge-reranker-v2-m3".
func NewReranker(baseURL, model string) *Reranker {
	return &Reranker{
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{},
	}
}

type rerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

type rerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type rerankResponse struct {
	Results []rerankResult `json:"results"`
}

// Rerank returns relevance scores in the same order as documents.
func (r *Reranker) Rerank(ctx context.Context, query string, documents []string) ([]float64, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	body, _ := json.Marshal(rerankRequest{
		Model:     r.model,
		Query:     query,
		Documents: documents,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/api/rerank", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama rerank: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama rerank status %d", resp.StatusCode)
	}

	var result rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode rerank response: %w", err)
	}

	scores := make([]float64, len(documents))
	for _, r := range result.Results {
		if r.Index >= 0 && r.Index < len(scores) {
			scores[r.Index] = r.RelevanceScore
		}
	}
	return scores, nil
}
