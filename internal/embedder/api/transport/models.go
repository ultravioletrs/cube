// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// MountModels registers model-listing endpoints.
func MountModels(r chi.Router, ollamaBaseURL string) {
	r.Get("/api/v1/models/ollama", listOllamaModelsHandler(ollamaBaseURL))
}

// MountModelConnection registers the authenticated provider connection test.
func MountModelConnection(r chi.Router, ollamaBaseURL string) {
	r.Post("/api/v1/models/test-connection", testModelConnectionHandler(ollamaBaseURL))
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func listOllamaModelsHandler(baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get(baseURL + "/api/tags") //nolint:noctx
		if err != nil {
			writeJSON(w, http.StatusBadGateway, errBody("ollama unreachable: "+err.Error()))
			return
		}
		defer resp.Body.Close()

		var tags ollamaTagsResponse
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			writeJSON(w, http.StatusBadGateway, errBody("ollama response invalid: "+err.Error()))
			return
		}

		names := make([]string, 0, len(tags.Models))
		for _, m := range tags.Models {
			// Skip models that don't support chat (embeddings, code-completion).
			name := strings.ToLower(m.Name)
			if strings.Contains(name, "embed") || strings.Contains(name, "starcoder") {
				continue
			}
			names = append(names, m.Name)
		}

		writeJSON(w, http.StatusOK, map[string]any{"models": names})
	}
}

type modelConnectionRequest struct {
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
}

type modelConnectionResponse struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message"`
}

func testModelConnectionHandler(ollamaBaseURL string) http.HandlerFunc {
	client := &http.Client{Timeout: 20 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		var req modelConnectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
		req.Model = strings.TrimSpace(req.Model)
		if req.Model == "" {
			writeJSON(w, http.StatusBadRequest, errBody("model is required"))
			return
		}

		var err error
		switch req.Provider {
		case "ollama":
			err = testOllamaConnection(r.Context(), client, ollamaBaseURL, req.Model)
		case "openai":
			err = testOpenAICompatibleConnection(r.Context(), client, req.BaseURL, req.Model, req.APIKey)
		default:
			writeJSON(w, http.StatusBadRequest, errBody("unsupported provider"))
			return
		}
		if err != nil {
			writeJSON(w, http.StatusOK, modelConnectionResponse{Connected: false, Message: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, modelConnectionResponse{Connected: true, Message: "Connection successful"})
	}
}

func testOllamaConnection(ctx context.Context, client *http.Client, baseURL, model string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return contextError("Ollama unavailable", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusError("Ollama unavailable", resp.StatusCode)
	}

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return contextError("Invalid Ollama response", err)
	}
	for _, available := range tags.Models {
		if available.Name == model {
			return nil
		}
	}
	return &connectionError{message: "Model is not available in Ollama"}
}

func testOpenAICompatibleConnection(ctx context.Context, client *http.Client, baseURL, model, apiKey string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if !allowedExternalModelURL(baseURL) {
		return &connectionError{message: "Provider URL is not allowed"}
	}
	if strings.TrimSpace(apiKey) == "" {
		return &connectionError{message: "API key is required"}
	}

	body, err := json.Marshal(map[string]any{
		"model":      model,
		"messages":   []map[string]string{{"role": "user", "content": "Reply OK"}},
		"max_tokens": 1,
	})
	if err != nil {
		return contextError("Could not build provider request", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return contextError("Could not build provider request", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return contextError("Provider unavailable", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return &connectionError{message: "Invalid API key or insufficient permissions"}
		case http.StatusNotFound:
			return &connectionError{message: "Model or provider endpoint not found"}
		default:
			return statusError("Provider rejected the connection test", resp.StatusCode)
		}
	}
	return nil
}

func allowedExternalModelURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	switch parsed.Host {
	case "api.openai.com", "api.anthropic.com":
		return true
	default:
		return false
	}
}

type connectionError struct {
	message string
}

func (e *connectionError) Error() string {
	return e.message
}

func contextError(message string, _ error) error {
	return &connectionError{message: message}
}

func statusError(message string, status int) error {
	return &connectionError{message: message + " (HTTP " + http.StatusText(status) + ")"}
}
