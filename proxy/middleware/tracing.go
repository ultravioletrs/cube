// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"context"

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

func (tm *tracingMiddleware) Identify(ctx context.Context, token string) error {
	ctx, span := tm.tracer.Start(ctx, "identify")
	defer span.End()

	return tm.svc.Identify(ctx, token)
}
