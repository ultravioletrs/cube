// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"time"
)

// Conversation is a persisted chat session belonging to a domain.
type Conversation struct {
	ID       string
	DomainID string
	UserID   string
	Title    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConversationMessage is a single turn stored inside a conversation.
type ConversationMessage struct {
	ID             string
	ConversationID string
	Role           string
	Content        string
	CreatedAt      time.Time
}

// ConversationRepository is the persistence interface for conversations.
type ConversationRepository interface {
	Create(ctx context.Context, domainID, userID, title string) (Conversation, error)
	List(ctx context.Context, domainID string) ([]Conversation, error)
	Get(ctx context.Context, id, domainID string) (Conversation, error)
	Delete(ctx context.Context, id, domainID string) error
	AppendMessages(ctx context.Context, conversationID string, msgs []ConversationMessage) error
	ListMessages(ctx context.Context, conversationID, domainID string) ([]ConversationMessage, error)
}
