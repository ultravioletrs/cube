// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package imageembedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Result is the image vector returned by an image embedding service.
type Result struct {
	Embedding  []float32 `json:"embedding"`
	Model      string    `json:"model"`
	Dimensions int       `json:"dimensions"`
}

// Client calls the external image embedding sidecar.
type Client struct {
	baseURL string
	model   string
	dims    int
	http    *http.Client
}

// New returns a sidecar client. If timeout is <= 0, a conservative default is used.
func New(baseURL, model string, dims int, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   strings.TrimSpace(model),
		dims:    dims,
		http:    &http.Client{Timeout: timeout},
	}
}

type embedRequest struct {
	ImageBase64 string `json:"image_base64"`
	MimeType    string `json:"mime_type,omitempty"`
	Model       string `json:"model,omitempty"`
	Dimensions  int    `json:"dimensions,omitempty"`
}

// EmbedImage sends image bytes to the sidecar and validates the returned dimensions.
func (c *Client) EmbedImage(ctx context.Context, name, mimeType string, image []byte) (Result, error) {
	if len(image) == 0 {
		return Result{}, fmt.Errorf("image embedding: empty image")
	}
	if strings.TrimSpace(mimeType) == "" {
		mimeType = mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	}

	body, err := json.Marshal(embedRequest{
		ImageBase64: base64Encode(image),
		MimeType:    mimeType,
		Model:       c.model,
		Dimensions:  c.dims,
	})
	if err != nil {
		return Result{}, fmt.Errorf("image embedding marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embed-image", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("image embedding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("image embedding status %d", resp.StatusCode)
	}

	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Result{}, fmt.Errorf("image embedding decode: %w", err)
	}
	if len(result.Embedding) == 0 {
		return Result{}, fmt.Errorf("image embedding: empty vector")
	}
	if c.dims > 0 && len(result.Embedding) != c.dims {
		return Result{}, fmt.Errorf("image embedding: expected %d dimensions, got %d", c.dims, len(result.Embedding))
	}
	if result.Dimensions == 0 {
		result.Dimensions = len(result.Embedding)
	}
	if strings.TrimSpace(result.Model) == "" {
		result.Model = c.model
	}
	return result, nil
}
