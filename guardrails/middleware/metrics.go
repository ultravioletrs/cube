// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"context"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/ultraviolet/cube/guardrails"
)

var _ guardrails.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     guardrails.Service
}

func (m *metricsMiddleware) CreatePolicy(ctx context.Context, policy guardrails.Policy) error {
	defer func(begin time.Time) {
		m.counter.With("method", "create_policy").Add(1)
		m.latency.With("method", "create_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.CreatePolicy(ctx, policy)
}

func (m *metricsMiddleware) GetPolicy(ctx context.Context, id string) (guardrails.Policy, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "get_policy").Add(1)
		m.latency.With("method", "get_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.GetPolicy(ctx, id)
}

func (m *metricsMiddleware) ListPolicies(ctx context.Context, limit, offset int) ([]guardrails.Policy, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "list_policies").Add(1)
		m.latency.With("method", "list_policies").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.ListPolicies(ctx, limit, offset)
}

func (m *metricsMiddleware) UpdatePolicy(ctx context.Context, policy guardrails.Policy) error {
	defer func(begin time.Time) {
		m.counter.With("method", "update_policy").Add(1)
		m.latency.With("method", "update_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.UpdatePolicy(ctx, policy)
}

func (m *metricsMiddleware) DeletePolicy(ctx context.Context, id string) error {
	defer func(begin time.Time) {
		m.counter.With("method", "delete_policy").Add(1)
		m.latency.With("method", "delete_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.DeletePolicy(ctx, id)
}

func (m *metricsMiddleware) GetRestrictedTopics(ctx context.Context) ([]string, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "get_restricted_topics").Add(1)
		m.latency.With("method", "get_restricted_topics").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.GetRestrictedTopics(ctx)
}

func (m *metricsMiddleware) UpdateRestrictedTopics(ctx context.Context, topics []string) error {
	defer func(begin time.Time) {
		m.counter.With("method", "update_restricted_topics").Add(1)
		m.latency.With("method", "update_restricted_topics").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.UpdateRestrictedTopics(ctx, topics)
}

func (m *metricsMiddleware) AddRestrictedTopic(ctx context.Context, topic string) error {
	defer func(begin time.Time) {
		m.counter.With("method", "add_restricted_topic").Add(1)
		m.latency.With("method", "add_restricted_topic").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.AddRestrictedTopic(ctx, topic)
}

func (m *metricsMiddleware) RemoveRestrictedTopic(ctx context.Context, topic string) error {
	defer func(begin time.Time) {
		m.counter.With("method", "remove_restricted_topic").Add(1)
		m.latency.With("method", "remove_restricted_topic").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.RemoveRestrictedTopic(ctx, topic)
}

func (m *metricsMiddleware) GetBiasPatterns(ctx context.Context) (map[string][]guardrails.BiasPattern, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "get_bias_patterns").Add(1)
		m.latency.With("method", "get_bias_patterns").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.GetBiasPatterns(ctx)
}

func (m *metricsMiddleware) UpdateBiasPatterns(ctx context.Context, patterns map[string][]guardrails.BiasPattern) error {
	defer func(begin time.Time) {
		m.counter.With("method", "update_bias_patterns").Add(1)
		m.latency.With("method", "update_bias_patterns").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.UpdateBiasPatterns(ctx, patterns)
}

func (m *metricsMiddleware) GetFactualityConfig(ctx context.Context) (guardrails.FactualityConfig, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "get_factuality_config").Add(1)
		m.latency.With("method", "get_factuality_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.GetFactualityConfig(ctx)
}

func (m *metricsMiddleware) UpdateFactualityConfig(ctx context.Context, config guardrails.FactualityConfig) error {
	defer func(begin time.Time) {
		m.counter.With("method", "update_factuality_config").Add(1)
		m.latency.With("method", "update_factuality_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.UpdateFactualityConfig(ctx, config)
}

func (m *metricsMiddleware) GetAuditLogs(ctx context.Context, limit int) ([]guardrails.AuditLog, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "get_audit_logs").Add(1)
		m.latency.With("method", "get_audit_logs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.GetAuditLogs(ctx, limit)
}

func (m *metricsMiddleware) ExportConfig(ctx context.Context) ([]byte, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "export_config").Add(1)
		m.latency.With("method", "export_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.ExportConfig(ctx)
}

func (m *metricsMiddleware) ImportConfig(ctx context.Context, data []byte) error {
	defer func(begin time.Time) {
		m.counter.With("method", "import_config").Add(1)
		m.latency.With("method", "import_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.ImportConfig(ctx, data)
}

func NewMetricsMiddleware(counter metrics.Counter, latency metrics.Histogram, svc guardrails.Service) guardrails.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (m *metricsMiddleware) Proxy() *httputil.ReverseProxy {
	defer func(begin time.Time) {
		m.counter.With("method", "proxy").Add(1)
		m.latency.With("method", "proxy").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.Proxy()
}

func (m *metricsMiddleware) ProcessRequest(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "process_request").Add(1)
		m.latency.With("method", "process_request").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.ProcessRequest(ctx, body, headers)
}

func (m *metricsMiddleware) ProcessResponse(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "process_response").Add(1)
		m.latency.With("method", "process_response").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.ProcessResponse(ctx, body, headers)
}

func (m *metricsMiddleware) ValidateRequest(ctx context.Context, request interface{}) error {
	defer func(begin time.Time) {
		m.counter.With("method", "validate_request").Add(1)
		m.latency.With("method", "validate_request").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.ValidateRequest(ctx, request)
}

func (m *metricsMiddleware) ValidateResponse(ctx context.Context, response interface{}) error {
	defer func(begin time.Time) {
		m.counter.With("method", "validate_response").Add(1)
		m.latency.With("method", "validate_response").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.ValidateResponse(ctx, response)
}
