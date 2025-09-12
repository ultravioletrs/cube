// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package guardrails

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/ultraviolet/cube/pkg/sdk"
)

type ChatService struct {
	config        *ServiceConfig
	repo          Repository
	configManager *ConfigManager
	nemoSDK       sdk.NeMoGuardrailsSDK
	logger        *slog.Logger
	idp           supermq.IDProvider
}

type ChatRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type ChatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
}

type RequestContext struct {
	Timestamp time.Time `json:"timestamp"`
}

func NewChatService(config *ServiceConfig, repo Repository, configManager *ConfigManager, logger *slog.Logger, idp supermq.IDProvider) *ChatService {
	nemoConfig := sdk.NeMoConfig{
		BaseURL:         config.GuardrailsURL,
		Timeout:         time.Duration(config.Timeout) * time.Second,
		TLSVerification: config.TLS.Enabled && !config.TLS.InsecureSkipVerify,
		MaxRetries:      3,
	}

	nemoSDK := sdk.NewNeMoGuardrailsSDK(nemoConfig)

	return &ChatService{
		config:        config,
		repo:          repo,
		configManager: configManager,
		nemoSDK:       nemoSDK,
		logger:        logger,
		idp:           idp,
	}
}

func (cs *ChatService) ProcessChatRequest(ctx context.Context, request *ChatRequest) (*ChatResponse, error) {
	reqCtx := &RequestContext{
		Timestamp: time.Now(),
	}

	cs.logger.Info("Processing chat request",
		"model", request.Model,
		"message_count", len(request.Messages),
	)

	nemoRequest, err := cs.transformRequestForNeMo(request)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to transform request for NeMo: %w", err))
	}

	nemoResponse, err := cs.sendToNeMoGuardrails(ctx, nemoRequest, reqCtx)
	if err != nil {
		cs.logger.Error("NeMo Guardrails request failed",
			"error", err,
		)
		return nil, errors.New(fmt.Sprintf("guardrails processing failed: %w", err))
	}

	finalResponse := cs.transformResponseFromNeMo(nemoResponse, request.Model)

	cs.logger.Info("Chat request processed successfully")

	return finalResponse, nil
}

func (cs *ChatService) transformRequestForNeMo(request *ChatRequest) (map[string]interface{}, error) {
	transformed := map[string]interface{}{
		"model":    request.Model,
		"messages": request.Messages,
	}

	if request.Temperature > 0 {
		transformed["temperature"] = request.Temperature
	}
	if request.MaxTokens > 0 {
		transformed["max_tokens"] = request.MaxTokens
	}
	if request.Stream {
		transformed["stream"] = request.Stream
	}

	return transformed, nil
}

func (cs *ChatService) sendToNeMoGuardrails(ctx context.Context, request map[string]interface{}, reqCtx *RequestContext) (*ChatResponse, error) {
	sdkRequest := sdk.ChatCompletionRequest{}

	if model, ok := request["model"].(string); ok {
		sdkRequest.Model = model
	}
	if temperature, ok := request["temperature"].(float64); ok {
		sdkRequest.Temperature = temperature
	}
	if maxTokens, ok := request["max_tokens"].(int); ok {
		sdkRequest.MaxTokens = maxTokens
	}
	if stream, ok := request["stream"].(bool); ok {
		sdkRequest.Stream = stream
	}

	if messages, ok := request["messages"]; ok {
		switch msgs := messages.(type) {
		case []interface{}:
			for _, msg := range msgs {
				if msgMap, ok := msg.(map[string]interface{}); ok {
					role, _ := msgMap["role"].(string)
					content, _ := msgMap["content"].(string)
					sdkRequest.Messages = append(sdkRequest.Messages, sdk.ChatMessage{
						Role:    role,
						Content: content,
					})
				}
			}
		case []ChatMessage:
			for _, msg := range msgs {
				sdkRequest.Messages = append(sdkRequest.Messages, sdk.ChatMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
	}

	cs.logger.Debug("Sending request to NeMo Guardrails via SDK",
		"model", sdkRequest.Model,
		"message_count", len(sdkRequest.Messages),
	)

	sdkResponse, err := cs.nemoSDK.ChatCompletion(ctx, sdkRequest)
	if err != nil {
		cs.logger.Error("NeMo Guardrails SDK request failed",
			"error", err,
		)
		return nil, errors.New(fmt.Sprintf("NeMo Guardrails request failed: %w", err))
	}

	chatResponse := &ChatResponse{
		ID:      sdkResponse.ID,
		Object:  sdkResponse.Object,
		Created: sdkResponse.Created,
		Model:   sdkResponse.Model,
		Usage: Usage{
			PromptTokens:     sdkResponse.Usage.PromptTokens,
			CompletionTokens: sdkResponse.Usage.CompletionTokens,
			TotalTokens:      sdkResponse.Usage.TotalTokens,
		},
	}

	for _, choice := range sdkResponse.Choices {
		chatResponse.Choices = append(chatResponse.Choices, ChatChoice{
			Index: choice.Index,
			Message: ChatMessage{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
			FinishReason: choice.FinishReason,
		})
	}

	return chatResponse, nil
}

func (cs *ChatService) transformResponseFromNeMo(response *ChatResponse, model string) *ChatResponse {
	if response.ID == "" {
		response.ID = "chat-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if response.Object == "" {
		response.Object = "chat.completion"
	}
	if response.Created == 0 {
		response.Created = time.Now().Unix()
	}
	if response.Model == "" {
		response.Model = model
	}

	return response
}
