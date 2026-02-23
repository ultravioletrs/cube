// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"context"
	"testing"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	authzmocks "github.com/absmach/supermq/pkg/authz/mocks"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/stretchr/testify/mock"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/middleware"
	"github.com/ultravioletrs/cube/proxy/mocks"
)

func TestAuthMiddleware_ProxyRequest(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name               string
		path               string
		method             string
		expectedPermission string
		isSuperAdminCheck  bool
	}{
		{
			name:               "Membership Permission (Default)",
			path:               "/random/path",
			method:             "GET",
			expectedPermission: "membership",
		},
		{
			name:               "Audit Log Permission",
			path:               "/audit/logs",
			method:             "GET",
			expectedPermission: "audit_log_permission",
		},
		{
			name:               "LLM Chat Completions",
			path:               "/v1/chat/completions",
			method:             "POST",
			expectedPermission: "llm_chat_completions_permission",
		},
		{
			name:               "LLM Chat (Ollama)",
			path:               "/api/chat",
			method:             "POST",
			expectedPermission: "llm_chat_completions_permission",
		},
		{
			name:               "LLM Completions",
			path:               "/v1/completions",
			method:             "POST",
			expectedPermission: "llm_completions_permission",
		},
		{
			name:               "LLM Generate (Ollama)",
			path:               "/api/generate",
			method:             "POST",
			expectedPermission: "llm_completions_permission",
		},
		{
			name:               "LLM Embeddings (OpenAI)",
			path:               "/v1/embeddings",
			method:             "POST",
			expectedPermission: "llm_embeddings_permission",
		},
		{
			name:               "LLM Embeddings (Ollama)",
			path:               "/api/embeddings",
			method:             "POST",
			expectedPermission: "llm_embeddings_permission",
		},
		{
			name:               "LLM Read Models (OpenAI)",
			path:               "/v1/models",
			method:             "GET",
			expectedPermission: "llm_read_permission",
		},
		{
			name:               "LLM Tags (Ollama)",
			path:               "/api/tags",
			method:             "GET",
			expectedPermission: "llm_read_permission",
		},
		{
			name:              "SuperAdmin Pull",
			path:              "/api/pull",
			method:            "POST",
			isSuperAdminCheck: true,
		},
		{
			name:              "SuperAdmin Delete",
			path:              "/api/delete",
			method:            "DELETE",
			isSuperAdminCheck: true,
		},
		{
			name:              "SuperAdmin Delete Model (OpenAI)",
			path:              "/v1/models/some-model",
			method:            "DELETE",
			isSuperAdminCheck: true,
		},
		{
			name:               "Regular Read Model (OpenAI)",
			path:               "/v1/models/some-model",
			method:             "GET",
			expectedPermission: "llm_read_permission",
		},
		{
			name:               "Transcription",
			path:               "/v1/audio/transcriptions",
			method:             "POST",
			expectedPermission: "llm_transcription_permission",
		},
		{
			name:               "Translation",
			path:               "/v1/audio/translations",
			method:             "POST",
			expectedPermission: "llm_translation_permission",
		},
		{
			name:               "Tokenizer (Utility)",
			path:               "/tokenize",
			method:             "POST",
			expectedPermission: "llm_utility_permission",
		},
		{
			name:               "Pooling",
			path:               "/pooling",
			method:             "POST",
			expectedPermission: "llm_pooling_permission",
		},
		{
			name:               "Classification",
			path:               "/classify",
			method:             "POST",
			expectedPermission: "llm_classification_permission",
		},
		{
			name:               "Scoring",
			path:               "/score",
			method:             "POST",
			expectedPermission: "llm_scoring_permission",
		},
		{
			name:               "Rerank",
			path:               "/rerank",
			method:             "POST",
			expectedPermission: "llm_rerank_permission",
		},
		{
			name:               "Ollama PS",
			path:               "/api/ps",
			method:             "GET",
			expectedPermission: "llm_read_permission",
		},
		{
			name:               "Ollama Version",
			path:               "/api/version",
			method:             "GET",
			expectedPermission: "llm_read_permission",
		},
		{
			name:               "Ollama System",
			path:               "/api/system",
			method:             "GET",
			expectedPermission: "llm_read_permission",
		},
		{
			name:               "Root Path (Health Check)",
			path:               "/",
			method:             "GET",
			expectedPermission: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := new(mocks.Service)
			svc.On("ProxyRequest", mock.Anything, mock.Anything, tc.path).Return(nil)

			auth := new(authzmocks.Authorization)
			auth.On("Authorize", mock.Anything, mock.MatchedBy(func(req authz.PolicyReq) bool {
				if tc.isSuperAdminCheck {
					if req.Permission != policies.AdminPermission {
						return false
					}

					if req.ObjectType != policies.PlatformType {
						return false
					}

					return true
				}

				if req.Permission != tc.expectedPermission {
					return false
				}

				if req.ObjectType != "domain" {
					return false
				}

				return true
			})).Return(nil)

			authMiddleware := middleware.AuthMiddleware(auth)(svc)

			// Inject method into context manually as the transport would
			ctx := context.WithValue(context.Background(), proxy.MethodContextKey, tc.method)
			session := &authn.Session{DomainID: "domain1", UserID: "user1", DomainUserID: "domainUser1"}

			err := authMiddleware.ProxyRequest(ctx, session, tc.path)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthMiddleware_UpdateAttestationPolicy(t *testing.T) {
	t.Parallel()

	policy := []byte("policy")

	t.Run("domain admin allowed", func(t *testing.T) {
		t.Parallel()

		svc := new(mocks.Service)
		svc.On("UpdateAttestationPolicy", mock.Anything, mock.Anything, policy).Return(nil)

		auth := new(authzmocks.Authorization)
		auth.On("Authorize", mock.Anything, mock.MatchedBy(func(req authz.PolicyReq) bool {
			return req.Permission == policies.AdminPermission &&
				req.ObjectType == "domain" &&
				req.Object == "domain1" &&
				req.Domain == "domain1" &&
				req.SubjectType == "user" &&
				req.SubjectKind == "users" &&
				req.Subject == "domainUser1"
		})).Return(nil).Once()

		authMiddleware := middleware.AuthMiddleware(auth)(svc)
		session := &authn.Session{DomainID: "domain1", UserID: "user1", DomainUserID: "domainUser1"}

		err := authMiddleware.UpdateAttestationPolicy(context.Background(), session, policy)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		svc.AssertCalled(t, "UpdateAttestationPolicy", mock.Anything, session, policy)
		auth.AssertExpectations(t)
	})

	t.Run("non-admin denied", func(t *testing.T) {
		t.Parallel()

		svc := new(mocks.Service)
		auth := new(authzmocks.Authorization)

		auth.On("Authorize", mock.Anything, mock.MatchedBy(func(req authz.PolicyReq) bool {
			return req.Permission == policies.AdminPermission &&
				req.ObjectType == "domain" &&
				req.Object == "domain1"
		})).Return(svcerr.ErrAuthorization).Once()

		auth.On("Authorize", mock.Anything, mock.MatchedBy(func(req authz.PolicyReq) bool {
			return req.Permission == policies.AdminPermission &&
				req.ObjectType == policies.PlatformType &&
				req.Object == policies.SuperMQObject
		})).Return(svcerr.ErrAuthorization).Once()

		authMiddleware := middleware.AuthMiddleware(auth)(svc)
		session := &authn.Session{DomainID: "domain1", UserID: "user1", DomainUserID: "domainUser1"}

		err := authMiddleware.UpdateAttestationPolicy(context.Background(), session, policy)
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		svc.AssertNotCalled(t, "UpdateAttestationPolicy", mock.Anything, mock.Anything, mock.Anything)
		auth.AssertExpectations(t)
	})

	t.Run("super admin allowed", func(t *testing.T) {
		t.Parallel()

		svc := new(mocks.Service)
		svc.On("UpdateAttestationPolicy", mock.Anything, mock.Anything, policy).Return(nil)

		auth := new(authzmocks.Authorization)
		auth.On("Authorize", mock.Anything, mock.MatchedBy(func(req authz.PolicyReq) bool {
			return req.Permission == policies.AdminPermission &&
				req.ObjectType == "domain" &&
				req.Object == "domain1"
		})).Return(svcerr.ErrAuthorization).Once()

		auth.On("Authorize", mock.Anything, mock.MatchedBy(func(req authz.PolicyReq) bool {
			return req.Permission == policies.AdminPermission &&
				req.ObjectType == policies.PlatformType &&
				req.Object == policies.SuperMQObject
		})).Return(nil).Once()

		authMiddleware := middleware.AuthMiddleware(auth)(svc)
		session := &authn.Session{DomainID: "domain1", UserID: "user1", DomainUserID: "domainUser1"}

		err := authMiddleware.UpdateAttestationPolicy(context.Background(), session, policy)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		svc.AssertCalled(t, "UpdateAttestationPolicy", mock.Anything, session, policy)
		auth.AssertExpectations(t)
	})

	t.Run("missing domain ID", func(t *testing.T) {
		t.Parallel()

		svc := new(mocks.Service)
		auth := new(authzmocks.Authorization)

		authMiddleware := middleware.AuthMiddleware(auth)(svc)
		session := &authn.Session{DomainID: "", UserID: "user1", DomainUserID: "domainUser1"}

		err := authMiddleware.UpdateAttestationPolicy(context.Background(), session, policy)
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		svc.AssertNotCalled(t, "UpdateAttestationPolicy", mock.Anything, mock.Anything, mock.Anything)
		auth.AssertNotCalled(t, "Authorize", mock.Anything, mock.Anything)
	})
}
