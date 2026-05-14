// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

type sourcesRepo struct {
	pool *pgxpool.Pool
}

// NewSourcesRepository returns a PostgreSQL-backed SourceRepository.
func NewSourcesRepository(pool *pgxpool.Pool) domain.SourceRepository {
	return &sourcesRepo{pool: pool}
}

func (r *sourcesRepo) Create(ctx context.Context, s domain.Source) (domain.Source, error) {
	cfg := s.Config
	if cfg == nil {
		cfg = json.RawMessage("{}")
	}

	const q = `
		INSERT INTO sources (domain_id, user_id, source_type, name, config, status, sync_enabled, auto_sync_interval)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	var id string
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx, q,
		s.DomainID, s.UserID, string(s.Type), s.Name, []byte(cfg),
		string(s.Status), s.SyncEnabled, s.AutoSyncInterval,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Source{}, domain.ErrConflict
		}
		return domain.Source{}, fmt.Errorf("insert source: %w", err)
	}

	s.ID = id
	s.CreatedAt = createdAt
	s.UpdatedAt = updatedAt
	return s, nil
}

func (r *sourcesRepo) GetByID(ctx context.Context, id, domainID string) (domain.Source, error) {
	const q = `
		SELECT id, domain_id, user_id, source_type, name, config, status,
		       sync_enabled, auto_sync_interval,
		       last_sync_at, last_sync_error, next_sync_at,
		       created_at, updated_at
		FROM sources
		WHERE id = $1 AND domain_id = $2`

	row := r.pool.QueryRow(ctx, q, id, domainID)
	s, err := scanSource(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Source{}, domain.ErrNotFound
		}
		return domain.Source{}, fmt.Errorf("get source: %w", err)
	}
	return s, nil
}

func (r *sourcesRepo) List(ctx context.Context, domainID string, p domain.Page) (domain.SourcePage, error) {
	if domainID == "" {
		return domain.SourcePage{}, nil
	}
	const q = `
		SELECT id, domain_id, user_id, source_type, name, config, status,
		       sync_enabled, auto_sync_interval,
		       last_sync_at, last_sync_error, next_sync_at,
		       created_at, updated_at
		FROM sources
		WHERE domain_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	limit := p.Limit
	if limit == 0 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx, q, domainID, limit, p.Offset)
	if err != nil {
		return domain.SourcePage{}, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()

	var sources []domain.Source
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return domain.SourcePage{}, fmt.Errorf("scan source: %w", err)
		}
		sources = append(sources, s)
	}
	if err := rows.Err(); err != nil {
		return domain.SourcePage{}, fmt.Errorf("iterate sources: %w", err)
	}

	var total uint64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sources WHERE domain_id = $1`, domainID,
	).Scan(&total); err != nil {
		return domain.SourcePage{}, fmt.Errorf("count sources: %w", err)
	}

	return domain.SourcePage{Sources: sources, Total: total}, nil
}

func (r *sourcesRepo) Delete(ctx context.Context, id, domainID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM sources WHERE id = $1 AND domain_id = $2`, id, domainID)
	if err != nil {
		return fmt.Errorf("delete source: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *sourcesRepo) UpdateSyncResult(
	ctx context.Context,
	id, domainID string,
	status domain.SourceStatus,
	lastSyncAt time.Time,
	lastSyncError *string,
) (domain.Source, error) {
	const q = `
		UPDATE sources
		SET status = $1,
		    last_sync_at = $2,
		    last_sync_error = $3,
		    updated_at = now()
		WHERE id = $4 AND domain_id = $5
		RETURNING id, domain_id, user_id, source_type, name, config, status,
		          sync_enabled, auto_sync_interval,
		          last_sync_at, last_sync_error, next_sync_at,
		          created_at, updated_at`

	var errMsg pgtype.Text
	if lastSyncError != nil {
		errMsg = pgtype.Text{String: *lastSyncError, Valid: true}
	}

	row := r.pool.QueryRow(ctx, q, string(status), lastSyncAt, errMsg, id, domainID)
	src, err := scanSource(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Source{}, domain.ErrNotFound
		}
		return domain.Source{}, fmt.Errorf("update source sync result: %w", err)
	}
	return src, nil
}

func (r *sourcesRepo) UpdateConfig(
	ctx context.Context,
	id, domainID string,
	config json.RawMessage,
) (domain.Source, error) {
	const q = `
		UPDATE sources
		SET config = $1,
		    status = $2,
		    last_sync_error = NULL,
		    updated_at = now()
		WHERE id = $3 AND domain_id = $4
		RETURNING id, domain_id, user_id, source_type, name, config, status,
		          sync_enabled, auto_sync_interval,
		          last_sync_at, last_sync_error, next_sync_at,
		          created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, []byte(config), string(domain.SourceStatusActive), id, domainID)
	src, err := scanSource(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Source{}, domain.ErrNotFound
		}
		return domain.Source{}, fmt.Errorf("update source config: %w", err)
	}
	return src, nil
}

// scanSource scans a source row from either a pgx.Row or pgx.Rows.
func scanSource(row interface {
	Scan(dest ...any) error
},
) (domain.Source, error) {
	var (
		s             domain.Source
		sourceType    string
		status        string
		config        []byte
		lastSyncAt    pgtype.Timestamptz
		lastSyncError pgtype.Text
		nextSyncAt    pgtype.Timestamptz
	)
	if err := row.Scan(
		&s.ID, &s.DomainID, &s.UserID, &sourceType, &s.Name, &config, &status,
		&s.SyncEnabled, &s.AutoSyncInterval,
		&lastSyncAt, &lastSyncError, &nextSyncAt,
		&s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return domain.Source{}, err
	}

	s.Type = domain.SourceType(sourceType)
	s.Status = domain.SourceStatus(status)
	s.Config = json.RawMessage(config)

	if lastSyncAt.Valid {
		t := lastSyncAt.Time
		s.LastSyncAt = &t
	}
	if lastSyncError.Valid {
		s.LastSyncError = &lastSyncError.String
	}
	if nextSyncAt.Valid {
		t := nextSyncAt.Time
		s.NextSyncAt = &t
	}
	return s, nil
}
