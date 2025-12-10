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
)

const (
	userType   = "user"
	usersKind  = "users"
	domainType = "domain"

	membershipPerm         = "membership"
	llmChatCompletionsPerm = "llm_chat_completions_permission"
	llmCompletionsPerm     = "llm_completions_permission"
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
	permission := membershipPerm

	// Check for audit log endpoints
	switch {
	case strings.Contains(path, "/audit/") || strings.HasSuffix(path, "/audit"):
		permission = auditLogPerm
	case strings.Contains(path, "/v1/chat/completions"):
		// Check for LLM endpoints
		permission = llmChatCompletionsPerm
	case strings.Contains(path, "/v1/completions"):
		permission = llmCompletionsPerm
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
