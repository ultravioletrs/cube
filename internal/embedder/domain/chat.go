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
)

// ChatEvent is a single item in the streaming response sent to the client.
type ChatEvent struct {
	Type           ChatEventType `json:"type"`
	Content        string        `json:"content,omitempty"`
	Citations      []Citation    `json:"citations,omitempty"`
	Error          string        `json:"error,omitempty"`
	ConversationID string        `json:"conversation_id,omitempty"`
}

// ChatService orchestrates the full RAG pipeline for a query.
type ChatService interface {
	// Chat embeds the query, retrieves relevant chunks, calls the LLM, and
	// streams events to the returned channel.  The channel is closed when the
	// stream ends (either successfully or after an error event).
	Chat(ctx context.Context, userID string, messages []ChatMessage, recordIDs []string) (<-chan ChatEvent, error)
}
