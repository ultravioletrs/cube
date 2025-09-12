// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/ultraviolet/cube/guardrails"
)

var _ guardrails.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    guardrails.Service
}

func (l *loggingMiddleware) CreatePolicy(ctx context.Context, policy guardrails.Policy) error {
	defer func(begin time.Time) {
		l.logger.Debug("CreatePolicy completed", "duration", time.Since(begin), "policy_id", policy.ID)
	}(time.Now())

	l.logger.Debug("Creating policy", "policy_name", policy.Name)
	return l.svc.CreatePolicy(ctx, policy)
}

func (l *loggingMiddleware) GetPolicy(ctx context.Context, id string) (guardrails.Policy, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetPolicy completed", "duration", time.Since(begin), "policy_id", id)
	}(time.Now())

	l.logger.Debug("Getting policy", "policy_id", id)
	return l.svc.GetPolicy(ctx, id)
}

func (l *loggingMiddleware) ListPolicies(ctx context.Context, limit, offset int) ([]guardrails.Policy, error) {
	defer func(begin time.Time) {
		l.logger.Debug("ListPolicies completed", "duration", time.Since(begin), "limit", limit, "offset", offset)
	}(time.Now())

	l.logger.Debug("Listing policies", "limit", limit, "offset", offset)
	return l.svc.ListPolicies(ctx, limit, offset)
}

func (l *loggingMiddleware) UpdatePolicy(ctx context.Context, policy guardrails.Policy) error {
	defer func(begin time.Time) {
		l.logger.Debug("UpdatePolicy completed", "duration", time.Since(begin), "policy_id", policy.ID)
	}(time.Now())

	l.logger.Debug("Updating policy", "policy_id", policy.ID, "policy_name", policy.Name)
	return l.svc.UpdatePolicy(ctx, policy)
}

func (l *loggingMiddleware) DeletePolicy(ctx context.Context, id string) error {
	defer func(begin time.Time) {
		l.logger.Debug("DeletePolicy completed", "duration", time.Since(begin), "policy_id", id)
	}(time.Now())

	l.logger.Debug("Deleting policy", "policy_id", id)
	return l.svc.DeletePolicy(ctx, id)
}

func (l *loggingMiddleware) GetRestrictedTopics(ctx context.Context) ([]string, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetRestrictedTopics completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Getting restricted topics")
	return l.svc.GetRestrictedTopics(ctx)
}

func (l *loggingMiddleware) UpdateRestrictedTopics(ctx context.Context, topics []string) error {
	defer func(begin time.Time) {
		l.logger.Debug("UpdateRestrictedTopics completed", "duration", time.Since(begin), "topic_count", len(topics))
	}(time.Now())

	l.logger.Debug("Updating restricted topics", "topic_count", len(topics))
	return l.svc.UpdateRestrictedTopics(ctx, topics)
}

func (l *loggingMiddleware) AddRestrictedTopic(ctx context.Context, topic string) error {
	defer func(begin time.Time) {
		l.logger.Debug("AddRestrictedTopic completed", "duration", time.Since(begin), "topic", topic)
	}(time.Now())

	l.logger.Debug("Adding restricted topic", "topic", topic)
	return l.svc.AddRestrictedTopic(ctx, topic)
}

func (l *loggingMiddleware) RemoveRestrictedTopic(ctx context.Context, topic string) error {
	defer func(begin time.Time) {
		l.logger.Debug("RemoveRestrictedTopic completed", "duration", time.Since(begin), "topic", topic)
	}(time.Now())

	l.logger.Debug("Removing restricted topic", "topic", topic)
	return l.svc.RemoveRestrictedTopic(ctx, topic)
}

func (l *loggingMiddleware) GetBiasPatterns(ctx context.Context) (map[string][]guardrails.BiasPattern, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetBiasPatterns completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Getting bias patterns")
	return l.svc.GetBiasPatterns(ctx)
}

func (l *loggingMiddleware) UpdateBiasPatterns(ctx context.Context, patterns map[string][]guardrails.BiasPattern) error {
	defer func(begin time.Time) {
		l.logger.Debug("UpdateBiasPatterns completed", "duration", time.Since(begin), "pattern_categories", len(patterns))
	}(time.Now())

	l.logger.Debug("Updating bias patterns", "pattern_categories", len(patterns))
	return l.svc.UpdateBiasPatterns(ctx, patterns)
}

func (l *loggingMiddleware) GetFactualityConfig(ctx context.Context) (guardrails.FactualityConfig, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetFactualityConfig completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Getting factuality config")
	return l.svc.GetFactualityConfig(ctx)
}

func (l *loggingMiddleware) UpdateFactualityConfig(ctx context.Context, config guardrails.FactualityConfig) error {
	defer func(begin time.Time) {
		l.logger.Debug("UpdateFactualityConfig completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Updating factuality config", "confidence_threshold", config.ConfidenceThreshold)
	return l.svc.UpdateFactualityConfig(ctx, config)
}

func (l *loggingMiddleware) GetAuditLogs(ctx context.Context, limit int) ([]guardrails.AuditLog, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetAuditLogs completed", "duration", time.Since(begin), "limit", limit)
	}(time.Now())

	l.logger.Debug("Getting audit logs", "limit", limit)
	return l.svc.GetAuditLogs(ctx, limit)
}

func (l *loggingMiddleware) ExportConfig(ctx context.Context) ([]byte, error) {
	defer func(begin time.Time) {
		l.logger.Debug("ExportConfig completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Exporting config")
	return l.svc.ExportConfig(ctx)
}

func (l *loggingMiddleware) ImportConfig(ctx context.Context, data []byte) error {
	defer func(begin time.Time) {
		l.logger.Debug("ImportConfig completed", "duration", time.Since(begin), "data_size", len(data))
	}(time.Now())

	l.logger.Debug("Importing config", "data_size", len(data))
	return l.svc.ImportConfig(ctx, data)
}

func NewLoggingMiddleware(logger *slog.Logger, svc guardrails.Service) guardrails.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (l *loggingMiddleware) Proxy() *httputil.ReverseProxy {
	l.logger.Info("Guardrails Proxy initialized", "service", "loggingMiddleware")
	return l.svc.Proxy()
}

func (l *loggingMiddleware) ProcessRequest(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	defer func(begin time.Time) {
		l.logger.Debug("ProcessRequest completed",
			"duration", time.Since(begin),
			"body_size", len(body))
	}(time.Now())

	l.logger.Debug("Processing request through guardrails", "body_size", len(body))
	return l.svc.ProcessRequest(ctx, body, headers)
}

func (l *loggingMiddleware) ProcessResponse(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	defer func(begin time.Time) {
		l.logger.Debug("ProcessResponse completed",
			"duration", time.Since(begin),
			"body_size", len(body))
	}(time.Now())

	l.logger.Debug("Processing response through guardrails", "body_size", len(body))
	return l.svc.ProcessResponse(ctx, body, headers)
}

func (l *loggingMiddleware) ValidateRequest(ctx context.Context, request interface{}) error {
	defer func(begin time.Time) {
		l.logger.Debug("ValidateRequest completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Validating request")
	return l.svc.ValidateRequest(ctx, request)
}

func (l *loggingMiddleware) ValidateResponse(ctx context.Context, response interface{}) error {
	defer func(begin time.Time) {
		l.logger.Debug("ValidateResponse completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Validating response")
	return l.svc.ValidateResponse(ctx, response)
}
