// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaConnectionChecksSelectedModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"models":[{"name":"llama3.2:3b"}]}`)
	}))
	defer server.Close()

	if err := testOllamaConnection(context.Background(), server.Client(), server.URL, "llama3.2:3b"); err != nil {
		t.Fatalf("expected model connection to succeed: %v", err)
	}
	if err := testOllamaConnection(context.Background(), server.Client(), server.URL, "missing"); err == nil || err.Error() != "Model is not available in Ollama" {
		t.Fatalf("expected missing model error, got %v", err)
	}
}

func TestAllowedExternalModelURL(t *testing.T) {
	for _, allowed := range []string{"https://api.openai.com", "https://api.anthropic.com"} {
		if !allowedExternalModelURL(allowed) {
			t.Fatalf("expected %q to be allowed", allowed)
		}
	}
	for _, blocked := range []string{
		"http://api.openai.com",
		"https://api.openai.com/v1",
		"https://example.com",
		"https://api.openai.com?key=secret",
	} {
		if allowedExternalModelURL(blocked) {
			t.Fatalf("expected %q to be blocked", blocked)
		}
	}
}

func TestOpenAIConnectionDoesNotExposeProviderResponse(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer secret-key" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error":"provider secret response"}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

	err := testOpenAICompatibleConnection(context.Background(), client, "https://api.openai.com", "gpt-4o-mini", "secret-key")
	if err == nil {
		t.Fatal("expected connection test to fail")
	}
	if strings.Contains(err.Error(), "provider secret response") || strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("connection error exposed sensitive provider data: %v", err)
	}
	if err.Error() != "Invalid API key or insufficient permissions" {
		t.Fatalf("unexpected connection error: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
