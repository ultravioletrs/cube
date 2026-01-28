// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"strings"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/router"
)

const (
	userType   = "user"
	usersKind  = "users"
	domainType = "domain"

	membershipPerm         = "membership"
	llmChatCompletionsPerm = "llm_chat_completions_permission"
	llmCompletionsPerm     = "llm_completions_permission"
	llmEmbeddingsPerm      = "llm_embeddings_permission"
	llmReadPerm            = "llm_read_permission"
	llmTranscriptionPerm   = "llm_transcription_permission"
	llmTranslationPerm     = "llm_translation_permission"
	llmUtilityPerm         = "llm_utility_permission"
	llmPoolingPerm         = "llm_pooling_permission"
	llmClassificationPerm  = "llm_classification_permission"
	llmScoringPerm         = "llm_scoring_permission"
	llmRerankPerm          = "llm_rerank_permission"
	auditLogPerm           = "audit_log_permission"
)

type authMiddleware struct {
	authz authz.Authorization
	next  proxy.Service
}

// AuthMiddleware adds authorization checks to the service.
func AuthMiddleware(auth authz.Authorization) func(proxy.Service) proxy.Service {
	return func(next proxy.Service) proxy.Service {
		return &authMiddleware{
			authz: auth,
			next:  next,
		}
	}
}

func (am *authMiddleware) ProxyRequest(ctx context.Context, session *authn.Session, path string) error {
	superAdminPaths := []string{
		"/api/pull",
		"/api/push",
		"/api/create",
		"/api/copy",
		"/api/delete",
		// Guardrails admin endpoints
		"/guardrails/configs",
		"/guardrails/versions",
		"/guardrails/reload",
	}

	permission := membershipPerm

	// Check for audit log endpoints
	switch {
	case strings.Contains(path, "/audit/") || strings.HasSuffix(path, "/audit"):
		permission = auditLogPerm
	case strings.Contains(path, "/v1/chat/completions") || strings.Contains(path, "/api/chat"):
		// Check for LLM endpoints
		permission = llmChatCompletionsPerm
	case strings.Contains(path, "/v1/completions") || strings.Contains(path, "/api/generate"):
		permission = llmCompletionsPerm
	case strings.Contains(path, "/v1/embeddings") || strings.Contains(path, "/api/embeddings"):
		permission = llmEmbeddingsPerm
	case strings.Contains(path, "/v1/models") || strings.Contains(path, "/api/tags") ||
		strings.Contains(path, "/api/show"):
		permission = llmReadPerm
	case strings.Contains(path, "/api/ps") || strings.Contains(path, "/api/version") ||
		strings.Contains(path, "/api/system"):
		permission = llmReadPerm
	case strings.Contains(path, "/v1/audio/transcriptions"):
		permission = llmTranscriptionPerm
	case strings.Contains(path, "/v1/audio/translations"):
		permission = llmTranslationPerm
	case strings.Contains(path, "/tokenize") || strings.Contains(path, "/detokenize"):
		permission = llmUtilityPerm
	case strings.Contains(path, "/pooling"):
		permission = llmPoolingPerm
	case strings.Contains(path, "/classify"):
		permission = llmClassificationPerm
	case strings.Contains(path, "/score"):
		permission = llmScoringPerm
	case strings.Contains(path, "/rerank"):
		permission = llmRerankPerm
	}

	for _, p := range superAdminPaths {
		if strings.Contains(path, p) {
			return am.checkSuperAdmin(ctx, session.UserID)
		}
	}

	// OpenAI/vLLM delete model check
	if strings.Contains(path, "/v1/models/") && ctx.Value(proxy.MethodContextKey) == "DELETE" {
		return am.checkSuperAdmin(ctx, session.UserID)
	}

	// Health check / Connection check - if session is valid (which it is if we are here), allow root path
	if path == "/" {
		return am.next.ProxyRequest(ctx, session, path)
	}

	if err := am.authorize(ctx, session, session.DomainID, permission); err != nil {
		return err
	}

	return am.next.ProxyRequest(ctx, session, path)
}

func (am *authMiddleware) Secure() string {
	return am.next.Secure()
}

// GetAttestationPolicy implements proxy.Service.
// GetAttestationPolicy implements proxy.Service.
func (am *authMiddleware) GetAttestationPolicy(ctx context.Context, session *authn.Session) ([]byte, error) {
	if session.DomainID == "" {
		return nil, svcerr.ErrAuthorization
	}

	return am.next.GetAttestationPolicy(ctx, session)
}

// UpdateAttestationPolicy implements proxy.Service.
func (am *authMiddleware) UpdateAttestationPolicy(ctx context.Context, session *authn.Session, policy []byte) error {
	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		return err
	}

	return am.next.UpdateAttestationPolicy(ctx, session, policy)
}

// CreateRoute implements proxy.Service.
func (am *authMiddleware) CreateRoute(ctx context.Context, session *authn.Session, route *router.RouteRule) error {
	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		return err
	}

	return am.next.CreateRoute(ctx, session, route)
}

// UpdateRoute implements proxy.Service.
func (am *authMiddleware) UpdateRoute(ctx context.Context, session *authn.Session, route *router.RouteRule) error {
	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		return err
	}

	return am.next.UpdateRoute(ctx, session, route)
}

// DeleteRoute implements proxy.Service.
func (am *authMiddleware) DeleteRoute(ctx context.Context, session *authn.Session, name string) error {
	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		return err
	}

	return am.next.DeleteRoute(ctx, session, name)
}

// GetRoute implements proxy.Service.
func (am *authMiddleware) GetRoute(
	ctx context.Context, session *authn.Session, name string,
) (*router.RouteRule, error) {
	// Routes are considered administrative - require super admin
	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		return nil, err
	}

	return am.next.GetRoute(ctx, session, name)
}

// ListRoutes implements proxy.Service.
func (am *authMiddleware) ListRoutes(ctx context.Context, session *authn.Session) ([]router.RouteRule, error) {
	// Routes are considered administrative - require super admin
	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		return nil, err
	}

	return am.next.ListRoutes(ctx, session)
}

func (am *authMiddleware) checkSuperAdmin(ctx context.Context, adminID string) error {
	if err := am.authz.Authorize(ctx, authz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     adminID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.SuperMQObject,
	}); err != nil {
		return err
	}

	return nil
}

func (am *authMiddleware) authorize(ctx context.Context, session *authn.Session, domainID, permission string) error {
	req := authz.PolicyReq{
		Domain:      domainID,
		SubjectType: userType,
		SubjectKind: usersKind,
		Subject:     session.DomainUserID,
		Permission:  permission,
		ObjectType:  domainType,
		Object:      domainID,
	}

	return am.authz.Authorize(ctx, req)
}
