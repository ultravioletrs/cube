// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/ultravioletrs/cube/internal/atom"
	"github.com/ultravioletrs/cube/internal/cubeauth"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/middleware"
	"github.com/ultravioletrs/cube/proxy/mocks"
)

type authzChecker struct {
	requests []atom.CheckRequest
}

func (a *authzChecker) Check(_ context.Context, _ string, req *atom.CheckRequest) error {
	a.requests = append(a.requests, *req)

	return nil
}

func TestAuthMiddleware_ProxyRequest(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		path           string
		method         string
		expectedAction string
		expectCheck    bool
	}{
		{
			name:           "Membership Permission (Default)",
			path:           "/random/path",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:           "Audit Log Permission",
			path:           "/audit/logs",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:           "LLM Chat Completions",
			path:           "/v1/chat/completions",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "LLM Chat (Ollama)",
			path:           "/api/chat",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "LLM Completions",
			path:           "/v1/completions",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "LLM Generate (Ollama)",
			path:           "/api/generate",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "LLM Embeddings (OpenAI)",
			path:           "/v1/embeddings",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "LLM Embeddings (Ollama)",
			path:           "/api/embeddings",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "LLM Read Models (OpenAI)",
			path:           "/v1/models",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:           "LLM Tags (Ollama)",
			path:           "/api/tags",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:           "Manage Pull",
			path:           "/api/pull",
			method:         "POST",
			expectedAction: "manage",
			expectCheck:    true,
		},
		{
			name:           "Manage Delete",
			path:           "/api/delete",
			method:         "DELETE",
			expectedAction: "manage",
			expectCheck:    true,
		},
		{
			name:           "Manage Delete Model (OpenAI)",
			path:           "/v1/models/some-model",
			method:         "DELETE",
			expectedAction: "manage",
			expectCheck:    true,
		},
		{
			name:           "Guardrails Admin Configs Modify",
			path:           "/guardrails/configs",
			method:         "POST",
			expectedAction: "manage",
			expectCheck:    true,
		},
		{
			name:           "Guardrails Admin Reload",
			path:           "/guardrails/reload",
			method:         "POST",
			expectedAction: "manage",
			expectCheck:    true,
		},
		{
			name:           "Guardrails View Configs",
			path:           "/guardrails/configs",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:           "Guardrails View Versions",
			path:           "/guardrails/versions",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:           "Guardrails Modify Versions",
			path:           "/guardrails/versions",
			method:         "POST",
			expectedAction: "manage",
			expectCheck:    true,
		},
		{
			name:           "Regular Read Model (OpenAI)",
			path:           "/v1/models/some-model",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:           "Transcription",
			path:           "/v1/audio/transcriptions",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "Translation",
			path:           "/v1/audio/translations",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "Tokenizer (Utility)",
			path:           "/tokenize",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "Pooling",
			path:           "/pooling",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "Classification",
			path:           "/classify",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "Scoring",
			path:           "/score",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "Rerank",
			path:           "/rerank",
			method:         "POST",
			expectedAction: "execute",
			expectCheck:    true,
		},
		{
			name:           "Ollama PS",
			path:           "/api/ps",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:           "Ollama Version",
			path:           "/api/version",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:           "Ollama System",
			path:           "/api/system",
			method:         "GET",
			expectedAction: "read",
			expectCheck:    true,
		},
		{
			name:        "Root Path (Health Check)",
			path:        "/",
			method:      "GET",
			expectCheck: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := new(mocks.Service)
			svc.On("ProxyRequest", mock.Anything, mock.Anything, tc.path).Return(nil)

			auth := &authzChecker{}

			authMiddleware := middleware.AuthMiddleware(auth)(svc)

			// Inject method into context manually as the transport would
			ctx := context.WithValue(context.Background(), proxy.MethodContextKey, tc.method)
			session := &cubeauth.Session{TenantID: "tenant1", EntityID: "entity1", Token: "token"}

			err := authMiddleware.ProxyRequest(ctx, session, tc.path)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tc.expectCheck {
				if len(auth.requests) != 0 {
					t.Fatalf("expected no authorization check, got %d", len(auth.requests))
				}

				return
			}

			if len(auth.requests) == 0 {
				t.Fatal("expected authorization check")
			}

			if got := auth.requests[0].Action; got != tc.expectedAction {
				t.Fatalf("expected action %q, got %q", tc.expectedAction, got)
			}

			if got := auth.requests[0].ObjectKind; got != "tenant" {
				t.Fatalf("expected object kind tenant, got %q", got)
			}

			if got := auth.requests[0].ObjectID; got != "tenant1" {
				t.Fatalf("expected object ID tenant1, got %q", got)
			}
		})
	}
}
