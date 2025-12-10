// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/ultravioletrs/cube/proxy"
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
// GetAttestationPolicy implements proxy.Service.
func (t *tracingMiddleware) GetAttestationPolicy(ctx context.Context, session *authn.Session) ([]byte, error) {
	return t.svc.GetAttestationPolicy(ctx, session)
}

// UpdateAttestationPolicy implements proxy.Service.
func (t *tracingMiddleware) UpdateAttestationPolicy(ctx context.Context, session *authn.Session, policy []byte) error {
	return t.svc.UpdateAttestationPolicy(ctx, session, policy)
}
