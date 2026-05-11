package transport

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ultravioletrs/cube/internal/embedder/auth"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

// MountConversations registers conversation routes on the given router.
// All routes require an authenticated user (auth.Middleware must run first).
func MountConversations(r chi.Router, repo domain.ConversationRepository) {
	r.Get("/api/v1/conversations", listConversations(repo))
	r.Post("/api/v1/conversations", createConversation(repo))
	r.Get("/api/v1/conversations/{id}", getConversation(repo))
	r.Delete("/api/v1/conversations/{id}", deleteConversation(repo))
	r.Post("/api/v1/conversations/{id}/messages", appendMessages(repo))
}

type conversationResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type messageResponse struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
}

func toConversationResponse(c domain.Conversation) conversationResponse {
	return conversationResponse{
		ID:        c.ID,
		UserID:    c.UserID,
		Title:     c.Title,
		CreatedAt: c.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: c.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func toMessageResponse(m domain.ConversationMessage) messageResponse {
	return messageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           m.Role,
		Content:        m.Content,
		CreatedAt:      m.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func listConversations(repo domain.ConversationRepository) http.HandlerFunc {
	type response struct {
		Conversations []conversationResponse `json:"conversations"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())
		convs, err := repo.List(r.Context(), userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		resp := response{Conversations: make([]conversationResponse, 0, len(convs))}
		for _, c := range convs {
			resp.Conversations = append(resp.Conversations, toConversationResponse(c))
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func createConversation(repo domain.ConversationRepository) http.HandlerFunc {
	type request struct {
		Title string `json:"title"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		userID := auth.UserID(r.Context())
		conv, err := repo.Create(r.Context(), userID, req.Title)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		writeJSON(w, http.StatusCreated, toConversationResponse(conv))
	}
}

func getConversation(repo domain.ConversationRepository) http.HandlerFunc {
	type response struct {
		Conversation conversationResponse `json:"conversation"`
		Messages     []messageResponse    `json:"messages"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		userID := auth.UserID(r.Context())

		conv, err := repo.Get(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errBody("conversation not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		msgs, err := repo.ListMessages(r.Context(), id, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		resp := response{
			Conversation: toConversationResponse(conv),
			Messages:     make([]messageResponse, 0, len(msgs)),
		}
		for _, m := range msgs {
			resp.Messages = append(resp.Messages, toMessageResponse(m))
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func deleteConversation(repo domain.ConversationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		userID := auth.UserID(r.Context())

		if err := repo.Delete(r.Context(), id, userID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errBody("conversation not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func appendMessages(repo domain.ConversationRepository) http.HandlerFunc {
	type msgInput struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Messages []msgInput `json:"messages"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		userID := auth.UserID(r.Context())

		// Verify ownership before appending.
		if _, err := repo.Get(r.Context(), id, userID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errBody("conversation not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		if len(req.Messages) == 0 {
			writeJSON(w, http.StatusBadRequest, errBody("messages is required"))
			return
		}

		msgs := make([]domain.ConversationMessage, len(req.Messages))
		for i, m := range req.Messages {
			msgs[i] = domain.ConversationMessage{Role: m.Role, Content: m.Content}
		}

		if err := repo.AppendMessages(r.Context(), id, msgs); err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
