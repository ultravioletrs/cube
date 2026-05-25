// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

// ChatMessage is a single turn in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"`
}

// Citation references a source chunk that grounded part of the answer.
type Citation struct {
	RecordID    string `json:"record_id"`
	RecordName  string `json:"record_name"`
	ExternalURL string `json:"external_url,omitempty"`
	ChunkIndex  int    `json:"chunk_index"`
	Excerpt     string `json:"excerpt"`
}

// ChatEventType identifies the kind of a streaming event.
type ChatEventType string

const (
	ChatEventToken        ChatEventType = "token"
	ChatEventCitations    ChatEventType = "citations"
	ChatEventError        ChatEventType = "error"
	ChatEventDone         ChatEventType = "done"
	ChatEventConversation ChatEventType = "conversation"
	// ChatEventWarning is emitted when retrieval degrades (e.g. embedding
	// service unreachable, no chunks found) so the client can surface it.
	ChatEventWarning ChatEventType = "warning"
)

// ChatEvent is a single item in the streaming response sent to the client.
type ChatEvent struct {
	Type           ChatEventType `json:"type"`
	Content        string        `json:"content,omitempty"`
	Citations      []Citation    `json:"citations,omitempty"`
	Error          string        `json:"error,omitempty"`
	ConversationID string        `json:"conversation_id,omitempty"`
}

// ModelConfig carries per-request LLM overrides sent by the client.
// Zero/empty values mean "use the server default".
type ModelConfig struct {
	// Provider selects the LLM backend: "ollama" or "openai".
	// Use "openai" for any OpenAI-compatible API (OpenAI, Anthropic, etc.).
	Provider string `json:"provider"`
	// BaseURL overrides the server-configured endpoint.
	// Leave empty to use the server default.
	BaseURL string `json:"base_url,omitempty"`
	// Model is the model identifier (e.g. "llama3.1:8b", "gpt-4o").
	Model string `json:"model"`
	// APIKey is required for OpenAI-compatible providers.
	// Never logged or persisted on the server.
	APIKey string `json:"api_key,omitempty"`
	// Temperature controls response randomness (0–1).
	Temperature float64 `json:"temperature"`
	// MaxTokens caps the response length (0 = provider default).
	MaxTokens int `json:"max_tokens,omitempty"`
}

// ChatService orchestrates the full RAG pipeline for a query.
type ChatService interface {
	// Chat embeds the query, retrieves relevant chunks, calls the LLM, and
	// streams events to the returned channel.  The channel is closed when the
	// stream ends (either successfully or after an error event).
	// modelCfg is optional; nil means use the server-configured default.
	Chat(ctx context.Context, domainID string, messages []ChatMessage, recordIDs []string, modelCfg *ModelConfig) (<-chan ChatEvent, error)
}
