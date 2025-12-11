package middleware

import (
	"context"
	"testing"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/policies"
)

type mockAuthz struct {
	authorizeFunc func(ctx context.Context, req authz.PolicyReq) error
}

func (m *mockAuthz) Authorize(ctx context.Context, req authz.PolicyReq) error {
	if m.authorizeFunc != nil {
		return m.authorizeFunc(ctx, req)
	}
	return nil
}

func (m *mockAuthz) AuthorizePAT(ctx context.Context, req authz.PatReq) error {
	return nil
}

type mockService struct {
	proxyRequestFunc func(ctx context.Context, session *authn.Session, path string) error
}

func (m *mockService) ProxyRequest(ctx context.Context, session *authn.Session, path string) error {
	if m.proxyRequestFunc != nil {
		return m.proxyRequestFunc(ctx, session, path)
	}
	return nil
}

func (m *mockService) Secure() string { return "" }
func (m *mockService) UpdateAttestationPolicy(ctx context.Context, session *authn.Session, policy []byte) error {
	return nil
}
func (m *mockService) GetAttestationPolicy(ctx context.Context, session *authn.Session) ([]byte, error) {
	return nil, nil
}

func TestAuthMiddleware_ProxyRequest(t *testing.T) {
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
			expectedPermission: membershipPerm,
		},
		{
			name:               "Audit Log Permission",
			path:               "/audit/logs",
			method:             "GET",
			expectedPermission: auditLogPerm,
		},
		{
			name:               "LLM Chat Completions",
			path:               "/v1/chat/completions",
			method:             "POST",
			expectedPermission: llmChatCompletionsPerm,
		},
		{
			name:               "LLM Chat (Ollama)",
			path:               "/api/chat",
			method:             "POST",
			expectedPermission: llmChatCompletionsPerm,
		},
		{
			name:               "LLM Completions",
			path:               "/v1/completions",
			method:             "POST",
			expectedPermission: llmCompletionsPerm,
		},
		{
			name:               "LLM Generate (Ollama)",
			path:               "/api/generate",
			method:             "POST",
			expectedPermission: llmCompletionsPerm,
		},
		{
			name:               "LLM Embeddings (OpenAI)",
			path:               "/v1/embeddings",
			method:             "POST",
			expectedPermission: llmEmbeddingsPerm,
		},
		{
			name:               "LLM Embeddings (Ollama)",
			path:               "/api/embeddings",
			method:             "POST",
			expectedPermission: llmEmbeddingsPerm,
		},
		{
			name:               "LLM Read Models (OpenAI)",
			path:               "/v1/models",
			method:             "GET",
			expectedPermission: llmReadPerm,
		},
		{
			name:               "LLM Tags (Ollama)",
			path:               "/api/tags",
			method:             "GET",
			expectedPermission: llmReadPerm,
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
			expectedPermission: llmReadPerm,
		},
		{
			name:               "Transcription",
			path:               "/v1/audio/transcriptions",
			method:             "POST",
			expectedPermission: llmTranscriptionPerm,
		},
		{
			name:               "Translation",
			path:               "/v1/audio/translations",
			method:             "POST",
			expectedPermission: llmTranslationPerm,
		},
		{
			name:               "Tokenizer (Utility)",
			path:               "/tokenize",
			method:             "POST",
			expectedPermission: llmUtilityPerm,
		},
		{
			name:               "Pooling",
			path:               "/pooling",
			method:             "POST",
			expectedPermission: llmPoolingPerm,
		},
		{
			name:               "Classification",
			path:               "/classify",
			method:             "POST",
			expectedPermission: llmClassificationPerm,
		},
		{
			name:               "Scoring",
			path:               "/score",
			method:             "POST",
			expectedPermission: llmScoringPerm,
		},
		{
			name:               "Rerank",
			path:               "/rerank",
			method:             "POST",
			expectedPermission: llmRerankPerm,
		},
		{
			name:               "Ollama PS",
			path:               "/api/ps",
			method:             "GET",
			expectedPermission: llmReadPerm,
		},
		{
			name:               "Ollama Version",
			path:               "/api/version",
			method:             "GET",
			expectedPermission: llmReadPerm,
		},
		{
			name:               "Ollama System",
			path:               "/api/system",
			method:             "GET",
			expectedPermission: llmReadPerm,
		},
		{
			name:               "Root Path (Health Check)",
			path:               "/",
			method:             "GET",
			expectedPermission: "", // Should bypass auth, so no authorize call expected. But wait, if unauthorized bypassed, my mock won't receive call.
			// However, my mock setup expects authorizeFunc to be called.
			// If I bypass authorize(), I need to handle that in the test expectation.
			// I'll assume for this test case that authorizeFunc is NOT called.
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{
				proxyRequestFunc: func(ctx context.Context, session *authn.Session, path string) error {
					return nil
				},
			}

			auth := &mockAuthz{
				authorizeFunc: func(ctx context.Context, req authz.PolicyReq) error {
					if tc.isSuperAdminCheck {
						if req.Permission != policies.AdminPermission {
							t.Errorf("expected permission %s, got %s", policies.AdminPermission, req.Permission)
						}
						// In checkSuperAdmin:
						// SubjectType: policies.UserType,
						// ObjectType:  policies.PlatformType,
						// Object:      policies.SuperMQObject,
						if req.ObjectType != policies.PlatformType {
							t.Errorf("expected object type %s, got %s", policies.PlatformType, req.ObjectType)
						}
						return nil
					}

					if req.Permission != tc.expectedPermission {
						t.Errorf("expected permission %s, got %s", tc.expectedPermission, req.Permission)
					}
					if req.ObjectType != domainType {
						t.Errorf("expected object type %s, got %s", domainType, req.ObjectType)
					}
					return nil
				},
			}

			middleware := AuthMiddleware(auth)(svc)

			// Inject method into context manually as the transport would
			ctx := context.WithValue(context.Background(), "method", tc.method)
			session := &authn.Session{DomainID: "domain1", UserID: "user1", DomainUserID: "domainUser1"}

			err := middleware.ProxyRequest(ctx, session, tc.path)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
