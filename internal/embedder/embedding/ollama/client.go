// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client calls an Ollama embedding endpoint.
type Client struct {
	baseURL string
	model   string
	dims    int
	client  *http.Client
}

// New returns an Ollama embedding client.
func New(baseURL, model string, dimensions int) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		dims:    dimensions,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// Dimensions returns the configured vector size.
func (c *Client) Dimensions() int { return c.dims }

// Embed sends a batch embedding request to Ollama.
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	type request struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}
	type response struct {
		Embeddings [][]float32 `json:"embeddings"`
	}

	body, _ := json.Marshal(request{Model: c.model, Input: texts})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		detail := strings.TrimSpace(string(msg))
		if detail == "" {
			return nil, fmt.Errorf("ollama embed status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("ollama embed status %d: %s", resp.StatusCode, detail)
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("ollama embed decode: %w", err)
	}
	if len(res.Embeddings) != len(texts) {
		return nil, fmt.Errorf("ollama embed: got %d embeddings for %d texts", len(res.Embeddings), len(texts))
	}
	return res.Embeddings, nil
}
