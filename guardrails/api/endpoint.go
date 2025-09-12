// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/ultraviolet/cube/guardrails"
)

func chatCompletionEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		fmt.Println("Testing here 1")
		fmt.Println("Chat request")
		req, ok := request.(chatCompletionRequest)
		if !ok {
			fmt.Println("Testing here 4")

			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
			return nil, err
		}

		fmt.Println("Testing here 3")

		serviceReq := &guardrails.ChatCompletionRequest{
			Model:       req.Model,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
			Stream:      req.Stream,
		}

		for _, msg := range req.Messages {
			serviceReq.Messages = append(serviceReq.Messages, guardrails.ChatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		response, err := svc.ProcessChatCompletion(ctx, serviceReq)
		if err != nil {
			return nil, err
		}

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

func getNeMoConfigEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		config, err := svc.GetNeMoConfig(ctx)
		if err != nil {
			return nil, err
		}

		return nemoConfigResponse{Config: config}, nil
	}
}

func getNeMoConfigYAMLEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		config, err := svc.GetNeMoConfigYAML(ctx)
		if err != nil {
			return nil, err
		}

		return config, nil
	}
}

func createFlowEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createFlowRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
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

		if err := svc.CreateFlow(ctx, &flow); err != nil {
			return nil, err
		}

		return createFlowResponse{
			ID:      flow.ID,
			Message: "Flow created successfully",
		}, nil
	}
}

// getFlowEndpoint creates endpoint for getting a flow by ID.
func getFlowEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(getFlowRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
			return nil, err
		}

		flow, err := svc.GetFlow(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return getFlowResponse{Flow: *flow}, nil
	}
}

func getFlowsEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		pm := &guardrails.PageMetadata{}
		flows, err := svc.GetFlows(ctx, pm)
		if err != nil {
			return nil, err
		}

		flowVals := make([]guardrails.Flow, len(flows))
		for i, f := range flows {
			flowVals[i] = *f
		}
		return getFlowsResponse{Flows: flowVals}, nil
	}
}

func updateFlowEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateFlowRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
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

		if err := svc.UpdateFlow(ctx, &flow); err != nil {
			return nil, err
		}

		updatedFlow, err := svc.GetFlow(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return updateFlowResponse{
			Flow:    *updatedFlow,
			Message: "Flow updated successfully",
		}, nil
	}
}

func deleteFlowEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteFlowRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
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

func createKBFileEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createKBFileRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
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

		if err := svc.CreateKBFile(ctx, &file); err != nil {
			return nil, err
		}

		return createKBFileResponse{
			ID:      file.ID,
			Message: "KB file created successfully",
		}, nil
	}
}

func getKBFileEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(getKBFileRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
			return nil, err
		}

		file, err := svc.GetKBFile(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return getKBFileResponse{File: *file}, nil
	}
}

func getKBFilesEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		pm := &guardrails.PageMetadata{}
		files, err := svc.GetKBFiles(ctx, pm)
		if err != nil {
			return nil, err
		}

		fileVals := make([]guardrails.KBFile, len(files))
		for i, f := range files {
			fileVals[i] = *f
		}
		return getKBFilesResponse{Files: fileVals}, nil
	}
}

func listKBFilesEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listKBFilesRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
			return nil, err
		}

		pm := guardrails.PageMetadata{
			Limit:    req.Limit,
			Offset:   req.Offset,
			Category: req.Category,
		}
		if len(req.Tags) > 0 {
			pm.User = strings.Join(req.Tags, ",")
		}
		files, err := svc.GetKBFiles(ctx, &pm)
		if err != nil {
			return nil, err
		}

		fileVals := make([]guardrails.KBFile, len(files))
		for i, f := range files {
			fileVals[i] = *f
		}
		response := listKBFilesResponse{
			Files: fileVals,
		}
		response.PageMetadata = req.PageMetadata
		response.Total = uint64(len(files))

		return response, nil
	}
}

func updateKBFileEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateKBFileRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
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

		if err := svc.UpdateKBFile(ctx, &file); err != nil {
			return nil, err
		}

		updatedFile, err := svc.GetKBFile(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return updateKBFileResponse{
			File:    *updatedFile,
			Message: "KB file updated successfully",
		}, nil
	}
}

func deleteKBFileEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteKBFileRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
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

func searchKBFilesEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(searchKBFilesRequest)
		if !ok {
			return nil, errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid request type"))
		}
		if err := (&req).validate(); err != nil {
			return nil, err
		}

		files, err := svc.SearchKBFiles(ctx, req.Query, req.Categories, req.Tags, int(req.Limit))
		if err != nil {
			return nil, err
		}

		fileVals := make([]guardrails.KBFile, len(files))
		for i, f := range files {
			fileVals[i] = *f
		}
		return searchKBFilesResponse{Files: fileVals}, nil
	}
}
