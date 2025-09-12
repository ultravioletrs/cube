// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"strings"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/ultraviolet/cube/guardrails"
)

// chatCompletionEndpoint creates endpoint for chat completion requests
func chatCompletionEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(chatCompletionRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		// Convert API request to service request format
		serviceReq := &guardrails.ChatCompletionRequest{
			Model:       req.Model,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			Stream:      req.Stream,
			UserID:      req.UserID,
		}

		// Convert messages
		for _, msg := range req.Messages {
			serviceReq.Messages = append(serviceReq.Messages, guardrails.ChatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		// Process request
		response, err := svc.ProcessChatCompletion(ctx, serviceReq)
		if err != nil {
			return nil, err
		}

		// Convert service response to API response
		apiResp := &chatCompletionResponse{
			ID:      response.ID,
			Object:  response.Object,
			Created: response.Created,
			Model:   response.Model,
			Usage: usage{
				PromptTokens:     response.Usage.PromptTokens,
				CompletionTokens: response.Usage.CompletionTokens,
				TotalTokens:      response.Usage.TotalTokens,
			},
		}

		// Convert choices
		for _, choice := range response.Choices {
			apiResp.Choices = append(apiResp.Choices, chatChoice{
				Index: choice.Index,
				Message: chatMessage{
					Role:    choice.Message.Role,
					Content: choice.Message.Content,
				},
				FinishReason: choice.FinishReason,
			})
		}

		return apiResp, nil
	}
}

// getNeMoConfigEndpoint creates endpoint for getting NeMo configuration
func getNeMoConfigEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		config, err := svc.GetNeMoConfig(ctx)
		if err != nil {
			return nil, err
		}

		return nemoConfigResponse{Config: config}, nil
	}
}

// getNeMoConfigYAMLEndpoint creates endpoint for getting NeMo configuration as YAML
func getNeMoConfigYAMLEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		config, err := svc.GetNeMoConfigYAML(ctx)
		if err != nil {
			return nil, err
		}

		return config, nil
	}
}

// Flow management endpoints

// createFlowEndpoint creates endpoint for creating a flow
func createFlowEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createFlowRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		flow := guardrails.Flow{
			Name:        req.Name,
			Description: req.Description,
			Content:     req.Content,
			Type:        req.Type,
			Active:      req.Active,
			Version:     1,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		}

		if err := svc.CreateFlow(ctx, flow); err != nil {
			return nil, err
		}

		return createFlowResponse{
			ID:      flow.ID,
			Message: "Flow created successfully",
		}, nil
	}
}

// getFlowEndpoint creates endpoint for getting a flow by ID
func getFlowEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(getFlowRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		flow, err := svc.GetFlow(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return getFlowResponse{Flow: flow}, nil
	}
}

// getFlowsEndpoint creates endpoint for listing flows
func getFlowsEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		flows, err := svc.GetFlows(ctx, guardrails.PageMetadata{})
		if err != nil {
			return nil, err
		}

		return getFlowsResponse{Flows: flows}, nil
	}
}

// updateFlowEndpoint creates endpoint for updating a flow
func updateFlowEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateFlowRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		flow := guardrails.Flow{
			ID:          req.ID,
			Name:        req.Name,
			Description: req.Description,
			Content:     req.Content,
			Type:        req.Type,
			Active:      req.Active,
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		}

		if err := svc.UpdateFlow(ctx, flow); err != nil {
			return nil, err
		}

		// Get updated flow to return
		updatedFlow, err := svc.GetFlow(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return updateFlowResponse{
			Flow:    updatedFlow,
			Message: "Flow updated successfully",
		}, nil
	}
}

// deleteFlowEndpoint creates endpoint for deleting a flow
func deleteFlowEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteFlowRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.DeleteFlow(ctx, req.ID); err != nil {
			return nil, err
		}

		return deleteFlowResponse{
			Message: "Flow deleted successfully",
		}, nil
	}
}

// Knowledge Base management endpoints

// createKBFileEndpoint creates endpoint for creating a KB file
func createKBFileEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createKBFileRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		file := guardrails.KBFile{
			Name:      req.Name,
			Content:   req.Content,
			Type:      req.Type,
			Category:  req.Category,
			Tags:      req.Tags,
			Metadata:  req.Metadata,
			Active:    req.Active,
			Version:   1,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		if err := svc.CreateKBFile(ctx, file); err != nil {
			return nil, err
		}

		return createKBFileResponse{
			ID:      file.ID,
			Message: "KB file created successfully",
		}, nil
	}
}

// getKBFileEndpoint creates endpoint for getting a KB file by ID
func getKBFileEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(getKBFileRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		file, err := svc.GetKBFile(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return getKBFileResponse{File: file}, nil
	}
}

// getKBFilesEndpoint creates endpoint for listing KB files
func getKBFilesEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		files, err := svc.GetKBFiles(ctx, guardrails.PageMetadata{})
		if err != nil {
			return nil, err
		}

		return getKBFilesResponse{Files: files}, nil
	}
}

// listKBFilesEndpoint creates endpoint for listing KB files with filtering
func listKBFilesEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listKBFilesRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		// Convert request parameters to PageMetadata
		pm := guardrails.PageMetadata{
			Limit:    req.Limit,
			Offset:   req.Offset,
			Category: req.Category,
		}
		if len(req.Tags) > 0 {
			// Join tags into User field for filtering
			pm.User = strings.Join(req.Tags, ",")
		}
		files, err := svc.GetKBFiles(ctx, pm)
		if err != nil {
			return nil, err
		}

		response := listKBFilesResponse{
			Files: files,
		}
		response.PageMetadata = req.PageMetadata
		response.Total = uint64(len(files))

		return response, nil
	}
}

// updateKBFileEndpoint creates endpoint for updating a KB file
func updateKBFileEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateKBFileRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		file := guardrails.KBFile{
			ID:        req.ID,
			Name:      req.Name,
			Content:   req.Content,
			Type:      req.Type,
			Category:  req.Category,
			Tags:      req.Tags,
			Metadata:  req.Metadata,
			Active:    req.Active,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		if err := svc.UpdateKBFile(ctx, file); err != nil {
			return nil, err
		}

		// Get updated file to return
		updatedFile, err := svc.GetKBFile(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return updateKBFileResponse{
			File:    updatedFile,
			Message: "KB file updated successfully",
		}, nil
	}
}

// deleteKBFileEndpoint creates endpoint for deleting a KB file
func deleteKBFileEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteKBFileRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.DeleteKBFile(ctx, req.ID); err != nil {
			return nil, err
		}

		return deleteKBFileResponse{
			Message: "KB file deleted successfully",
		}, nil
	}
}

// searchKBFilesEndpoint creates endpoint for searching KB files
func searchKBFilesEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(searchKBFilesRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := req.validate(); err != nil {
			return nil, err
		}

		files, err := svc.SearchKBFiles(ctx, req.Query, req.Categories, req.Tags, int(req.Limit))
		if err != nil {
			return nil, err
		}

		return searchKBFilesResponse{Files: files}, nil
	}
}
