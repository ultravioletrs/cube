// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package guardrails

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/llm"
)

type fakeLLM struct {
	called bool
}

func (f *fakeLLM) StreamChat(_ context.Context, _ []llm.Message, out chan<- string) error {
	f.called = true
	defer close(out)
	out <- "allowed"
	return nil
}

func TestGuardedClientBlocksWithoutCallingInnerLLM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/guardrails/validate" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"decision": "BLOCK",
			"refusal":  "blocked by policy",
		})
	}))
	defer srv.Close()

	inner := &fakeLLM{}
	client := NewController(New(srv.URL)).Wrap(inner)

	out := make(chan string, 1)
	if err := client.StreamChat(context.Background(), []llm.Message{{Role: "user", Content: "bad"}}, out); err != nil {
		t.Fatalf("StreamChat returned error: %v", err)
	}
	if inner.called {
		t.Fatalf("expected inner LLM not to be called")
	}
	if got := <-out; got != "blocked by policy" {
		t.Fatalf("expected refusal, got %q", got)
	}
}

func TestControllerToggleIsSharedAcrossWrappedClients(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"decision": "BLOCK",
			"refusal":  "blocked by policy",
		})
	}))
	defer srv.Close()

	ctrl := NewController(New(srv.URL))
	ctrl.SetEnabled(false)

	inner := &fakeLLM{}
	client := ctrl.Wrap(inner)

	out := make(chan string, 1)
	if err := client.StreamChat(context.Background(), []llm.Message{{Role: "user", Content: "bad"}}, out); err != nil {
		t.Fatalf("StreamChat returned error: %v", err)
	}
	if !inner.called {
		t.Fatalf("expected inner LLM to be called when shared controller is disabled")
	}
	if got := <-out; got != "allowed" {
		t.Fatalf("expected inner response, got %q", got)
	}
}
