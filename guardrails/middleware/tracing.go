// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"context"
	"net/http"
	"net/http/httputil"

	"github.com/ultraviolet/cube/guardrails"
	"go.opentelemetry.io/otel/trace"
)

var _ guardrails.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    guardrails.Service
}

func (t *tracingMiddleware) CreatePolicy(ctx context.Context, policy guardrails.Policy) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.CreatePolicy")
	defer span.End()

	return t.svc.CreatePolicy(ctx, policy)
}

func (t *tracingMiddleware) GetPolicy(ctx context.Context, id string) (guardrails.Policy, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetPolicy")
	defer span.End()

	return t.svc.GetPolicy(ctx, id)
}

func (t *tracingMiddleware) ListPolicies(ctx context.Context, limit, offset int) ([]guardrails.Policy, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.ListPolicies")
	defer span.End()

	return t.svc.ListPolicies(ctx, limit, offset)
}

func (t *tracingMiddleware) UpdatePolicy(ctx context.Context, policy guardrails.Policy) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.UpdatePolicy")
	defer span.End()

	return t.svc.UpdatePolicy(ctx, policy)
}

func (t *tracingMiddleware) DeletePolicy(ctx context.Context, id string) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.DeletePolicy")
	defer span.End()

	return t.svc.DeletePolicy(ctx, id)
}

func (t *tracingMiddleware) GetRestrictedTopics(ctx context.Context) ([]string, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetRestrictedTopics")
	defer span.End()

	return t.svc.GetRestrictedTopics(ctx)
}

func (t *tracingMiddleware) UpdateRestrictedTopics(ctx context.Context, topics []string) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.UpdateRestrictedTopics")
	defer span.End()

	return t.svc.UpdateRestrictedTopics(ctx, topics)
}

func (t *tracingMiddleware) AddRestrictedTopic(ctx context.Context, topic string) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.AddRestrictedTopic")
	defer span.End()

	return t.svc.AddRestrictedTopic(ctx, topic)
}

func (t *tracingMiddleware) RemoveRestrictedTopic(ctx context.Context, topic string) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.RemoveRestrictedTopic")
	defer span.End()

	return t.svc.RemoveRestrictedTopic(ctx, topic)
}

func (t *tracingMiddleware) GetBiasPatterns(ctx context.Context) (map[string][]guardrails.BiasPattern, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetBiasPatterns")
	defer span.End()

	return t.svc.GetBiasPatterns(ctx)
}

func (t *tracingMiddleware) UpdateBiasPatterns(ctx context.Context, patterns map[string][]guardrails.BiasPattern) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.UpdateBiasPatterns")
	defer span.End()

	return t.svc.UpdateBiasPatterns(ctx, patterns)
}

func (t *tracingMiddleware) GetFactualityConfig(ctx context.Context) (guardrails.FactualityConfig, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetFactualityConfig")
	defer span.End()

	return t.svc.GetFactualityConfig(ctx)
}

func (t *tracingMiddleware) UpdateFactualityConfig(ctx context.Context, config guardrails.FactualityConfig) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.UpdateFactualityConfig")
	defer span.End()

	return t.svc.UpdateFactualityConfig(ctx, config)
}

func (t *tracingMiddleware) GetAuditLogs(ctx context.Context, limit int) ([]guardrails.AuditLog, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetAuditLogs")
	defer span.End()

	return t.svc.GetAuditLogs(ctx, limit)
}

func (t *tracingMiddleware) ExportConfig(ctx context.Context) ([]byte, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.ExportConfig")
	defer span.End()

	return t.svc.ExportConfig(ctx)
}

func (t *tracingMiddleware) ImportConfig(ctx context.Context, data []byte) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.ImportConfig")
	defer span.End()

	return t.svc.ImportConfig(ctx, data)
}

func NewTracingMiddleware(tracer trace.Tracer, svc guardrails.Service) guardrails.Service {
	return &tracingMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

func (t *tracingMiddleware) Proxy() *httputil.ReverseProxy {
	return t.svc.Proxy()
}

func (t *tracingMiddleware) ProcessRequest(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.ProcessRequest")
	defer span.End()

	return t.svc.ProcessRequest(ctx, body, headers)
}

func (t *tracingMiddleware) ProcessResponse(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.ProcessResponse")
	defer span.End()

	return t.svc.ProcessResponse(ctx, body, headers)
}

func (t *tracingMiddleware) ValidateRequest(ctx context.Context, request interface{}) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.ValidateRequest")
	defer span.End()

	return t.svc.ValidateRequest(ctx, request)
}

func (t *tracingMiddleware) ValidateResponse(ctx context.Context, response interface{}) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.ValidateResponse")
	defer span.End()

	return t.svc.ValidateResponse(ctx, response)
}
