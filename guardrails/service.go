package guardrails

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/ultraviolet/cube/pkg/sdk"
)

type service struct {
	config        *ServiceConfig
	repo          Repository
	transport     *http.Transport
	httpClient    *http.Client
	configManager *ConfigManager
	chatService   *ChatService
	logger        *slog.Logger
	idp           supermq.IDProvider
	nemoSDK       sdk.NeMoGuardrailsSDK
}

func New(config *ServiceConfig, repo Repository, logger *slog.Logger, idp supermq.IDProvider) (Service, error) {
	if config.TargetURL == "" {
		return nil, errors.New("target URL must be provided")
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	if config.TLS.Enabled {
		tlsConfig, err := setTLSConfig(config)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("failed to set TLS config: %w", err))
		}
		transport.TLSClientConfig = tlsConfig
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	nemoConfig := sdk.NeMoConfig{
		BaseURL:         config.GuardrailsURL,
		Timeout:         time.Duration(config.Timeout) * time.Second,
		TLSVerification: config.TLS.Enabled && !config.TLS.InsecureSkipVerify,
		MaxRetries:      3,
	}
	nemoSDK := sdk.NewNeMoGuardrailsSDK(nemoConfig)

	svc := &service{
		config:     config,
		repo:       repo,
		transport:  transport,
		httpClient: httpClient,
		logger:     logger,
		idp:        idp,
		nemoSDK:    nemoSDK,
	}

	svc.configManager = NewConfigManager(repo, logger, config.PolicyConfigPath, idp, nemoSDK)

	svc.chatService = NewChatService(config, repo, svc.configManager, logger, idp)

	return svc, nil
}

func (s *service) Proxy() *httputil.ReverseProxy {
	var target *url.URL
	var err error

	target, err = url.Parse(s.config.GuardrailsURL)
	if err != nil {
		panic(fmt.Sprintf("invalid guardrails URL: %v", err))
	}
	s.logger.Info("Guardrails enabled: routing through", "url", s.config.GuardrailsURL)

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Transport = s.transport

	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return reverseProxy
}

func (s *service) ProcessChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	chatReq := &ChatRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
	}

	for _, msg := range req.Messages {
		chatReq.Messages = append(chatReq.Messages, ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	chatResp, err := s.chatService.ProcessChatRequest(ctx, chatReq)
	if err != nil {
		s.logger.Error("Chat completion failed", "error", err)
		return nil, errors.New(fmt.Sprintf("chat completion failed: %w", err))
	}

	apiResp := &ChatCompletionResponse{
		ID:      chatResp.ID,
		Object:  chatResp.Object,
		Created: chatResp.Created,
		Model:   chatResp.Model,
		Usage: Usage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}

	for _, choice := range chatResp.Choices {
		apiResp.Choices = append(apiResp.Choices, ChatChoice{
			Index: choice.Index,
			Message: ChatMessage{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
			FinishReason: choice.FinishReason,
		})
	}

	return apiResp, nil
}

func (s *service) GetNeMoConfig(ctx context.Context) ([]byte, error) {
	if s.configManager == nil {
		return nil, errors.New("config manager not initialized")
	}

	config, err := s.configManager.GenerateNeMoConfig(ctx)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to generate NeMo config: %w", err))
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to marshal config to JSON: %w", err))
	}

	return configJSON, nil
}

func (s *service) GetNeMoConfigYAML(ctx context.Context) ([]byte, error) {
	if s.configManager == nil {
		return nil, errors.New("config manager not initialized")
	}
	return s.configManager.GetConfigYAML(ctx)
}

func (s *service) CreateFlow(ctx context.Context, flow *Flow) error {
	id, err := s.idp.ID()
	if err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, err)
	}
	flow.ID = id

	if err := s.repo.CreateFlow(ctx, flow); err != nil {
		return err
	}

	if s.configManager != nil {

		if err := s.configManager.PushConfigurationToNeMo(ctx); err != nil {
			s.logger.Error("Failed to push configuration to NeMo", "error", err)
		}
	}

	return nil
}

func (s *service) GetFlow(ctx context.Context, id string) (*Flow, error) {
	return s.repo.GetFlow(ctx, id)
}

func (s *service) GetFlows(ctx context.Context, pm *PageMetadata) ([]*Flow, error) {
	return s.repo.GetFlows(ctx, pm)
}

func (s *service) UpdateFlow(ctx context.Context, flow *Flow) error {
	if err := s.repo.UpdateFlow(ctx, flow); err != nil {
		return err
	}

	if s.configManager != nil {

		if err := s.configManager.PushConfigurationToNeMo(ctx); err != nil {
			s.logger.Error("Failed to push configuration to NeMo", "error", err)
		}
	}

	return nil
}

func (s *service) DeleteFlow(ctx context.Context, id string) error {
	if err := s.repo.DeleteFlow(ctx, id); err != nil {
		return err
	}

	if s.configManager != nil {

		if err := s.configManager.PushConfigurationToNeMo(ctx); err != nil {
			s.logger.Error("Failed to push configuration to NeMo", "error", err)
		}
	}

	return nil
}

func (s *service) CreateKBFile(ctx context.Context, file *KBFile) error {
	id, err := s.idp.ID()
	if err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, err)
	}
	file.ID = id

	if err := s.repo.CreateKBFile(ctx, file); err != nil {
		return err
	}

	if s.configManager != nil {

		if err := s.configManager.PushConfigurationToNeMo(ctx); err != nil {
			s.logger.Error("Failed to push configuration to NeMo", "error", err)
		}
	}

	return nil
}

func (s *service) GetKBFile(ctx context.Context, id string) (*KBFile, error) {
	return s.repo.GetKBFile(ctx, id)
}

func (s *service) GetKBFiles(ctx context.Context, pm *PageMetadata) ([]*KBFile, error) {
	return s.repo.GetKBFiles(ctx, pm)
}

func (s *service) UpdateKBFile(ctx context.Context, file *KBFile) error {
	if err := s.repo.UpdateKBFile(ctx, file); err != nil {
		return err
	}

	if s.configManager != nil {

		if err := s.configManager.PushConfigurationToNeMo(ctx); err != nil {
			s.logger.Error("Failed to push configuration to NeMo", "error", err)
		}
	}

	return nil
}

func (s *service) DeleteKBFile(ctx context.Context, id string) error {
	_, _ = s.repo.GetKBFile(ctx, id)

	if err := s.repo.DeleteKBFile(ctx, id); err != nil {
		return err
	}

	if s.configManager != nil {

		if err := s.configManager.PushConfigurationToNeMo(ctx); err != nil {
			s.logger.Error("Failed to push configuration to NeMo", "error", err)
		}
	}

	return nil
}

func (s *service) SearchKBFiles(ctx context.Context, query string, categories, tags []string, limit int) ([]*KBFile, error) {
	return s.repo.SearchKBFiles(ctx, query, categories, tags, limit)
}
