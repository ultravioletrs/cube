// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package guardrails

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/llm"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type validateMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type validateRequest struct {
	Messages []validateMessage `json:"messages"`
}

type validateResponse struct {
	Decision      string  `json:"decision"`
	Refusal       string  `json:"refusal"`
	ViolationType string  `json:"violation_type"`
	LatencyMs     float64 `json:"latency_ms"`
}

// Check validates messages against guardrails input filters.
// Returns allow=true if the input is safe, or allow=false with the refusal
// message if it was blocked. Returns an error if the service is unreachable.
func (c *Client) Check(ctx context.Context, messages []llm.Message) (allow bool, refusal string, err error) {
	msgs := make([]validateMessage, len(messages))
	for i, m := range messages {
		msgs[i] = validateMessage{Role: m.Role, Content: m.Content}
	}

	body, _ := json.Marshal(validateRequest{Messages: msgs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/guardrails/validate", bytes.NewReader(body))
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("guardrails validate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("guardrails validate status %d", resp.StatusCode)
	}

	var result validateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "", fmt.Errorf("guardrails decode: %w", err)
	}

	return result.Decision == "ALLOW", result.Refusal, nil
}

// GuardedClient wraps any llm.Client with guardrails input validation.
// Blocked messages are returned as a single token (the refusal text) without
// calling the inner LLM. Allowed messages pass through to the inner client.
// The enabled flag can be toggled at runtime without restarting.
type GuardedClient struct {
	inner   llm.Client
	checker *Client
	enabled atomic.Bool
}

// NewGuardedClient returns a GuardedClient with guardrails enabled by default.
func NewGuardedClient(inner llm.Client, checker *Client) *GuardedClient {
	gc := &GuardedClient{inner: inner, checker: checker}
	gc.enabled.Store(true)
	return gc
}

func (g *GuardedClient) IsEnabled() bool   { return g.enabled.Load() }
func (g *GuardedClient) SetEnabled(v bool) { g.enabled.Store(v) }

func (g *GuardedClient) StreamChat(ctx context.Context, messages []llm.Message, out chan<- string) error {
	if g.enabled.Load() {
		allow, refusal, err := g.checker.Check(ctx, messages)
		if err != nil {
			defer close(out)
			return fmt.Errorf("guardrails unavailable: %w", err)
		}
		if !allow {
			defer close(out)
			if refusal != "" {
				select {
				case out <- refusal:
				case <-ctx.Done():
				}
			}
			return nil
		}
	}
	return g.inner.StreamChat(ctx, messages, out)
}
