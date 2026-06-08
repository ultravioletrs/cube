// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package imageembedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEmbedImageSendsRequestAndValidatesDimensions(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embed-image" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(Result{
			Embedding:  []float32{0.1, 0.2, 0.3},
			Model:      "test",
			Dimensions: 3,
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "test", 3, time.Second)
	result, err := client.EmbedImage(context.Background(), "scan.png", "image/png", []byte{1, 2, 3})
	if err != nil {
		t.Fatalf("EmbedImage returned error: %v", err)
	}
	if len(result.Embedding) != 3 {
		t.Fatalf("expected 3 dimensions, got %d", len(result.Embedding))
	}
	if got["image_base64"] == "" {
		t.Fatalf("expected image_base64 request field")
	}
	if got["mime_type"] != "image/png" {
		t.Fatalf("expected image/png mime type, got %v", got["mime_type"])
	}
}

func TestEmbedImageDimensionMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(Result{Embedding: []float32{0.1, 0.2}})
	}))
	defer srv.Close()

	client := New(srv.URL, "test", 3, time.Second)
	if _, err := client.EmbedImage(context.Background(), "scan.png", "image/png", []byte{1}); err == nil {
		t.Fatalf("expected dimension mismatch error")
	}
}

func TestEmbedTextSendsRequestAndValidatesDimensions(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embed-text" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(Result{
			Embedding:  []float32{0.1, 0.2, 0.3},
			Model:      "test",
			Dimensions: 3,
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "test", 3, time.Second)
	result, err := client.EmbedText(context.Background(), "red product photo")
	if err != nil {
		t.Fatalf("EmbedText returned error: %v", err)
	}
	if len(result.Embedding) != 3 {
		t.Fatalf("expected 3 dimensions, got %d", len(result.Embedding))
	}
	if got["text"] != "red product photo" {
		t.Fatalf("expected text request field, got %v", got["text"])
	}
}
