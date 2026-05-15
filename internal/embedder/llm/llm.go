// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package llm

import "context"

// Message is a single turn in a chat conversation.
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// Client streams chat completions from an LLM provider.
type Client interface {
	// StreamChat sends messages to the model and writes tokens to out.
	// It closes out when the stream ends (success or error).
	StreamChat(ctx context.Context, messages []Message, out chan<- string) error
}

// Config describes how to connect to an LLM provider.
type Config struct {
	Provider    string  // "openai" | "ollama"
	BaseURL     string
	Model       string
	APIKey      string  // required for openai
	Temperature float64 // 0 = provider default
	MaxTokens   int     // 0 = provider default
}

// ClientFactory builds an LLM client from a Config.
// Injected at startup to avoid circular imports between the llm and
// concrete provider packages.
type ClientFactory func(cfg Config) Client
