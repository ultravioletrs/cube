// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5"
	kitendpoint "github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultravioletrs/cube/agent/audit"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/endpoint"
	"github.com/ultravioletrs/cube/proxy/router"
)

const ContentType = "application/json"

// ErrGuardrailsEvalFailed indicates that guardrails evaluation returned a non-OK status.
var ErrGuardrailsEvalFailed = errors.New("guardrails evaluation returned non-OK status")

// GuardrailsConfig holds configuration for the guardrails sidecar.
type GuardrailsConfig struct {
	Enabled  bool
	URL      string
	AgentURL string
}

type EvaluationResponse struct {
	Decision           string   `json:"decision"`
	Reason             string   `json:"reason"`
	GuardrailsResponse string   `json:"guardrails_response"`
	EvaluationTimeMs   float64  `json:"evaluation_time_ms"`
	TriggeredRails     []string `json:"triggered_rails"`
}

// ChatCompletionResponse represents an OpenAI-compatible chat completion response.
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *ChatCompletionUsage   `json:"usage,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int                    `json:"index"`
	Message      ChatCompletionMessage  `json:"message"`
	Delta        *ChatCompletionMessage `json:"delta,omitempty"`
	FinishReason string                 `json:"finish_reason"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatRequest represents the incoming chat request format.
type ChatRequest struct {
	Messages []ChatCompletionMessage `json:"messages"`
}

// GuardrailsEvalRequest is the format expected by the guardrails evaluation endpoints.
type GuardrailsEvalRequest struct {
	Messages []ChatCompletionMessage `json:"messages"`
}

// OllamaMessage represents a message in the Ollama chat format.
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaChatResponse represents an Ollama-compatible chat response.
type OllamaChatResponse struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            OllamaMessage `json:"message"`
	Done               bool          `json:"done"`
	DoneReason         string        `json:"done_reason,omitempty"`
	TotalDuration      int64         `json:"total_duration,omitempty"`
	LoadDuration       int64         `json:"load_duration,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64         `json:"prompt_eval_duration,omitempty"`
	EvalCount          int           `json:"eval_count,omitempty"`
	EvalDuration       int64         `json:"eval_duration,omitempty"`
}

type OllamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

func extractAssistantContentFromOllama(resp *OllamaChatResponse) string {
	return resp.Message.Content
}

func buildOutputEvalRequest(originalBody []byte, assistantContent string) *GuardrailsEvalRequest {
	var originalReq OllamaChatRequest
	if err := json.Unmarshal(originalBody, &originalReq); err != nil {
		log.Printf("buildOutputEvalRequest: Failed to parse original request: %v", err)

		return nil
	}

	messages := make([]ChatCompletionMessage, 0, len(originalReq.Messages)+1)
	for _, m := range originalReq.Messages {
		messages = append(messages, ChatCompletionMessage(m))
	}

	messages = append(messages, ChatCompletionMessage{
		Role:    "assistant",
		Content: assistantContent,
	})

	return &GuardrailsEvalRequest{
		Messages: messages,
	}
}

func MakeHandler(
	svc proxy.Service,
	instanceID string,
	auditSvc audit.Service,
	authn mgauthn.AuthNMiddleware,
	idp supermq.IDProvider,
	proxyTransport http.RoundTripper,
	rter *router.Router,
	guardrailsCfg GuardrailsConfig,
) http.Handler {
	endpoints := endpoint.MakeEndpoints(svc)

	mux := chi.NewRouter()

	mux.Get("/health", supermq.Health("cube-proxy", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	// Route management endpoints (public, requires authentication)
	mux.Post("/api/routes", authn.Middleware()(kithttp.NewServer(
		endpoints.CreateRoute,
		decodeCreateRouteRequest,
		encodeCreateRouteResponse,
	)).ServeHTTP)

	mux.Get("/api/routes", authn.Middleware()(kithttp.NewServer(
		endpoints.ListRoutes,
		decodeListRoutesRequest,
		encodeListRoutesResponse,
	)).ServeHTTP)

	mux.Get("/api/routes/{name}", authn.Middleware()(kithttp.NewServer(
		endpoints.GetRoute,
		decodeGetRouteRequest,
		encodeGetRouteResponse,
	)).ServeHTTP)

	mux.Put("/api/routes/{name}", authn.Middleware()(kithttp.NewServer(
		endpoints.UpdateRoute,
		decodeUpdateRouteRequest,
		encodeUpdateRouteResponse,
	)).ServeHTTP)

	mux.Delete("/api/routes/{name}", authn.Middleware()(kithttp.NewServer(
		endpoints.DeleteRoute,
		decodeDeleteRouteRequest,
		encodeDeleteRouteResponse,
	)).ServeHTTP)

	mux.Route("/{domainID}", func(r chi.Router) {
		r.Use(authn.Middleware(), api.RequestIDMiddleware(idp))
		r.Use(auditSvc.Middleware)

		r.Get("/attestation/policy", kithttp.NewServer(
			endpoints.GetAttestationPolicy,
			decodeGetAttestationPolicyRequest,
			encodeGetAttestationPolicyResponse,
		).ServeHTTP)

		// Chat completions with guardrails orchestration
		if guardrailsCfg.Enabled {
			r.Post("/api/chat", guardrailsHandler(
				endpoints.ProxyRequest,
				proxyTransport,
				guardrailsCfg,
				rter,
			))
		}

		// Proxy all other requests using the router
		r.Handle("/*", makeProxyHandler(endpoints.ProxyRequest, proxyTransport, rter))
	})

	mux.Post("/attestation/policy", kithttp.NewServer(
		endpoints.UpdateAttestationPolicy,
		decodeUpdateAttestationPolicyRequest,
		encodeUpdateAttestationPolicyResponse,
	).ServeHTTP)

	return mux
}

// makeProxyHandler creates a http.HandlerFunc that proxies requests.
func makeProxyHandler(
	proxyEndpoint kitendpoint.Endpoint, transport http.RoundTripper, rter *router.Router,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), proxy.MethodContextKey, r.Method)

		session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		domainID := chi.URLParam(r, "domainID")

		// Check authorization via endpoint
		req := endpoint.ProxyRequestRequest{
			Session:  session,
			DomainID: domainID,
			Path:     r.URL.Path,
		}

		if _, err := proxyEndpoint(ctx, req); err != nil {
			encodeError(ctx, err, w)

			return
		}

		// Determine target using router
		targetURL, stripPrefix, err := rter.DetermineTarget(r)
		if err != nil {
			log.Printf("Failed to determine target: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)

			return
		}

		target, err := url.Parse(targetURL)
		if err != nil {
			log.Printf("Invalid target URL %s: %v", targetURL, err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)

			return
		}

		prxy := httputil.NewSingleHostReverseProxy(target)
		prxy.Transport = transport

		prxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}

		originalDirector := prxy.Director
		prxy.Director = func(req *http.Request) {
			originalDirector(req)

			// Strip domainID prefix
			if domainID := chi.URLParam(req, "domainID"); domainID != "" {
				prefix := "/" + domainID

				req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
				if req.URL.RawPath != "" {
					req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, prefix)
				}
			}

			// Strip configured prefix
			if stripPrefix != "" {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, stripPrefix)
				if req.URL.RawPath != "" {
					req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, stripPrefix)
				}
			}
		}

		// Proceed to proxy
		prxy.ServeHTTP(w, r)
	}
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", ContentType)

	if errors.Is(err, errUnauthorized) {
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		w.WriteHeader(http.StatusForbidden) // Default to forbidden for auth errors
	}

	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": err.Error(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var errUnauthorized = errors.New("unauthorized")

const guardrailsTimeout = 30 * time.Minute

func guardrailsHandler(
	proxyEndpoint kitendpoint.Endpoint,
	transport http.RoundTripper,
	cfg GuardrailsConfig,
	rter *router.Router,
) http.HandlerFunc {
	client := &http.Client{
		Transport: transport,
		Timeout:   guardrailsTimeout,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		if _, err := proxyEndpoint(ctx, endpoint.ProxyRequestRequest{
			Session:  session,
			DomainID: chi.URLParam(r, "domainID"),
			Path:     r.URL.Path,
		}); err != nil {
			encodeError(ctx, err, w)

			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)

			return
		}

		r.Body.Close()

		guardrailsCtx, cancel := context.WithTimeout(ctx, guardrailsTimeout)
		defer cancel()

		// Evaluate input
		evalResp, err := evaluateGuardrails(guardrailsCtx, client, cfg.URL+"/guardrails/evaluate/input", bodyBytes)
		if err != nil {
			log.Printf("guardrailsHandler: Input evaluation failed: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)

			return
		}

		if evalResp.Decision == "BLOCK" {
			log.Printf("guardrailsHandler: Request BLOCKED by input rails. Triggered: %v", evalResp.TriggeredRails)
			writeGuardrailsBlockResponse(w, evalResp.GuardrailsResponse)

			return
		}

		// Forward to agent
		agentURL, err := buildAgentURL(r, rter)
		if err != nil {
			log.Printf("guardrailsHandler: Failed to build agent URL: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)

			return
		}

		agentBody, agentStatus, err := forwardRequest(ctx, client, agentURL, bodyBytes)
		if err != nil {
			log.Printf("guardrailsHandler: Failed to call Agent: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)

			return
		}

		if agentStatus >= 400 {
			log.Printf("guardrailsHandler: Agent returned error status %d, passing through", agentStatus)
			writeAgentResponse(w, agentStatus, agentBody)

			return
		}

		// Evaluate output
		outputEvalResp := evaluateOutput(guardrailsCtx, client, cfg.URL, bodyBytes, agentBody)
		if outputEvalResp != nil && outputEvalResp.Decision == "BLOCK" {
			log.Printf("guardrailsHandler: Response BLOCKED by output rails. Triggered: %v", outputEvalResp.TriggeredRails)
			writeGuardrailsBlockResponse(w, outputEvalResp.GuardrailsResponse)

			return
		}

		writeAgentResponse(w, agentStatus, agentBody)
	}
}

func evaluateGuardrails(
	ctx context.Context, client *http.Client, evalURL string, body []byte,
) (*EvaluationResponse, error) {
	evalBody, evalStatus, err := forwardRequest(ctx, client, evalURL, body)
	if err != nil {
		return nil, err
	}

	if evalStatus != http.StatusOK {
		return nil, ErrGuardrailsEvalFailed
	}

	var evalResp EvaluationResponse
	if err := json.Unmarshal(evalBody, &evalResp); err != nil {
		return nil, err
	}

	return &evalResp, nil
}

func buildAgentURL(r *http.Request, rter *router.Router) (string, error) {
	targetURL, stripPrefix, err := rter.DetermineTarget(r)
	if err != nil {
		return "", err
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}

	agentPath := r.URL.Path
	if domainID := chi.URLParam(r, "domainID"); domainID != "" {
		agentPath = strings.TrimPrefix(agentPath, "/"+domainID)
	}

	if stripPrefix != "" {
		agentPath = strings.TrimPrefix(agentPath, stripPrefix)
	}

	return target.String() + agentPath, nil
}

func evaluateOutput(
	ctx context.Context, client *http.Client, guardrailsURL string, requestBody, agentBody []byte,
) *EvaluationResponse {
	var agentResp OllamaChatResponse
	if err := json.Unmarshal(agentBody, &agentResp); err != nil {
		log.Printf("guardrailsHandler: Failed to parse agent response for output evaluation: %v", err)

		return nil
	}

	assistantContent := extractAssistantContentFromOllama(&agentResp)
	if assistantContent == "" {
		log.Printf("guardrailsHandler: No assistant content found, skipping output evaluation")

		return nil
	}

	outputEvalReq := buildOutputEvalRequest(requestBody, assistantContent)
	if outputEvalReq == nil {
		return nil
	}

	outputEvalBody, err := json.Marshal(outputEvalReq)
	if err != nil {
		log.Printf("guardrailsHandler: Failed to marshal output eval request: %v", err)

		return nil
	}

	evalResp, err := evaluateGuardrails(ctx, client, guardrailsURL+"/guardrails/evaluate/output", outputEvalBody)
	if err != nil {
		log.Printf("guardrailsHandler: Output evaluation failed: %v", err)

		return nil
	}

	return evalResp
}

func writeGuardrailsBlockResponse(w http.ResponseWriter, content string) {
	resp := OllamaChatResponse{
		Model:     "guardrails",
		CreatedAt: time.Now().Format(time.RFC3339),
		Message: OllamaMessage{
			Role:    "assistant",
			Content: content,
		},
		Done:       true,
		DoneReason: "stop",
	}

	w.Header().Set("Content-Type", ContentType)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("writeGuardrailsBlockResponse: Failed to encode response: %v", err)
	}
}

func writeAgentResponse(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func forwardRequest(
	ctx context.Context,
	client *http.Client,
	railsURL string,
	body []byte,
) (b []byte, status int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, railsURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", ContentType)

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return respBody, resp.StatusCode, nil
}
