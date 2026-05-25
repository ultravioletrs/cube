// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ultravioletrs/cube/internal/embedder/llm"
)

// Client streams chat completions from the OpenAI API.
type Client struct {
	baseURL     string
	model       string
	apiKey      string
	temperature float64
	maxTokens   int
	http        *http.Client
}

// New returns an OpenAI chat streaming client.
func New(baseURL, model, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 0}, // no timeout — streaming
	}
}

// NewFromConfig returns an OpenAI client configured from an llm.Config.
func NewFromConfig(cfg llm.Config) *Client {
	return &Client{
		baseURL:     cfg.BaseURL,
		model:       cfg.Model,
		apiKey:      cfg.APIKey,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		http:        &http.Client{Timeout: 0},
	}
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type streamRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// StreamChat sends messages to OpenAI and writes tokens to out.
func (c *Client) StreamChat(ctx context.Context, messages []llm.Message, out chan<- string) error {
	defer close(out)

	msgs := make([]openAIMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	body, err := json.Marshal(streamRequest{
		Model:       c.model,
		Messages:    msgs,
		Stream:      true,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
	})
	if err != nil {
		return fmt.Errorf("openai chat marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("openai chat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openai chat status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			token := chunk.Choices[0].Delta.Content
			if token != "" {
				select {
				case out <- token:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
	return scanner.Err()
}
