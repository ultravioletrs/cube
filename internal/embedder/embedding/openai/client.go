// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client calls the OpenAI embeddings endpoint.
type Client struct {
	baseURL string
	model   string
	apiKey  string
	dims    int
	client  *http.Client
}

// New returns an OpenAI embedding client.
func New(baseURL, model, apiKey string, dimensions int) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		dims:    dimensions,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// Dimensions returns the configured vector size.
func (c *Client) Dimensions() int { return c.dims }

// Embed sends a batch embedding request to OpenAI.
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	type request struct {
		Model      string   `json:"model"`
		Input      []string `json:"input"`
		Dimensions *int     `json:"dimensions,omitempty"`
	}
	type embeddingObject struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	}
	type response struct {
		Data []embeddingObject `json:"data"`
	}

	reqBody := request{
		Model: c.model,
		Input: texts,
	}
	if c.dims > 0 {
		reqBody.Dimensions = &c.dims
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embed status %d", resp.StatusCode)
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("openai embed decode: %w", err)
	}
	if len(res.Data) != len(texts) {
		return nil, fmt.Errorf("openai embed: got %d embeddings for %d texts", len(res.Data), len(texts))
	}

	vecs := make([][]float32, len(texts))
	for _, item := range res.Data {
		if item.Index < len(vecs) {
			if c.dims > 0 && len(item.Embedding) != c.dims {
				return nil, fmt.Errorf("openai embed: expected %d dimensions, got %d", c.dims, len(item.Embedding))
			}
			vecs[item.Index] = item.Embedding
		}
	}
	return vecs, nil
}
