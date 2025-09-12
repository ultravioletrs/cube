// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package guardrails

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/absmach/supermq/pkg/errors"
)

type TLSConfig struct {
	Enabled            bool
	InsecureSkipVerify bool
	CertFile           string
	KeyFile            string
	CAFile             string
	MinVersion         uint16
	MaxVersion         uint16
}

type ServiceConfig struct {
	GuardrailsURL    string `env:"GUARDRAILS_URL"       envDefault:"http://nemo-guardrails:8001"`
	TargetURL        string `env:"TARGET_URL"           envDefault:"http://cube-agent:8901"`
	TLS              TLSConfig
	PolicyConfigPath string `env:"POLICY_CONFIG_PATH"   envDefault:"/config/guardrails_config.yaml"`
	Timeout          int    `env:"TIMEOUT"              envDefault:"30"`
}

type Service interface {
	// Policy Management
	CreatePolicy(ctx context.Context, policy Policy) error
	GetPolicy(ctx context.Context, id string) (Policy, error)
	ListPolicies(ctx context.Context, limit, offset int) ([]Policy, error)
	UpdatePolicy(ctx context.Context, policy Policy) error
	DeletePolicy(ctx context.Context, id string) error

	// Repository methods for external UI
	// Restricted Topics management
	GetRestrictedTopics(ctx context.Context) ([]string, error)
	UpdateRestrictedTopics(ctx context.Context, topics []string) error
	AddRestrictedTopic(ctx context.Context, topic string) error
	RemoveRestrictedTopic(ctx context.Context, topic string) error

	// Bias Patterns management
	GetBiasPatterns(ctx context.Context) (map[string][]BiasPattern, error)
	UpdateBiasPatterns(ctx context.Context, patterns map[string][]BiasPattern) error

	// Factuality Config management
	GetFactualityConfig(ctx context.Context) (FactualityConfig, error)
	UpdateFactualityConfig(ctx context.Context, config FactualityConfig) error

	// Audit Log management
	GetAuditLogs(ctx context.Context, limit int) ([]AuditLog, error)

	// Configuration export/import
	ExportConfig(ctx context.Context) ([]byte, error)
	ImportConfig(ctx context.Context, data []byte) error

	// Content Processing
	ProcessRequest(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error)
	ProcessResponse(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error)
	ValidateRequest(ctx context.Context, request interface{}) error
	ValidateResponse(ctx context.Context, response interface{}) error

	// Proxy functionality
	Proxy() *httputil.ReverseProxy
}

// Repository interface for data access
type Repository interface {
	// Policy management
	CreatePolicy(ctx context.Context, policy Policy) error
	GetPolicy(ctx context.Context, id string) (Policy, error)
	ListPolicies(ctx context.Context, limit, offset int) ([]Policy, error)
	UpdatePolicy(ctx context.Context, policy Policy) error
	DeletePolicy(ctx context.Context, id string) error

	// Restricted Topics management
	GetRestrictedTopics(ctx context.Context) ([]string, error)
	UpdateRestrictedTopics(ctx context.Context, topics []string) error
	AddRestrictedTopic(ctx context.Context, topic string) error
	RemoveRestrictedTopic(ctx context.Context, topic string) error

	// Bias Patterns management
	GetBiasPatterns(ctx context.Context) (map[string][]BiasPattern, error)
	UpdateBiasPatterns(ctx context.Context, patterns map[string][]BiasPattern) error

	// Factuality Config management
	GetFactualityConfig(ctx context.Context) (FactualityConfig, error)
	UpdateFactualityConfig(ctx context.Context, config FactualityConfig) error

	// Audit Log management
	CreateAuditLog(ctx context.Context, log AuditLog) error
	GetAuditLogs(ctx context.Context, limit int) ([]AuditLog, error)

	// Configuration export/import
	ExportConfig(ctx context.Context) ([]byte, error)
	ImportConfig(ctx context.Context, data []byte) error
}

type service struct {
	config     *ServiceConfig
	repo       Repository
	transport  *http.Transport
	httpClient *http.Client
}

func New(config *ServiceConfig, repo Repository) (Service, error) {
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
			return nil, fmt.Errorf("failed to set TLS config: %w", err)
		}
		transport.TLSClientConfig = tlsConfig
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	return &service{
		config:     config,
		repo:       repo,
		transport:  transport,
		httpClient: httpClient,
	}, nil
}

func (s *service) ProcessRequest(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.GuardrailsURL+"/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request to guardrails: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("guardrails request failed with status %d", resp.StatusCode)
	}

	return respBody, resp.Header, nil
}

func (s *service) ProcessResponse(ctx context.Context, body []byte, headers http.Header) ([]byte, http.Header, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return body, headers, nil
	}

	if err := s.ValidateResponse(ctx, response); err != nil {
		filteredResponse := map[string]interface{}{
			"error": map[string]string{
				"message": "Response blocked by content policy",
				"type":    "content_policy_violation",
			},
		}
		filteredBody, _ := json.Marshal(filteredResponse)
		return filteredBody, headers, nil
	}

	return body, headers, nil
}

func (s *service) ValidateRequest(ctx context.Context, request interface{}) error {
	reqMap, ok := request.(map[string]interface{})
	if !ok {
		return nil
	}

	// Handle different request formats

	// Check for chat completion format with messages
	if messages, ok := reqMap["messages"].([]interface{}); ok {
		for _, msg := range messages {
			msgMap, ok := msg.(map[string]interface{})
			if !ok {
				continue
			}

			content, ok := msgMap["content"].(string)
			if !ok {
				continue
			}

			if err := s.validateContent(content); err != nil {
				return err
			}
		}
	}

	// Check for direct prompt field (Ollama format)
	if prompt, ok := reqMap["prompt"].(string); ok {
		if err := s.validateContent(prompt); err != nil {
			return err
		}
	}

	// Check for query field
	if query, ok := reqMap["query"].(string); ok {
		if err := s.validateContent(query); err != nil {
			return err
		}
	}

	// Check for text field
	if text, ok := reqMap["text"].(string); ok {
		if err := s.validateContent(text); err != nil {
			return err
		}
	}

	// Check for content field
	if content, ok := reqMap["content"].(string); ok {
		if err := s.validateContent(content); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) ValidateResponse(ctx context.Context, response interface{}) error {
	respMap, ok := response.(map[string]interface{})
	if !ok {
		return nil
	}

	choices, ok := respMap["choices"].([]interface{})
	if !ok {
		return nil
	}

	for _, choice := range choices {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			continue
		}

		message, ok := choiceMap["message"].(map[string]interface{})
		if !ok {
			continue
		}

		content, ok := message["content"].(string)
		if !ok {
			continue
		}

		if err := s.validateContent(content); err != nil {
			return err
		}
	}

	return nil
}

// Proxy implements guardrails.Service.
func (s *service) Proxy() *httputil.ReverseProxy {
	// When guardrails are enabled, proxy to nemo-guardrails first
	// Otherwise, proxy directly to the target (agent)
	var target *url.URL
	var err error

	target, err = url.Parse(s.config.GuardrailsURL)
	if err != nil {
		panic(fmt.Sprintf("invalid guardrails URL: %v", err))
	}
	fmt.Printf("Guardrails enabled: routing through %s\n", s.config.GuardrailsURL)

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Transport = s.transport

	// Store original director and wrap it with validation
	//originalDirector := reverseProxy.Director
	//reverseProxy.Director = func(req *http.Request) {
	//	// Store the original path before modification
	//	originalPath := req.URL.Path
	//
	//	// Apply original director first
	//	originalDirector(req)
	//
	//	// Ensure the path is preserved correctly
	//	req.URL.Path = originalPath
	//	req.URL.RawPath = originalPath
	//
	//	fmt.Printf("Proxying request: Method=%s, Path=%s, URL=%s\n", req.Method, req.URL.Path, req.URL.String())
	//
	//	// Validate all requests when guardrails are enabled
	//	if s.config.Enabled && req.Body != nil {
	//		body, err := io.ReadAll(req.Body)
	//		if err == nil {
	//			// Parse request JSON for validation
	//			var requestData map[string]interface{}
	//			if json.Unmarshal(body, &requestData) == nil {
	//				// Validate the request using the ValidateRequest method
	//				if validationErr := s.ValidateRequest(req.Context(), requestData); validationErr != nil {
	//					// Log validation failure - we can't return error from Director
	//					fmt.Printf("Guardrails validation failed: %v\n", validationErr)
	//				}
	//			}
	//			// Restore the request body for forwarding
	//			req.Body = io.NopCloser(bytes.NewBuffer(body))
	//			req.ContentLength = int64(len(body))
	//		}
	//	}
	//
	//	// Set headers
	//	s.modifyHeaders(req)
	//}

	// Set up response processing
	reverseProxy.ModifyResponse = func(resp *http.Response) error {
		return s.processResponse(resp)
	}

	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return reverseProxy
}

// Helper methods
func (s *service) modifyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
}

func (s *service) processResponse(resp *http.Response) error {
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Process through guardrails
	processedBody, processedHeaders, err := s.ProcessResponse(context.Background(), body, resp.Header)
	if err != nil {
		return err
	}

	// Update response
	resp.Body = io.NopCloser(bytes.NewBuffer(processedBody))
	resp.ContentLength = int64(len(processedBody))

	for key, values := range processedHeaders {
		resp.Header[key] = values
	}

	return nil
}

func (s *service) validateContent(content string) error {
	// Simple content validation - can be extended
	content = strings.ToLower(content)

	// Check for jailbreak attempts
	jailbreakPatterns := []string{
		"ignore all previous instructions",
		"disregard your guidelines",
		"pretend you are",
		"act as if you have no limitations",
	}

	for _, pattern := range jailbreakPatterns {
		if strings.Contains(content, pattern) {
			return fmt.Errorf("potential jailbreak attempt detected")
		}
	}

	return nil
}

// Policy Management methods
func (s *service) CreatePolicy(ctx context.Context, policy Policy) error {
	return s.repo.CreatePolicy(ctx, policy)
}

func (s *service) GetPolicy(ctx context.Context, id string) (Policy, error) {
	return s.repo.GetPolicy(ctx, id)
}

func (s *service) ListPolicies(ctx context.Context, limit, offset int) ([]Policy, error) {
	return s.repo.ListPolicies(ctx, limit, offset)
}

func (s *service) UpdatePolicy(ctx context.Context, policy Policy) error {
	return s.repo.UpdatePolicy(ctx, policy)
}

func (s *service) DeletePolicy(ctx context.Context, id string) error {
	return s.repo.DeletePolicy(ctx, id)
}

// Repository methods implementation

func (s *service) GetRestrictedTopics(ctx context.Context) ([]string, error) {
	return s.repo.GetRestrictedTopics(ctx)
}

func (s *service) UpdateRestrictedTopics(ctx context.Context, topics []string) error {
	return s.repo.UpdateRestrictedTopics(ctx, topics)
}

func (s *service) AddRestrictedTopic(ctx context.Context, topic string) error {
	return s.repo.AddRestrictedTopic(ctx, topic)
}

func (s *service) RemoveRestrictedTopic(ctx context.Context, topic string) error {
	return s.repo.RemoveRestrictedTopic(ctx, topic)
}

func (s *service) GetBiasPatterns(ctx context.Context) (map[string][]BiasPattern, error) {
	return s.repo.GetBiasPatterns(ctx)
}

func (s *service) UpdateBiasPatterns(ctx context.Context, patterns map[string][]BiasPattern) error {
	return s.repo.UpdateBiasPatterns(ctx, patterns)
}

func (s *service) GetFactualityConfig(ctx context.Context) (FactualityConfig, error) {
	return s.repo.GetFactualityConfig(ctx)
}

func (s *service) UpdateFactualityConfig(ctx context.Context, config FactualityConfig) error {
	return s.repo.UpdateFactualityConfig(ctx, config)
}

func (s *service) GetAuditLogs(ctx context.Context, limit int) ([]AuditLog, error) {
	return s.repo.GetAuditLogs(ctx, limit)
}

func (s *service) ExportConfig(ctx context.Context) ([]byte, error) {
	return s.repo.ExportConfig(ctx)
}

func (s *service) ImportConfig(ctx context.Context, data []byte) error {
	return s.repo.ImportConfig(ctx, data)
}

// Model types
type Policy struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	Enabled     bool   `json:"enabled" db:"enabled"`
	Rules       string `json:"rules" db:"rules"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

type AuditLog struct {
	ID        string `json:"id" db:"id"`
	Action    string `json:"action" db:"action"`
	Resource  string `json:"resource" db:"resource"`
	UserID    string `json:"user_id" db:"user_id"`
	Timestamp string `json:"timestamp" db:"timestamp"`
	Details   string `json:"details" db:"details"`
}

// TLS Configuration helpers
func DefaultTLSConfig() TLSConfig {
	return TLSConfig{
		Enabled:            true,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	}
}

func InsecureTLSConfig() TLSConfig {
	return TLSConfig{
		Enabled:            false,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	}
}

func setTLSConfig(config *ServiceConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLS.InsecureSkipVerify,
	}

	if config.TLS.MinVersion != 0 {
		tlsConfig.MinVersion = config.TLS.MinVersion
	}

	if config.TLS.MaxVersion != 0 {
		tlsConfig.MaxVersion = config.TLS.MaxVersion
	}

	if config.TLS.CertFile != "" && config.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}
