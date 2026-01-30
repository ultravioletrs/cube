// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/router"
)

var _ proxy.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    proxy.Service
}

func NewLoggingMiddleware(logger *slog.Logger, svc proxy.Service) proxy.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (l *loggingMiddleware) ProxyRequest(ctx context.Context, session *authn.Session, path string) (err error) {
	defer func(begin time.Time) {
		l.logger.Info("ProxyRequest", "path", path, "took", time.Since(begin), "error", err)
	}(time.Now())

	return l.svc.ProxyRequest(ctx, session, path)
}

func (l *loggingMiddleware) Secure() string {
	return l.svc.Secure()
}

// GetAttestationPolicy implements proxy.Service.
func (l *loggingMiddleware) GetAttestationPolicy(
	ctx context.Context, session *authn.Session,
) (policy []byte, err error) {
	defer func(begin time.Time) {
		l.logger.Info("GetAttestationPolicy", "took", time.Since(begin), "error", err)
	}(time.Now())

	return l.svc.GetAttestationPolicy(ctx, session)
}

// UpdateAttestationPolicy implements proxy.Service.
func (l *loggingMiddleware) UpdateAttestationPolicy(
	ctx context.Context, session *authn.Session, policy []byte,
) (err error) {
	defer func(begin time.Time) {
		l.logger.Info("UpdateAttestationPolicy", "took", time.Since(begin), "error", err)
	}(time.Now())

	return l.svc.UpdateAttestationPolicy(ctx, session, policy)
}

// CreateRoute implements proxy.Service.
func (l *loggingMiddleware) CreateRoute(
	ctx context.Context, session *authn.Session, route *router.RouteRule,
) (createdRoute *router.RouteRule, err error) {
	defer func(begin time.Time) {
		l.logger.Info("CreateRoute", "took", time.Since(begin), "error", err)
	}(time.Now())

	return l.svc.CreateRoute(ctx, session, route)
}

// UpdateRoute implements proxy.Service.
func (l *loggingMiddleware) UpdateRoute(
	ctx context.Context, session *authn.Session, route *router.RouteRule,
) (updatedRoute *router.RouteRule, err error) {
	defer func(begin time.Time) {
		l.logger.Info("UpdateRoute", "took", time.Since(begin), "error", err)
	}(time.Now())

	return l.svc.UpdateRoute(ctx, session, route)
}

// DeleteRoute implements proxy.Service.
func (l *loggingMiddleware) DeleteRoute(ctx context.Context, session *authn.Session, name string) (err error) {
	defer func(begin time.Time) {
		l.logger.Info("DeleteRoute", "took", time.Since(begin), "error", err)
	}(time.Now())

	return l.svc.DeleteRoute(ctx, session, name)
}

// GetRoute implements proxy.Service.
func (l *loggingMiddleware) GetRoute(
	ctx context.Context, session *authn.Session, name string,
) (rule *router.RouteRule, err error) {
	defer func(begin time.Time) {
		l.logger.Info("GetRoute", "took", time.Since(begin), "error", err)
	}(time.Now())

	return l.svc.GetRoute(ctx, session, name)
}

// ListRoutes implements proxy.Service.
func (l *loggingMiddleware) ListRoutes(
	ctx context.Context, session *authn.Session, offset, limit uint64,
) (rules []router.RouteRule, total uint64, err error) {
	defer func(begin time.Time) {
		l.logger.Info("ListRoutes", "offset", offset, "limit", limit, "took", time.Since(begin), "error", err)
	}(time.Now())

	return l.svc.ListRoutes(ctx, session, offset, limit)
}
