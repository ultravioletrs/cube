// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"context"
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

func (m *metricsMiddleware) ProcessChatCompletion(
	ctx context.Context,
	req *guardrails.ChatCompletionRequest,
) (*guardrails.ChatCompletionResponse, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "process_chat_completion"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	response, err := m.svc.ProcessChatCompletion(ctx, req)

	if err == nil && response != nil {
		if tokenCounter, ok := m.counter.(interface {
			With(labelValues ...string) metrics.Counter
		}); ok {
			tokenCounter.With("method", "tokens_total").Add(float64(response.Usage.TotalTokens))
			tokenCounter.With("method", "tokens_prompt").Add(float64(response.Usage.PromptTokens))
			tokenCounter.With("method", "tokens_completion").Add(float64(response.Usage.CompletionTokens))
		}

		if choiceCounter, ok := m.counter.(interface {
			With(labelValues ...string) metrics.Counter
		}); ok {
			choiceCounter.With("method", "response_choices").Add(float64(len(response.Choices)))
		}
	}

	if err != nil {
		errorLvs := []string{"method", "process_chat_completion_error"}
		m.counter.With(errorLvs...).Add(1)
	}

	return response, err
}

func (m *metricsMiddleware) GetNeMoConfig(ctx context.Context) ([]byte, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_nemo_config"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	config, err := m.svc.GetNeMoConfig(ctx)
	if err == nil {
		if sizeHistogram, ok := m.latency.(interface {
			With(labelValues ...string) metrics.Histogram
		}); ok {
			sizeHistogram.With("method", "config_size_bytes").Observe(float64(len(config)))
		}
	}

	return config, err
}

func (m *metricsMiddleware) GetNeMoConfigYAML(ctx context.Context) ([]byte, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_nemo_config_yaml"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	config, err := m.svc.GetNeMoConfigYAML(ctx)
	if err == nil {
		if sizeHistogram, ok := m.latency.(interface {
			With(labelValues ...string) metrics.Histogram
		}); ok {
			sizeHistogram.With("method", "config_yaml_size_bytes").Observe(float64(len(config)))
		}
	}

	return config, err
}

func NewMetricsMiddleware(
	svc guardrails.Service,
	counter metrics.Counter,
	latency metrics.Histogram,
) guardrails.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (m *metricsMiddleware) Proxy() *httputil.ReverseProxy {
	defer func(begin time.Time) {
		lvs := []string{"method", "proxy"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.Proxy()
}

func (m *metricsMiddleware) CreateFlow(ctx context.Context, flow *guardrails.Flow) error {
	defer func(begin time.Time) {
		lvs := []string{"method", "create_flow"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.CreateFlow(ctx, flow)
}

func (m *metricsMiddleware) GetFlow(ctx context.Context, id string) (*guardrails.Flow, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_flow"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.GetFlow(ctx, id)
}

func (m *metricsMiddleware) GetFlows(
	ctx context.Context,
	pm *guardrails.PageMetadata,
) ([]*guardrails.Flow, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_flows"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.GetFlows(ctx, pm)
}

func (m *metricsMiddleware) UpdateFlow(ctx context.Context, flow *guardrails.Flow) error {
	defer func(begin time.Time) {
		lvs := []string{"method", "update_flow"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.UpdateFlow(ctx, flow)
}

func (m *metricsMiddleware) DeleteFlow(ctx context.Context, id string) error {
	defer func(begin time.Time) {
		lvs := []string{"method", "delete_flow"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.DeleteFlow(ctx, id)
}

func (m *metricsMiddleware) CreateKBFile(ctx context.Context, file *guardrails.KBFile) error {
	defer func(begin time.Time) {
		lvs := []string{"method", "create_kb_file"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	if sizeHistogram, ok := m.latency.(interface {
		With(labelValues ...string) metrics.Histogram
	}); ok {
		sizeHistogram.With("method", "kb_file_size_bytes").Observe(float64(len(file.Content)))
	}

	return m.svc.CreateKBFile(ctx, file)
}

func (m *metricsMiddleware) GetKBFile(ctx context.Context, id string) (*guardrails.KBFile, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_kb_file"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.GetKBFile(ctx, id)
}

func (m *metricsMiddleware) GetKBFiles(
	ctx context.Context,
	pm *guardrails.PageMetadata,
) ([]*guardrails.KBFile, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_kb_files"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.GetKBFiles(ctx, pm)
}

func (m *metricsMiddleware) UpdateKBFile(ctx context.Context, file *guardrails.KBFile) error {
	defer func(begin time.Time) {
		lvs := []string{"method", "update_kb_file"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.UpdateKBFile(ctx, file)
}

func (m *metricsMiddleware) DeleteKBFile(ctx context.Context, id string) error {
	defer func(begin time.Time) {
		lvs := []string{"method", "delete_kb_file"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.DeleteKBFile(ctx, id)
}

func (m *metricsMiddleware) SearchKBFiles(
	ctx context.Context,
	query string,
	categories, tags []string,
	limit int,
) ([]*guardrails.KBFile, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "search_kb_files"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.SearchKBFiles(ctx, query, categories, tags, limit)
}
