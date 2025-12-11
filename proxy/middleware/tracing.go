// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/router"
	"go.opentelemetry.io/otel/trace"
)

var _ proxy.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    proxy.Service
}

func NewTracingMiddleware(tracer trace.Tracer, svc proxy.Service) proxy.Service {
	return &tracingMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

func (t *tracingMiddleware) ProxyRequest(ctx context.Context, session *authn.Session, path string) error {
	ctx, span := t.tracer.Start(ctx, "ProxyRequest")
	defer span.End()

	return t.svc.ProxyRequest(ctx, session, path)
}

func (t *tracingMiddleware) Secure() string {
	return t.svc.Secure()
}

// GetAttestationPolicy implements proxy.Service.
func (t *tracingMiddleware) GetAttestationPolicy(ctx context.Context, session *authn.Session) ([]byte, error) {
	return t.svc.GetAttestationPolicy(ctx, session)
}

// UpdateAttestationPolicy implements proxy.Service.
func (t *tracingMiddleware) UpdateAttestationPolicy(ctx context.Context, session *authn.Session, policy []byte) error {
	return t.svc.UpdateAttestationPolicy(ctx, session, policy)
}

// CreateRoute implements proxy.Service.
func (t *tracingMiddleware) CreateRoute(ctx context.Context, session *authn.Session, route *router.RouteRule) error {
	ctx, span := t.tracer.Start(ctx, "CreateRoute")
	defer span.End()

	return t.svc.CreateRoute(ctx, session, route)
}

// UpdateRoute implements proxy.Service.
func (t *tracingMiddleware) UpdateRoute(ctx context.Context, session *authn.Session, route *router.RouteRule) error {
	ctx, span := t.tracer.Start(ctx, "UpdateRoute")
	defer span.End()

	return t.svc.UpdateRoute(ctx, session, route)
}

// DeleteRoute implements proxy.Service.
func (t *tracingMiddleware) DeleteRoute(ctx context.Context, session *authn.Session, name string) error {
	ctx, span := t.tracer.Start(ctx, "DeleteRoute")
	defer span.End()

	return t.svc.DeleteRoute(ctx, session, name)
}

// GetRoute implements proxy.Service.
func (t *tracingMiddleware) GetRoute(
	ctx context.Context, session *authn.Session, name string,
) (*router.RouteRule, error) {
	ctx, span := t.tracer.Start(ctx, "GetRoute")
	defer span.End()

	return t.svc.GetRoute(ctx, session, name)
}

// ListRoutes implements proxy.Service.
func (t *tracingMiddleware) ListRoutes(ctx context.Context, session *authn.Session) ([]router.RouteRule, error) {
	ctx, span := t.tracer.Start(ctx, "ListRoutes")
	defer span.End()

	return t.svc.ListRoutes(ctx, session)
}
