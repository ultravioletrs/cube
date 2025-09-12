// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"context"
	"log/slog"
	"net/http/httputil"
	"time"

	"github.com/ultraviolet/cube/guardrails"
)

var _ guardrails.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    guardrails.Service
}

func (l *loggingMiddleware) CreateFlow(ctx context.Context, flow *guardrails.Flow) error {
	defer func(begin time.Time) {
		l.logger.Debug("CreateFlow completed", "duration", time.Since(begin), "flow_id", flow.ID)
	}(time.Now())

	l.logger.Debug("Creating flow", "flow_name", flow.Name, "flow_type", flow.Type)

	return l.svc.CreateFlow(ctx, flow)
}

func (l *loggingMiddleware) GetFlow(ctx context.Context, id string) (*guardrails.Flow, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetFlow completed", "duration", time.Since(begin), "flow_id", id)
	}(time.Now())

	l.logger.Debug("Getting flow", "flow_id", id)

	return l.svc.GetFlow(ctx, id)
}

func (l *loggingMiddleware) GetFlows(
	ctx context.Context,
	pm *guardrails.PageMetadata,
) ([]*guardrails.Flow, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetFlows completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Getting all flows")

	return l.svc.GetFlows(ctx, pm)
}

func (l *loggingMiddleware) UpdateFlow(ctx context.Context, flow *guardrails.Flow) error {
	defer func(begin time.Time) {
		l.logger.Debug("UpdateFlow completed", "duration", time.Since(begin), "flow_id", flow.ID)
	}(time.Now())

	l.logger.Debug("Updating flow", "flow_id", flow.ID, "flow_name", flow.Name, "flow_type", flow.Type)

	return l.svc.UpdateFlow(ctx, flow)
}

func (l *loggingMiddleware) DeleteFlow(ctx context.Context, id string) error {
	defer func(begin time.Time) {
		l.logger.Debug("DeleteFlow completed", "duration", time.Since(begin), "flow_id", id)
	}(time.Now())

	l.logger.Debug("Deleting flow", "flow_id", id)

	return l.svc.DeleteFlow(ctx, id)
}

func (l *loggingMiddleware) CreateKBFile(ctx context.Context, file *guardrails.KBFile) error {
	defer func(begin time.Time) {
		l.logger.Debug("CreateKBFile completed", "duration", time.Since(begin), "kb_file_id", file.ID)
	}(time.Now())

	l.logger.Debug("Creating KB file", "kb_file_name", file.Name, "kb_file_type", file.Type, "category", file.Category)

	return l.svc.CreateKBFile(ctx, file)
}

func (l *loggingMiddleware) GetKBFile(ctx context.Context, id string) (*guardrails.KBFile, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetKBFile completed", "duration", time.Since(begin), "kb_file_id", id)
	}(time.Now())

	l.logger.Debug("Getting KB file", "kb_file_id", id)

	return l.svc.GetKBFile(ctx, id)
}

func (l *loggingMiddleware) GetKBFiles(
	ctx context.Context,
	pm *guardrails.PageMetadata,
) ([]*guardrails.KBFile, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetKBFiles completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Getting all KB files")

	return l.svc.GetKBFiles(ctx, pm)
}

func (l *loggingMiddleware) UpdateKBFile(ctx context.Context, file *guardrails.KBFile) error {
	defer func(begin time.Time) {
		l.logger.Debug("UpdateKBFile completed", "duration", time.Since(begin), "kb_file_id", file.ID)
	}(time.Now())

	l.logger.Debug("Updating KB file", "kb_file_id", file.ID, "kb_file_name", file.Name, "category", file.Category)

	return l.svc.UpdateKBFile(ctx, file)
}

func (l *loggingMiddleware) DeleteKBFile(ctx context.Context, id string) error {
	defer func(begin time.Time) {
		l.logger.Debug("DeleteKBFile completed", "duration", time.Since(begin), "kb_file_id", id)
	}(time.Now())

	l.logger.Debug("Deleting KB file", "kb_file_id", id)

	return l.svc.DeleteKBFile(ctx, id)
}

func (l *loggingMiddleware) SearchKBFiles(
	ctx context.Context,
	query string,
	categories, tags []string,
	limit int,
) ([]*guardrails.KBFile, error) {
	defer func(begin time.Time) {
		l.logger.Debug(
			"SearchKBFiles completed",
			"duration", time.Since(begin),
			"query", query,
			"category_count", len(categories),
			"tag_count", len(tags),
			"limit", limit,
		)
	}(time.Now())

	l.logger.Debug(
		"Searching KB files",
		"query", query,
		"category_count", len(categories),
		"tag_count", len(tags),
		"limit", limit,
	)

	return l.svc.SearchKBFiles(ctx, query, categories, tags, limit)
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

func (l *loggingMiddleware) ProcessChatCompletion(
	ctx context.Context,
	req *guardrails.ChatCompletionRequest,
) (*guardrails.ChatCompletionResponse, error) {
	defer func(begin time.Time) {
		l.logger.Debug("ProcessChatCompletion completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Processing chat completion", "model", req.Model)

	return l.svc.ProcessChatCompletion(ctx, req)
}

func (l *loggingMiddleware) GetNeMoConfig(ctx context.Context) ([]byte, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetNeMoConfig completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Getting NeMo configuration")

	return l.svc.GetNeMoConfig(ctx)
}

func (l *loggingMiddleware) GetNeMoConfigYAML(ctx context.Context) ([]byte, error) {
	defer func(begin time.Time) {
		l.logger.Debug("GetNeMoConfigYAML completed", "duration", time.Since(begin))
	}(time.Now())

	l.logger.Debug("Getting NeMo configuration as YAML")

	return l.svc.GetNeMoConfigYAML(ctx)
}
