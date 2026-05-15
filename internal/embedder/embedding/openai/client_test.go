// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbedSendsDimensionsWhenConfigured(t *testing.T) {
	var got map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"index":     0,
					"embedding": []float64{0.1, 0.2, 0.3},
				},
			},
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "text-embedding-3-small", "k", 3)
	_, err := client.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("Embed returned error: %v", err)
	}

	if _, ok := got["dimensions"]; !ok {
		t.Fatalf("expected dimensions field in request body")
	}
}

func TestEmbedDimensionMismatchReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"index":     0,
					"embedding": []float64{0.1, 0.2},
				},
			},
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "text-embedding-3-small", "k", 3)
	_, err := client.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatalf("expected dimension mismatch error")
	}
}
