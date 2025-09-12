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
	RequestID string    `json:"request_id"`
	UserID    string    `json:"user_id,omitempty"`
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

func (cs *ChatService) ProcessChatRequest(ctx context.Context, request *ChatRequest, userID string) (*ChatResponse, error) {
	requestID, err := cs.idp.ID()
	if err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	reqCtx := &RequestContext{
		RequestID: requestID,
		UserID:    userID,
		Timestamp: time.Now(),
	}

	cs.logger.Info("Processing chat request",
		"request_id", requestID,
		"user_id", userID,
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
			"request_id", requestID,
			"error", err,
		)
		return nil, errors.New(fmt.Sprintf("guardrails processing failed: %w", err))
	}

	cs.logLLMResponse(requestID, nemoResponse, request)

	finalResponse := cs.transformResponseFromNeMo(nemoResponse, requestID, request.Model)

	cs.logger.Info("Chat request processed successfully",
		"request_id", requestID,
	)

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
	sdkRequest := sdk.ChatCompletionRequest{
		UserID: reqCtx.UserID,
	}

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
		"request_id", reqCtx.RequestID,
		"model", sdkRequest.Model,
		"message_count", len(sdkRequest.Messages),
	)

	sdkResponse, err := cs.nemoSDK.ChatCompletion(ctx, sdkRequest)
	if err != nil {
		cs.logger.Error("NeMo Guardrails SDK request failed",
			"request_id", reqCtx.RequestID,
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

func (cs *ChatService) transformResponseFromNeMo(response *ChatResponse, requestID, model string) *ChatResponse {
	if response.ID == "" {
		response.ID = requestID
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

func (cs *ChatService) logLLMResponse(requestID string, response *ChatResponse, originalRequest *ChatRequest) {
	responseLength := 0
	choiceCount := len(response.Choices)
	var finishReasons []string

	for _, choice := range response.Choices {
		responseLength += len(choice.Message.Content)
		finishReasons = append(finishReasons, choice.FinishReason)
	}

	processingTime := time.Now().Unix() - response.Created

	cs.logger.Info("LLM response generated",
		"event", "llm_response_logged",
		"request_id", requestID,
		"response_id", response.ID,
		"model", response.Model,
		"choice_count", choiceCount,
		"response_length", responseLength,
		"finish_reasons", finishReasons,
		"estimated_processing_time_seconds", processingTime,
		"prompt_tokens", response.Usage.PromptTokens,
		"completion_tokens", response.Usage.CompletionTokens,
		"total_tokens", response.Usage.TotalTokens,
		"original_model_requested", originalRequest.Model,
		"temperature", originalRequest.Temperature,
		"max_tokens", originalRequest.MaxTokens,
		"stream_mode", originalRequest.Stream,
		"timestamp", time.Now().Format(time.RFC3339),
	)

	for i, choice := range response.Choices {
		cs.logger.Debug("LLM response choice detail",
			"event", "llm_choice_logged",
			"request_id", requestID,
			"choice_index", choice.Index,
			"choice_position", i,
			"message_role", choice.Message.Role,
			"message_length", len(choice.Message.Content),
			"finish_reason", choice.FinishReason,
			"content_preview", truncateString(choice.Message.Content, 100),
			"timestamp", time.Now().Format(time.RFC3339),
		)
	}

	cs.logResponseQualityMetrics(requestID, response)
}

func (cs *ChatService) logResponseQualityMetrics(requestID string, response *ChatResponse) {
	for _, choice := range response.Choices {
		content := choice.Message.Content

		metrics := map[string]interface{}{
			"word_count":        len([]rune(content)) / 5,
			"character_count":   len(content),
			"contains_code":     containsCodePatterns(content),
			"contains_urls":     containsURLs(content),
			"contains_numbers":  containsNumbers(content),
			"response_type":     detectResponseType(content),
			"safety_indicators": checkSafetyIndicators(content),
		}

		cs.logger.Debug("LLM response quality metrics",
			"event", "response_quality_logged",
			"request_id", requestID,
			"choice_index", choice.Index,
			"metrics", metrics,
			"timestamp", time.Now().Format(time.RFC3339),
		)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func containsCodePatterns(content string) bool {
	codeIndicators := []string{"```", "def ", "function ", "class ", "import ", "const ", "var ", "let "}
	for _, indicator := range codeIndicators {
		if len(content) > 0 && contains(content, indicator) {
			return true
		}
	}
	return false
}

func containsURLs(content string) bool {
	return contains(content, "http://") || contains(content, "https://") || contains(content, "www.")
}

func containsNumbers(content string) bool {
	for _, char := range content {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return false
}

func detectResponseType(content string) string {
	if containsCodePatterns(content) {
		return "code"
	}
	if containsURLs(content) {
		return "reference"
	}
	if len(content) > 500 {
		return "detailed"
	}
	if len(content) < 50 {
		return "brief"
	}
	return "standard"
}

func checkSafetyIndicators(content string) map[string]bool {
	return map[string]bool{
		"disclaimer_present":    contains(content, "I cannot") || contains(content, "I'm sorry"),
		"uncertainty_expressed": contains(content, "I'm not sure") || contains(content, "might be"),
		"helpful_tone":          contains(content, "help") || contains(content, "assist"),
		"refusal_present":       contains(content, "cannot do") || contains(content, "not appropriate"),
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
