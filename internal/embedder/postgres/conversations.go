// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

type conversationsRepo struct {
	pool *pgxpool.Pool
}

// NewConversationsRepository returns a PostgreSQL-backed ConversationRepository.
func NewConversationsRepository(pool *pgxpool.Pool) domain.ConversationRepository {
	return &conversationsRepo{pool: pool}
}

func (r *conversationsRepo) Create(ctx context.Context, domainID, userID, title string) (domain.Conversation, error) {
	var c domain.Conversation
	err := r.pool.QueryRow(ctx,
		`INSERT INTO conversations (domain_id, user_id, title) VALUES ($1, $2, $3)
		 RETURNING id, user_id, title, created_at, updated_at`,
		domainID, userID, title,
	).Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	c.DomainID = domainID
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	return c, nil
}

func (r *conversationsRepo) List(ctx context.Context, domainID string) ([]domain.Conversation, error) {
	if domainID == "" {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, domain_id, user_id, title, created_at, updated_at
		 FROM conversations WHERE domain_id = $1 ORDER BY updated_at DESC`,
		domainID,
	)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var convs []domain.Conversation
	for rows.Next() {
		var c domain.Conversation
		if err := rows.Scan(&c.ID, &c.DomainID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (r *conversationsRepo) Get(ctx context.Context, id, domainID string) (domain.Conversation, error) {
	var c domain.Conversation
	err := r.pool.QueryRow(ctx,
		`SELECT id, domain_id, user_id, title, created_at, updated_at
		 FROM conversations WHERE id = $1 AND domain_id = $2`,
		id, domainID,
	).Scan(&c.ID, &c.DomainID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Conversation{}, domain.ErrNotFound
		}
		return domain.Conversation{}, fmt.Errorf("get conversation: %w", err)
	}
	return c, nil
}

func (r *conversationsRepo) Delete(ctx context.Context, id, domainID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM conversations WHERE id = $1 AND domain_id = $2`, id, domainID,
	)
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *conversationsRepo) AppendMessages(ctx context.Context, conversationID string, msgs []domain.ConversationMessage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin append messages tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, m := range msgs {
		_, err := tx.Exec(ctx,
			`INSERT INTO conversation_messages (conversation_id, role, content) VALUES ($1, $2, $3)`,
			conversationID, m.Role, m.Content,
		)
		if err != nil {
			return fmt.Errorf("insert conversation message: %w", err)
		}
	}

	_, err = tx.Exec(ctx,
		`UPDATE conversations SET updated_at = now() WHERE id = $1`, conversationID,
	)
	if err != nil {
		return fmt.Errorf("update conversation timestamp: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *conversationsRepo) ListMessages(ctx context.Context, conversationID, domainID string) ([]domain.ConversationMessage, error) {
	// Verify domain ownership via join.
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.conversation_id, m.role, m.content, m.created_at
		 FROM conversation_messages m
		 JOIN conversations c ON c.id = m.conversation_id
		 WHERE m.conversation_id = $1 AND c.domain_id = $2
		 ORDER BY m.created_at`,
		conversationID, domainID,
	)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var msgs []domain.ConversationMessage
	for rows.Next() {
		var m domain.ConversationMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
