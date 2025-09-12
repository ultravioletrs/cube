// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"context"
	"net/http/httputil"

	"github.com/ultraviolet/cube/guardrails"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ guardrails.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    guardrails.Service
}

func (t *tracingMiddleware) ProcessChatCompletion(
	ctx context.Context,
	req *guardrails.ChatCompletionRequest,
) (*guardrails.ChatCompletionResponse, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.ProcessChatCompletion")
	defer span.End()

	// Add attributes to the span for better observability
	span.SetAttributes(
		attribute.String("guardrails.model", req.Model),
		attribute.Int("guardrails.message_count", len(req.Messages)),
		attribute.Float64("guardrails.temperature", req.Temperature),
		attribute.Int("guardrails.max_tokens", req.MaxTokens),
		attribute.Bool("guardrails.stream", req.Stream),
	)

	response, err := t.svc.ProcessChatCompletion(ctx, req)

	// Add response attributes if successful
	if err == nil && response != nil {
		span.SetAttributes(
			attribute.String("guardrails.response_id", response.ID),
			attribute.String("guardrails.response_model", response.Model),
			attribute.Int("guardrails.choice_count", len(response.Choices)),
			attribute.Int("guardrails.prompt_tokens", response.Usage.PromptTokens),
			attribute.Int("guardrails.completion_tokens", response.Usage.CompletionTokens),
			attribute.Int("guardrails.total_tokens", response.Usage.TotalTokens),
		)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return response, err
}

func (t *tracingMiddleware) GetNeMoConfig(ctx context.Context) ([]byte, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetNeMoConfig")
	defer span.End()

	config, err := t.svc.GetNeMoConfig(ctx)
	if err == nil {
		span.SetAttributes(attribute.Int("guardrails.config_size", len(config)))
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return config, err
}

func (t *tracingMiddleware) GetNeMoConfigYAML(ctx context.Context) ([]byte, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetNeMoConfigYAML")
	defer span.End()

	config, err := t.svc.GetNeMoConfigYAML(ctx)
	if err == nil {
		span.SetAttributes(
			attribute.Int("guardrails.config_size", len(config)),
			attribute.String("guardrails.config_format", "yaml"),
		)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return config, err
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

func (t *tracingMiddleware) CreateFlow(ctx context.Context, flow *guardrails.Flow) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.CreateFlow")
	defer span.End()

	// Add attributes to the span
	span.SetAttributes(
		attribute.String("guardrails.flow_id", flow.ID),
		attribute.String("guardrails.flow_name", flow.Name),
		attribute.String("guardrails.flow_type", flow.Type),
		attribute.Bool("guardrails.flow_active", flow.Active),
		attribute.Int("guardrails.flow_version", flow.Version),
	)

	err := t.svc.CreateFlow(ctx, flow)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (t *tracingMiddleware) GetFlow(ctx context.Context, id string) (*guardrails.Flow, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetFlow")
	defer span.End()

	span.SetAttributes(attribute.String("guardrails.flow_id", id))

	flow, err := t.svc.GetFlow(ctx, id)
	if err == nil {
		span.SetAttributes(
			attribute.String("guardrails.flow_name", flow.Name),
			attribute.String("guardrails.flow_type", flow.Type),
			attribute.Bool("guardrails.flow_active", flow.Active),
		)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return flow, err
}

func (t *tracingMiddleware) GetFlows(
	ctx context.Context,
	pm *guardrails.PageMetadata,
) ([]*guardrails.Flow, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetFlows")
	defer span.End()

	flows, err := t.svc.GetFlows(ctx, pm)
	if err == nil {
		span.SetAttributes(attribute.Int("guardrails.flows_count", len(flows)))
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return flows, err
}

func (t *tracingMiddleware) UpdateFlow(ctx context.Context, flow *guardrails.Flow) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.UpdateFlow")
	defer span.End()

	span.SetAttributes(
		attribute.String("guardrails.flow_id", flow.ID),
		attribute.String("guardrails.flow_name", flow.Name),
		attribute.String("guardrails.flow_type", flow.Type),
		attribute.Bool("guardrails.flow_active", flow.Active),
	)

	err := t.svc.UpdateFlow(ctx, flow)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (t *tracingMiddleware) DeleteFlow(ctx context.Context, id string) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.DeleteFlow")
	defer span.End()

	span.SetAttributes(attribute.String("guardrails.flow_id", id))

	err := t.svc.DeleteFlow(ctx, id)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (t *tracingMiddleware) CreateKBFile(ctx context.Context, file *guardrails.KBFile) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.CreateKBFile")
	defer span.End()

	span.SetAttributes(
		attribute.String("guardrails.kb_file_id", file.ID),
		attribute.String("guardrails.kb_file_name", file.Name),
		attribute.String("guardrails.kb_file_type", file.Type),
		attribute.String("guardrails.kb_file_category", file.Category),
		attribute.Int("guardrails.kb_file_content_size", len(file.Content)),
		attribute.Int("guardrails.kb_file_tags_count", len(file.Tags)),
		attribute.Bool("guardrails.kb_file_active", file.Active),
		attribute.Int("guardrails.kb_file_version", file.Version),
	)

	err := t.svc.CreateKBFile(ctx, file)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (t *tracingMiddleware) GetKBFile(ctx context.Context, id string) (*guardrails.KBFile, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetKBFile")
	defer span.End()

	span.SetAttributes(attribute.String("guardrails.kb_file_id", id))

	file, err := t.svc.GetKBFile(ctx, id)
	if err == nil {
		span.SetAttributes(
			attribute.String("guardrails.kb_file_name", file.Name),
			attribute.String("guardrails.kb_file_type", file.Type),
			attribute.String("guardrails.kb_file_category", file.Category),
			attribute.Int("guardrails.kb_file_content_size", len(file.Content)),
			attribute.Bool("guardrails.kb_file_active", file.Active),
		)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return file, err
}

func (t *tracingMiddleware) GetKBFiles(
	ctx context.Context,
	pm *guardrails.PageMetadata,
) ([]*guardrails.KBFile, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.GetKBFiles")
	defer span.End()

	files, err := t.svc.GetKBFiles(ctx, pm)
	if err == nil {
		span.SetAttributes(attribute.Int("guardrails.kb_files_count", len(files)))
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return files, err
}

func (t *tracingMiddleware) UpdateKBFile(ctx context.Context, file *guardrails.KBFile) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.UpdateKBFile")
	defer span.End()

	span.SetAttributes(
		attribute.String("guardrails.kb_file_id", file.ID),
		attribute.String("guardrails.kb_file_name", file.Name),
		attribute.String("guardrails.kb_file_type", file.Type),
		attribute.String("guardrails.kb_file_category", file.Category),
		attribute.Int("guardrails.kb_file_content_size", len(file.Content)),
		attribute.Bool("guardrails.kb_file_active", file.Active),
	)

	err := t.svc.UpdateKBFile(ctx, file)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (t *tracingMiddleware) DeleteKBFile(ctx context.Context, id string) error {
	ctx, span := t.tracer.Start(ctx, "guardrails.DeleteKBFile")
	defer span.End()

	span.SetAttributes(attribute.String("guardrails.kb_file_id", id))

	err := t.svc.DeleteKBFile(ctx, id)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (t *tracingMiddleware) SearchKBFiles(
	ctx context.Context,
	query string,
	categories, tags []string,
	limit int,
) ([]*guardrails.KBFile, error) {
	ctx, span := t.tracer.Start(ctx, "guardrails.SearchKBFiles")
	defer span.End()

	span.SetAttributes(
		attribute.String("guardrails.search_query", query),
		attribute.Int("guardrails.search_query_length", len(query)),
		attribute.Int("guardrails.search_categories_count", len(categories)),
		attribute.Int("guardrails.search_tags_count", len(tags)),
		attribute.Int("guardrails.search_limit", limit),
	)

	files, err := t.svc.SearchKBFiles(ctx, query, categories, tags, limit)
	if err == nil {
		span.SetAttributes(
			attribute.Int("guardrails.search_results_count", len(files)),
			attribute.Bool("guardrails.search_has_results", len(files) > 0),
		)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return files, err
}
