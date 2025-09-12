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
	"github.com/ultraviolet/cube/guardrails/api"
)

var _ guardrails.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     guardrails.Service
}

func (m *metricsMiddleware) ProcessChatCompletion(ctx context.Context, req *guardrails.ChatCompletionRequest) (*guardrails.ChatCompletionResponse, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "process_chat_completion"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	response, err := m.svc.ProcessChatCompletion(ctx, req)

	if err == nil && response != nil {
		if tokenCounter, ok := m.counter.(interface {
			With(...string) metrics.Counter
		}); ok {
			tokenCounter.With("metric", "tokens_total").Add(float64(response.Usage.TotalTokens))
			tokenCounter.With("metric", "tokens_prompt").Add(float64(response.Usage.PromptTokens))
			tokenCounter.With("metric", "tokens_completion").Add(float64(response.Usage.CompletionTokens))
		}

		if choiceCounter, ok := m.counter.(interface {
			With(...string) metrics.Counter
		}); ok {
			choiceCounter.With("metric", "response_choices").Add(float64(len(response.Choices)))
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
			With(...string) metrics.Histogram
		}); ok {
			sizeHistogram.With("metric", "config_size_bytes").Observe(float64(len(config)))
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
			With(...string) metrics.Histogram
		}); ok {
			sizeHistogram.With("metric", "config_yaml_size_bytes").Observe(float64(len(config)))
		}
	}

	return config, err
}

func NewMetricsMiddleware(svc guardrails.Service, counter metrics.Counter, latency metrics.Histogram) guardrails.Service {
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
	}(time.Now().UTC())

	return m.svc.Proxy()
}

func (m *metricsMiddleware) ProcessRequest(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "process_request").Add(1)
		m.latency.With("method", "process_request").Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.ProcessRequest(ctx, body, headers)
}

func (m *metricsMiddleware) ProcessResponse(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "process_response").Add(1)
		m.latency.With("method", "process_response").Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.ProcessResponse(ctx, body, headers)
}

func (m *metricsMiddleware) ValidateRequest(ctx context.Context, request interface{}) error {
	defer func(begin time.Time) {
		m.counter.With("method", "validate_request").Add(1)
		m.latency.With("method", "validate_request").Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.ValidateRequest(ctx, request)
}

func (m *metricsMiddleware) ValidateResponse(ctx context.Context, response interface{}) error {
	defer func(begin time.Time) {
		m.counter.With("method", "validate_response").Add(1)
		m.latency.With("method", "validate_response").Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.ValidateResponse(ctx, response)
}

func (m *metricsMiddleware) CreateFlow(ctx context.Context, flow guardrails.Flow) error {
	defer func(begin time.Time) {
		lvs := []string{"method", "create_flow"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.CreateFlow(ctx, flow)
}

func (m *metricsMiddleware) GetFlow(ctx context.Context, id string) (guardrails.Flow, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_flow"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.GetFlow(ctx, id)
}

func (m *metricsMiddleware) GetFlows(ctx context.Context, pm api.PageMetadata) ([]guardrails.Flow, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_flows"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	flows, err := m.svc.GetFlows(ctx, pm)

	if err == nil {
		if flowCounter, ok := m.counter.(interface {
			With(...string) metrics.Counter
		}); ok {
			flowCounter.With("metric", "flows_total").Add(float64(len(flows)))
		}
	}

	return flows, err
}

func (m *metricsMiddleware) UpdateFlow(ctx context.Context, flow guardrails.Flow) error {
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

func (m *metricsMiddleware) CreateKBFile(ctx context.Context, file guardrails.KBFile) error {
	defer func(begin time.Time) {
		lvs := []string{"method", "create_kb_file"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	if sizeHistogram, ok := m.latency.(interface {
		With(...string) metrics.Histogram
	}); ok {
		sizeHistogram.With("metric", "kb_file_size_bytes").Observe(float64(len(file.Content)))
	}

	return m.svc.CreateKBFile(ctx, file)
}

func (m *metricsMiddleware) GetKBFile(ctx context.Context, id string) (guardrails.KBFile, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_kb_file"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	return m.svc.GetKBFile(ctx, id)
}

func (m *metricsMiddleware) GetKBFiles(ctx context.Context, pm api.PageMetadata) ([]guardrails.KBFile, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "get_kb_files"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	files, err := m.svc.GetKBFiles(ctx, pm)

	if err == nil {
		if fileCounter, ok := m.counter.(interface {
			With(...string) metrics.Counter
		}); ok {
			fileCounter.With("metric", "kb_files_total").Add(float64(len(files)))
		}
	}

	return files, err
}

func (m *metricsMiddleware) UpdateKBFile(ctx context.Context, file guardrails.KBFile) error {
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

func (m *metricsMiddleware) SearchKBFiles(ctx context.Context, query string, categories []string, tags []string, limit int) ([]guardrails.KBFile, error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "search_kb_files"}
		m.counter.With(lvs...).Add(1)
		m.latency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now().UTC())

	files, err := m.svc.SearchKBFiles(ctx, query, categories, tags, limit)

	if err == nil {
		if searchCounter, ok := m.counter.(interface {
			With(...string) metrics.Counter
		}); ok {
			searchCounter.With("metric", "kb_search_queries").Add(1)
			searchCounter.With("metric", "kb_search_results").Add(float64(len(files)))
		}
	}

	if queryHistogram, ok := m.latency.(interface {
		With(...string) metrics.Histogram
	}); ok {
		queryHistogram.With("metric", "search_query_length").Observe(float64(len(query)))
		queryHistogram.With("metric", "search_categories_count").Observe(float64(len(categories)))
		queryHistogram.With("metric", "search_tags_count").Observe(float64(len(tags)))
	}

	return files, err
}
