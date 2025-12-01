// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/ultraviolet/cube/proxy"
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

func (t *tracingMiddleware) ProxyRequest(ctx context.Context, session authn.Session, domainID, path string) error {
	ctx, span := t.tracer.Start(ctx, "ProxyRequest")
	defer span.End()
	return t.svc.ProxyRequest(ctx, session, domainID, path)
}

func (t *tracingMiddleware) ListAuditLogs(ctx context.Context, session authn.Session, domainID string, query proxy.AuditLogQuery) (map[string]interface{}, error) {
	ctx, span := t.tracer.Start(ctx, "ListAuditLogs")
	defer span.End()
	return t.svc.ListAuditLogs(ctx, session, domainID, query)
}

func (t *tracingMiddleware) ExportAuditLogs(ctx context.Context, session authn.Session, domainID string, query proxy.AuditLogQuery) ([]byte, string, error) {
	ctx, span := t.tracer.Start(ctx, "ExportAuditLogs")
	defer span.End()
	return t.svc.ExportAuditLogs(ctx, session, domainID, query)
}

func (t *tracingMiddleware) Secure() string {
	return t.svc.Secure()
}
