// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"strings"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	"github.com/ultraviolet/cube/proxy"
)

const (
	userType   = "user"
	usersKind  = "users"
	domainType = "domain"

	membershipPerm         = "membership"
	llmChatCompletionsPerm = "llm_chat_completions_permission"
	llmCompletionsPerm     = "llm_completions_permission"
	auditLogReadPerm       = "audit_log_read_permission"
	auditLogExportPerm     = "audit_log_export_permission"
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

	// Check for LLM endpoints
	if strings.Contains(path, "/v1/chat/completions") {
		permission = llmChatCompletionsPerm
	} else if strings.Contains(path, "/v1/completions") {
		permission = llmCompletionsPerm
	}

	if err := am.authorize(ctx, session, session.DomainID, permission); err != nil {
		return err
	}

	return am.next.ProxyRequest(ctx, session, path)
}

func (am *authMiddleware) ListAuditLogs(
	ctx context.Context, session *authn.Session, query *proxy.AuditLogQuery,
) (map[string]any, error) {
	if err := am.authorize(ctx, session, session.DomainID, auditLogReadPerm); err != nil {
		return nil, err
	}

	return am.next.ListAuditLogs(ctx, session, query)
}

func (am *authMiddleware) ExportAuditLogs(
	ctx context.Context, session *authn.Session, query *proxy.AuditLogQuery,
) (body []byte, ctType string, err error) {
	if err := am.authorize(ctx, session, session.DomainID, auditLogExportPerm); err != nil {
		return nil, "", err
	}

	return am.next.ExportAuditLogs(ctx, session, query)
}

func (am *authMiddleware) Secure() string {
	return am.next.Secure()
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
