// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	nemoContentTypeJSON = "application/json"
)

var (
	ErrWebhookSecretNotConfigured = errors.New("webhook secret not configured")
	ErrHMACWriteFailed            = errors.New("failed to write to HMAC")
	ErrConfigurationPushFailed    = errors.New("configuration push failed")
	ErrServerError                = errors.New("server error")
	ErrRequestFailed              = errors.New("request failed after retries")
)

type NeMoGuardrailsSDK interface {
	ChatCompletion(ctx context.Context, request ChatCompletionRequest) (*ChatCompletionResponse, error)

	ProcessRequest(ctx context.Context, body []byte, headers map[string]string) (*ProcessResponse, error)

	PushConfiguration(ctx context.Context, config ConfigurationPush) error
}

type ChatCompletionRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	UserID      string        `json:"-"` // Set from headers, not sent in body
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   Usage        `json:"usage"`
}

type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ProcessResponse struct {
	Body    []byte
	Headers http.Header
	Status  int
}

type RequestContext struct {
	RequestID string `json:"request_id"`
	UserID    string `json:"user_id,omitempty"`
}

type ConfigurationPush struct {
	BaseConfig    map[string]interface{} `json:"base_config"`
	FlowsConfig   map[string]interface{} `json:"flows_config"`
	KnowledgeBase []KBFile               `json:"knowledge_base"`
	Timestamp     int64                  `json:"timestamp"`
	Version       string                 `json:"version"`
}

type KBFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Type    string `json:"type"` // e.g., "markdown", "text"
}

type ConfigFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Path    string `json:"path"`
	Type    string `json:"type"` // e.g., "colang", "yaml", "python"
}

type NeMoConfig struct {
	BaseURL         string
	Timeout         time.Duration
	TLSVerification bool
	MaxRetries      int
	WebhookSecret   string
}

type nemoSDK struct {
	baseURL       string
	client        *http.Client
	maxRetries    int
	webhookSecret string
}

func NewNeMoGuardrailsSDK(conf NeMoConfig) NeMoGuardrailsSDK {
	if conf.Timeout == 0 {
		conf.Timeout = 30 * time.Second
	}
	if conf.MaxRetries == 0 {
		conf.MaxRetries = 3
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !conf.TLSVerification,
		},
	}

	webhookSecret := conf.WebhookSecret
	if webhookSecret == "" {
		webhookSecret = os.Getenv("NEMO_WEBHOOK_SECRET")
	}
	if webhookSecret == "" {
		webhookSecret = "default-secret-change-in-production"
	}

	return &nemoSDK{
		baseURL:       conf.BaseURL,
		maxRetries:    conf.MaxRetries,
		webhookSecret: webhookSecret,
		client: &http.Client{
			Timeout:   conf.Timeout,
			Transport: otelhttp.NewTransport(transport),
		},
	}
}

func (sdk *nemoSDK) ChatCompletion(ctx context.Context, request ChatCompletionRequest) (*ChatCompletionResponse, error) {
	url := fmt.Sprintf("%s/v1/chat/completions", sdk.baseURL)

	headers := map[string]string{
		"Content-Type": nemoContentTypeJSON,
	}
	if request.UserID != "" {
		headers["X-User-ID"] = request.UserID
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.sendWithRetry(ctx, "POST", url, bytes.NewReader(requestBody), headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var chatResponse ChatCompletionResponse
	if err := json.Unmarshal(responseBody, &chatResponse); err != nil {
		return nil, err
	}

	return &chatResponse, nil
}

func (sdk *nemoSDK) ProcessRequest(ctx context.Context, body []byte, headers map[string]string) (*ProcessResponse, error) {
	url := fmt.Sprintf("%s/v1/chat/completions", sdk.baseURL)

	if headers == nil {
		headers = make(map[string]string)
	}

	if _, exists := headers["Content-Type"]; !exists {
		headers["Content-Type"] = nemoContentTypeJSON
	}

	resp, err := sdk.sendWithRetry(ctx, "POST", url, bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &ProcessResponse{
		Body:    responseBody,
		Headers: resp.Header,
		Status:  resp.StatusCode,
	}, nil
}

func (sdk *nemoSDK) PushConfiguration(ctx context.Context, config ConfigurationPush) error {
	url := fmt.Sprintf("%s:8080/webhook/config", sdk.baseURL) // Use webhook port

	if config.Timestamp == 0 {
		config.Timestamp = time.Now().Unix()
	}

	configBody, err := json.Marshal(config)
	if err != nil {
		return err
	}

	signature, err := sdk.generateHMACSignature(configBody)
	if err != nil {
		return err
	}

	headers := map[string]string{
		"Content-Type":     nemoContentTypeJSON,
		"X-Config-Version": config.Version,
		"X-Signature":      fmt.Sprintf("sha256=%s", signature),
		"X-Timestamp":      fmt.Sprintf("%d", config.Timestamp),
	}

	resp, err := sdk.sendWithRetry(ctx, "POST", url, bytes.NewReader(configBody), headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		responseBody, _ := io.ReadAll(resp.Body)
		return errors.Wrap(ErrConfigurationPushFailed, fmt.Errorf("status %d: %s", resp.StatusCode, string(responseBody)))
	}

	return nil
}

func (sdk *nemoSDK) sendWithRetry(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	var lastErr error
	var bodyBytes []byte

	if body != nil {
		var err error

		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i <= sdk.maxRetries; i++ {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, err
		}

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := sdk.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if err != nil {
			lastErr = err
		} else {
			resp.Body.Close()
			lastErr = errors.Wrap(ErrServerError, fmt.Errorf("%d", resp.StatusCode))
		}

		if i < sdk.maxRetries {
			waitTime := time.Duration(i+1) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
				// Continue to next retry
			}
		}
	}

	return nil, errors.Wrap(ErrRequestFailed, fmt.Errorf("%d retries: %w", sdk.maxRetries, lastErr))
}

func (sdk *nemoSDK) generateHMACSignature(body []byte) (string, error) {
	if sdk.webhookSecret == "" {
		return "", ErrWebhookSecretNotConfigured
	}

	h := hmac.New(sha256.New, []byte(sdk.webhookSecret))
	_, err := h.Write(body)

	if err != nil {
		return "", errors.Wrap(ErrHMACWriteFailed, err)
	}

	signature := hex.EncodeToString(h.Sum(nil))
	return signature, nil
}
