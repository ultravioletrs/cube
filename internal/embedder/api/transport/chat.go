// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ultravioletrs/cube/internal/embedder/auth"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

// MountChat registers the streaming chat endpoint.
func MountChat(r chi.Router, svc domain.ChatService, conversations domain.ConversationRepository) {
	r.Post("/api/v1/chat", chatHandler(svc, conversations))
}

type chatRequest struct {
	Messages       []domain.ChatMessage `json:"messages"`
	RecordIDs      []string             `json:"record_ids,omitempty"`
	ConversationID string               `json:"conversation_id,omitempty"`
	Model          *domain.ModelConfig  `json:"model,omitempty"`
}

func chatHandler(svc domain.ChatService, conversations domain.ConversationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())
		domainID := auth.DomainID(r.Context())

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		if len(req.Messages) == 0 {
			writeJSON(w, http.StatusBadRequest, errBody("messages is required"))
			return
		}

		// Ensure we have a conversation to save messages into.
		convID := req.ConversationID
		isNew := convID == ""
		if isNew && conversations != nil {
			title := conversationTitle(req.Messages)
			conv, err := conversations.Create(r.Context(), domainID, userID, title)
			if err == nil {
				convID = conv.ID
			} else {
				slog.Warn("create conversation failed", "err", err, "domain_id", domainID, "user_id", userID)
			}
		}

		// Persist the user messages that were sent.
		if convID != "" && conversations != nil {
			_ = conversations.AppendMessages(r.Context(), convID, toDomainMessages(req.Messages))
		}

		events, err := svc.Chat(r.Context(), domainID, req.Messages, req.RecordIDs, req.Model)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("chat failed: "+err.Error()))
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		flusher, canFlush := w.(http.Flusher)

		writeSSE := func(ev domain.ChatEvent) {
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if canFlush {
				flusher.Flush()
			}
		}

		// Tell the client the conversation ID so it can track/continue the chat.
		if convID != "" && isNew {
			writeSSE(domain.ChatEvent{Type: domain.ChatEventConversation, ConversationID: convID})
		}

		// Stream tokens and accumulate the assistant reply for persistence.
		var assistantContent string
		for ev := range events {
			if ev.Type == domain.ChatEventToken {
				assistantContent += ev.Content
			}
			writeSSE(ev)
			if ev.Type == domain.ChatEventDone || ev.Type == domain.ChatEventError {
				break
			}
		}

		// Persist the assistant reply.
		if convID != "" && assistantContent != "" && conversations != nil {
			_ = conversations.AppendMessages(r.Context(), convID, []domain.ConversationMessage{
				{Role: "assistant", Content: assistantContent},
			})
		}
	}
}

func conversationTitle(msgs []domain.ChatMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" && msgs[i].Content != "" {
			t := msgs[i].Content
			if len(t) > 80 {
				t = t[:80] + "…"
			}
			return t
		}
	}
	return "Untitled"
}

func toDomainMessages(msgs []domain.ChatMessage) []domain.ConversationMessage {
	out := make([]domain.ConversationMessage, len(msgs))
	for i, m := range msgs {
		out[i] = domain.ConversationMessage{Role: m.Role, Content: m.Content}
	}
	return out
}
