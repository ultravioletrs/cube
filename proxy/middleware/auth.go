// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ultravioletrs/cube/internal/atom"
	"github.com/ultravioletrs/cube/internal/cubeauth"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/router"
)

const (
	actionRead    = "read"
	actionExecute = "execute"
	actionManage  = "manage"
)

var ErrAuthorization = errors.New("authorization denied")

type AuthzChecker interface {
	Check(ctx context.Context, callerToken string, req atom.CheckRequest) error
}

type authMiddleware struct {
	authz AuthzChecker
	next  proxy.Service
}

func AuthMiddleware(authz AuthzChecker) func(proxy.Service) proxy.Service {
	return func(next proxy.Service) proxy.Service {
		return &authMiddleware{
			authz: authz,
			next:  next,
		}
	}
}

func (am *authMiddleware) ProxyRequest(ctx context.Context, session *cubeauth.Session, path string) error {
	if err := am.checkAdminPaths(ctx, session, path); err != nil {
		return err
	}

	if path == "/" {
		return am.next.ProxyRequest(ctx, session, path)
	}

	if err := am.authorize(ctx, session, determineAction(path)); err != nil {
		return err
	}

	return am.next.ProxyRequest(ctx, session, path)
}

func (am *authMiddleware) Secure() string {
	return am.next.Secure()
}

func (am *authMiddleware) GetAttestationPolicy(ctx context.Context, session *cubeauth.Session) ([]byte, error) {
	if session.TenantID == "" {
		return nil, ErrAuthorization
	}
	if err := am.authorize(ctx, session, actionRead); err != nil {
		return nil, err
	}
	return am.next.GetAttestationPolicy(ctx, session)
}

func (am *authMiddleware) UpdateAttestationPolicy(
	ctx context.Context, session *cubeauth.Session, policy []byte,
) error {
	if err := am.authorize(ctx, session, actionManage); err != nil {
		return err
	}
	return am.next.UpdateAttestationPolicy(ctx, session, policy)
}

func (am *authMiddleware) CreateRoute(
	ctx context.Context, session *cubeauth.Session, route *router.RouteRule,
) (*router.RouteRule, error) {
	if err := am.authorize(ctx, session, actionManage); err != nil {
		return nil, err
	}
	return am.next.CreateRoute(ctx, session, route)
}

func (am *authMiddleware) UpdateRoute(
	ctx context.Context, session *cubeauth.Session, name string, route *router.RouteRule,
) (*router.RouteRule, error) {
	if err := am.authorize(ctx, session, actionManage); err != nil {
		return nil, err
	}
	return am.next.UpdateRoute(ctx, session, name, route)
}

func (am *authMiddleware) DeleteRoute(ctx context.Context, session *cubeauth.Session, name string) error {
	if err := am.authorize(ctx, session, actionManage); err != nil {
		return err
	}
	return am.next.DeleteRoute(ctx, session, name)
}

func (am *authMiddleware) GetRoute(
	ctx context.Context, session *cubeauth.Session, name string,
) (*router.RouteRule, error) {
	if err := am.authorize(ctx, session, actionRead); err != nil {
		return nil, err
	}
	return am.next.GetRoute(ctx, session, name)
}

func (am *authMiddleware) ListRoutes(
	ctx context.Context, session *cubeauth.Session, offset, limit uint64,
) ([]router.RouteRule, uint64, error) {
	if err := am.authorize(ctx, session, actionRead); err != nil {
		return nil, 0, err
	}
	return am.next.ListRoutes(ctx, session, offset, limit)
}

func (am *authMiddleware) authorize(ctx context.Context, session *cubeauth.Session, action string) error {
	if am.authz == nil {
		return ErrAuthorization
	}
	req := atom.CheckRequest{
		SubjectID: session.EntityID,
		Action:    action,
		Context: map[string]string{
			"cube_service": "proxy",
		},
	}
	if session.TenantID != "" {
		req.ObjectKind = "tenant"
		req.ObjectID = session.TenantID
	}
	if err := am.authz.Check(ctx, session.Token, req); err != nil {
		return err
	}
	return nil
}

func (am *authMiddleware) checkAdminPaths(ctx context.Context, session *cubeauth.Session, path string) error {
	adminPaths := []string{
		"/api/pull",
		"/api/push",
		"/api/create",
		"/api/copy",
		"/api/delete",
	}
	for _, p := range adminPaths {
		if strings.Contains(path, p) {
			return am.authorize(ctx, session, actionManage)
		}
	}

	guardrailsAdminPaths := []string{
		"/guardrails/configs",
		"/guardrails/versions",
		"/guardrails/reload",
	}
	for _, p := range guardrailsAdminPaths {
		if strings.Contains(path, p) {
			if ctx.Value(proxy.MethodContextKey) != http.MethodGet || strings.Contains(path, "/guardrails/reload") {
				return am.authorize(ctx, session, actionManage)
			}
		}
	}

	if strings.Contains(path, "/v1/models/") && ctx.Value(proxy.MethodContextKey) == http.MethodDelete {
		return am.authorize(ctx, session, actionManage)
	}
	return nil
}

func determineAction(path string) string {
	switch {
	case strings.Contains(path, "/audit/") || strings.HasSuffix(path, "/audit"):
		return actionRead
	case strings.Contains(path, "/v1/models") || strings.Contains(path, "/api/tags") ||
		strings.Contains(path, "/api/show") || strings.Contains(path, "/api/ps") ||
		strings.Contains(path, "/api/version") || strings.Contains(path, "/api/system"):
		return actionRead
	case strings.Contains(path, "/v1/chat/completions") ||
		strings.Contains(path, "/api/chat") ||
		strings.Contains(path, "/v1/completions") ||
		strings.Contains(path, "/api/generate") ||
		strings.Contains(path, "/v1/embeddings") ||
		strings.Contains(path, "/api/embeddings") ||
		strings.Contains(path, "/v1/audio/transcriptions") ||
		strings.Contains(path, "/v1/audio/translations") ||
		strings.Contains(path, "/tokenize") ||
		strings.Contains(path, "/detokenize") ||
		strings.Contains(path, "/pooling") ||
		strings.Contains(path, "/classify") ||
		strings.Contains(path, "/score") ||
		strings.Contains(path, "/rerank"):
		return actionExecute
	}
	return actionRead
}
