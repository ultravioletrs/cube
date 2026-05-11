// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ultravioletrs/cube/internal/embedder/llm"
)

// Client streams chat completions from an Ollama server.
type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

// New returns an Ollama chat streaming client.
func New(baseURL, model string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: 0}, // no timeout — streaming
	}
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type chatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// StreamChat sends messages to Ollama and writes tokens to out.
func (c *Client) StreamChat(ctx context.Context, messages []llm.Message, out chan<- string) error {
	defer close(out)

	msgs := make([]ollamaMessage, len(messages))
	for i, m := range messages {
		msgs[i] = ollamaMessage{Role: m.Role, Content: m.Content}
	}

	body, _ := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: msgs,
		Stream:   true,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ollama chat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama chat status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var line chatResponse
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Message.Content != "" {
			select {
			case out <- line.Message.Content:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if line.Done {
			break
		}
	}
	return scanner.Err()
}
