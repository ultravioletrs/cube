// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

type recordsRepo struct {
	pool *pgxpool.Pool
}

// NewRecordsRepository returns a PostgreSQL-backed RecordRepository.
func NewRecordsRepository(pool *pgxpool.Pool) domain.RecordRepository {
	return &recordsRepo{pool: pool}
}

func (r *recordsRepo) GetByID(ctx context.Context, id, domainID string) (domain.Record, error) {
	const q = `
		SELECT r.id, r.domain_id, r.user_id, r.source_id, r.name, r.format, r.status,
		       r.external_id, r.external_url, r.external_ref, r.mime_type,
		       r.description, r.chunk_count, r.size_bytes, r.page_count,
		       r.source_version, r.source_modified_at, r.error,
		       r.created_at, r.updated_at,
		       s.id, s.name, s.source_type, s.status
		FROM records r
		LEFT JOIN sources s ON s.id = r.source_id
		WHERE r.id = $1 AND r.domain_id = $2`

	row := r.pool.QueryRow(ctx, q, id, domainID)
	rec, err := scanRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Record{}, domain.ErrNotFound
		}
		return domain.Record{}, fmt.Errorf("get record: %w", err)
	}
	return rec, nil
}

func (r *recordsRepo) List(
	ctx context.Context,
	domainID string,
	f domain.RecordFilter,
	p domain.Page,
) (domain.RecordPage, error) {
	if domainID == "" {
		return domain.RecordPage{}, nil
	}
	args := []any{domainID}
	conds := []string{"r.domain_id = $1"}

	if f.SourceID != nil {
		args = append(args, *f.SourceID)
		conds = append(conds, fmt.Sprintf("r.source_id = $%d", len(args)))
	}
	if f.Status != nil {
		args = append(args, string(*f.Status))
		conds = append(conds, fmt.Sprintf("r.status = $%d", len(args)))
	}
	if f.Format != nil {
		args = append(args, string(*f.Format))
		conds = append(conds, fmt.Sprintf("r.format = $%d", len(args)))
	}

	where := strings.Join(conds, " AND ")

	limit := p.Limit
	if limit == 0 {
		limit = 20
	}

	q := fmt.Sprintf(`
		SELECT r.id, r.domain_id, r.user_id, r.source_id, r.name, r.format, r.status,
		       r.external_id, r.external_url, r.external_ref, r.mime_type,
		       r.description, r.chunk_count, r.size_bytes, r.page_count,
		       r.source_version, r.source_modified_at, r.error,
		       r.created_at, r.updated_at,
		       s.id, s.name, s.source_type, s.status
		FROM records r
		LEFT JOIN sources s ON s.id = r.source_id
		WHERE %s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d`,
		where, len(args)+1, len(args)+2,
	)
	args = append(args, limit, p.Offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return domain.RecordPage{}, fmt.Errorf("list records: %w", err)
	}
	defer rows.Close()

	var records []domain.Record
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return domain.RecordPage{}, fmt.Errorf("scan record: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return domain.RecordPage{}, fmt.Errorf("iterate records: %w", err)
	}

	countArgs := args[:len(args)-2] // strip LIMIT/OFFSET
	var total uint64
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM records r WHERE %s", where), countArgs...,
	).Scan(&total); err != nil {
		return domain.RecordPage{}, fmt.Errorf("count records: %w", err)
	}

	return domain.RecordPage{Records: records, Total: total}, nil
}

func scanRecord(row interface {
	Scan(dest ...any) error
},
) (domain.Record, error) {
	var (
		rec              domain.Record
		sourceID         pgtype.Text
		format           string
		status           string
		externalID       pgtype.Text
		externalURL      pgtype.Text
		externalRef      pgtype.Text
		mimeType         pgtype.Text
		description      pgtype.Text
		chunkCount       pgtype.Int4
		sizeBytes        pgtype.Int8
		pageCount        pgtype.Int4
		sourceVersion    pgtype.Text
		sourceModifiedAt pgtype.Timestamptz
		recError         pgtype.Text
		linkID           pgtype.Text
		linkName         pgtype.Text
		linkType         pgtype.Text
		linkStatus       pgtype.Text
	)
	if err := row.Scan(
		&rec.ID, &rec.DomainID, &rec.UserID, &sourceID, &rec.Name, &format, &status,
		&externalID, &externalURL, &externalRef, &mimeType,
		&description, &chunkCount, &sizeBytes, &pageCount,
		&sourceVersion, &sourceModifiedAt, &recError,
		&rec.CreatedAt, &rec.UpdatedAt,
		&linkID, &linkName, &linkType, &linkStatus,
	); err != nil {
		return domain.Record{}, err
	}

	rec.Format = domain.RecordFormat(format)
	rec.Status = domain.RecordStatus(status)

	if !sourceID.Valid || sourceID.String == "" {
		return domain.Record{}, fmt.Errorf("record %s is missing source_id", rec.ID)
	}
	rec.SourceID = sourceID.String
	if externalID.Valid {
		rec.ExternalID = externalID.String
	}
	if externalURL.Valid {
		rec.ExternalURL = externalURL.String
	}
	if externalRef.Valid {
		rec.ExternalRef = externalRef.String
	}
	if mimeType.Valid {
		rec.MimeType = mimeType.String
	}
	if description.Valid {
		rec.Description = description.String
	}
	if chunkCount.Valid {
		n := int(chunkCount.Int32)
		rec.ChunkCount = &n
	}
	if sizeBytes.Valid {
		n := sizeBytes.Int64
		rec.SizeBytes = &n
	}
	if pageCount.Valid {
		n := int(pageCount.Int32)
		rec.PageCount = &n
	}
	if sourceVersion.Valid {
		rec.SourceVersion = sourceVersion.String
	}
	if sourceModifiedAt.Valid {
		t := sourceModifiedAt.Time
		rec.SourceModifiedAt = &t
	}
	if recError.Valid {
		rec.Error = &recError.String
	}
	if linkID.Valid {
		rec.Source = &domain.RecordSourceLink{
			ID:     linkID.String,
			Name:   linkName.String,
			Type:   domain.SourceType(linkType.String),
			Status: domain.SourceStatus(linkStatus.String),
		}
	}
	return rec, nil
}

func (r *recordsRepo) Create(ctx context.Context, rec domain.Record) (domain.Record, error) {
	const q = `
		INSERT INTO records
			(domain_id, user_id, source_id, name, format, status,
			 external_id, external_url, external_ref, mime_type,
			 source_version, source_modified_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, domain_id, user_id, source_id, name, format, status,
		          external_id, external_url, external_ref, mime_type,
		          description, chunk_count, size_bytes, page_count,
		          source_version, source_modified_at, error,
		          created_at, updated_at,
		          NULL, NULL, NULL, NULL`

	var sourceModifiedAt pgtype.Timestamptz
	if rec.SourceModifiedAt != nil {
		sourceModifiedAt = pgtype.Timestamptz{Time: *rec.SourceModifiedAt, Valid: true}
	}

	row := r.pool.QueryRow(ctx, q,
		rec.DomainID, rec.UserID, rec.SourceID, rec.Name, string(rec.Format), string(rec.Status),
		rec.ExternalID, rec.ExternalURL, rec.ExternalRef, rec.MimeType,
		rec.SourceVersion, sourceModifiedAt,
	)
	created, err := scanRecord(row)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Record{}, domain.ErrConflict
		}
		return domain.Record{}, fmt.Errorf("create record: %w", err)
	}
	return created, nil
}

func (r *recordsRepo) ListQueued(ctx context.Context, limit int) ([]domain.Record, error) {
	const q = `
		SELECT r.id, r.domain_id, r.user_id, r.source_id, r.name, r.format, r.status,
		       r.external_id, r.external_url, r.external_ref, r.mime_type,
		       r.description, r.chunk_count, r.size_bytes, r.page_count,
		       r.source_version, r.source_modified_at, r.error,
		       r.created_at, r.updated_at,
		       s.id, s.name, s.source_type, s.status
		FROM records r
		LEFT JOIN sources s ON s.id = r.source_id
		WHERE r.status = 'queued'
		ORDER BY r.created_at
		LIMIT $1`

	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("list queued: %w", err)
	}
	defer rows.Close()

	var records []domain.Record
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan queued record: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *recordsRepo) UpdateStatus(ctx context.Context, id string, s domain.RecordStatus, errMsg string) error {
	if errMsg != "" {
		_, err := r.pool.Exec(ctx,
			`UPDATE records SET status=$1, error=$2, updated_at=now() WHERE id=$3 AND status <> 'cancelled'`,
			string(s), errMsg, id,
		)
		return err
	}
	if s == domain.RecordStatusCancelled || s == domain.RecordStatusQueued {
		_, err := r.pool.Exec(ctx,
			`UPDATE records SET status=$1, error=NULL, updated_at=now() WHERE id=$2`,
			string(s), id,
		)
		return err
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE records SET status=$1, error=NULL, updated_at=now() WHERE id=$2 AND status <> 'cancelled'`,
		string(s), id,
	)
	return err
}

func (r *recordsRepo) UpdateAfterIngest(ctx context.Context, id string, res domain.IngestResult) error {
	var pageCount pgtype.Int4
	if res.PageCount != nil {
		pageCount = pgtype.Int4{Int32: int32(*res.PageCount), Valid: true}
	}

	_, err := r.pool.Exec(ctx,
		`UPDATE records
		 SET status='indexed',
		     chunk_count=$1,
		     size_bytes=$2,
		     page_count=$3,
		     description=COALESCE(NULLIF($4, ''), description),
		     error=NULL,
		     updated_at=now()
		 WHERE id=$5 AND status <> 'cancelled'`,
		res.ChunkCount, res.SizeBytes, pageCount, res.Description, id,
	)
	return err
}

func (r *recordsRepo) UpsertFromSource(ctx context.Context, rec domain.Record) (domain.RecordUpsertResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.RecordUpsertResult{}, fmt.Errorf("begin record upsert tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const selectQ = `
		SELECT r.id, r.domain_id, r.user_id, r.source_id, r.name, r.format, r.status,
		       r.external_id, r.external_url, r.external_ref, r.mime_type,
		       r.description, r.chunk_count, r.size_bytes, r.page_count,
		       r.source_version, r.source_modified_at, r.error,
		       r.created_at, r.updated_at,
		       s.id, s.name, s.source_type, s.status
		FROM records r
		LEFT JOIN sources s ON s.id = r.source_id
		WHERE r.domain_id = $1 AND r.source_id = $2 AND r.external_id = $3
		FOR UPDATE OF r`

	existing, err := scanRecord(tx.QueryRow(ctx, selectQ, rec.DomainID, rec.SourceID, rec.ExternalID))
	switch {
	case err == nil:
	case errors.Is(err, pgx.ErrNoRows):
		created, err := r.createInTx(ctx, tx, rec)
		if err != nil {
			return domain.RecordUpsertResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.RecordUpsertResult{}, fmt.Errorf("commit record upsert create: %w", err)
		}
		return domain.RecordUpsertResult{Record: created, State: domain.RecordUpsertCreated}, nil
	default:
		return domain.RecordUpsertResult{}, fmt.Errorf("select record for upsert: %w", err)
	}

	needsRequeue := existing.SourceVersion != rec.SourceVersion || existing.Status != domain.RecordStatusIndexed
	if !needsRequeue {
		rec.Status = existing.Status
		rec.ChunkCount = existing.ChunkCount
		rec.SizeBytes = existing.SizeBytes
		rec.PageCount = existing.PageCount
	}
	updated, err := r.updateFromSourceInTx(ctx, tx, existing.ID, rec, needsRequeue)
	if err != nil {
		return domain.RecordUpsertResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RecordUpsertResult{}, fmt.Errorf("commit record upsert update: %w", err)
	}

	state := domain.RecordUpsertUnchanged
	if needsRequeue {
		state = domain.RecordUpsertUpdated
	}
	return domain.RecordUpsertResult{Record: updated, State: state}, nil
}

func (r *recordsRepo) createInTx(ctx context.Context, tx pgx.Tx, rec domain.Record) (domain.Record, error) {
	const q = `
		INSERT INTO records
			(domain_id, user_id, source_id, name, format, status,
			 external_id, external_url, external_ref, mime_type,
			 source_version, source_modified_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, domain_id, user_id, source_id, name, format, status,
		          external_id, external_url, external_ref, mime_type,
		          description, chunk_count, size_bytes, page_count,
		          source_version, source_modified_at, error,
		          created_at, updated_at,
		          NULL, NULL, NULL, NULL`

	var sourceModifiedAt pgtype.Timestamptz
	if rec.SourceModifiedAt != nil {
		sourceModifiedAt = pgtype.Timestamptz{Time: *rec.SourceModifiedAt, Valid: true}
	}

	created, err := scanRecord(tx.QueryRow(ctx, q,
		rec.DomainID, rec.UserID, rec.SourceID, rec.Name, string(rec.Format), string(rec.Status),
		rec.ExternalID, rec.ExternalURL, rec.ExternalRef, rec.MimeType,
		rec.SourceVersion, sourceModifiedAt,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Record{}, domain.ErrConflict
		}
		return domain.Record{}, fmt.Errorf("create record: %w", err)
	}
	return created, nil
}

func (r *recordsRepo) updateFromSourceInTx(
	ctx context.Context,
	tx pgx.Tx,
	id string,
	rec domain.Record,
	requeue bool,
) (domain.Record, error) {
	status := rec.Status
	if status == "" {
		status = domain.RecordStatusQueued
	}

	const baseQ = `
		UPDATE records
		SET name = $1,
		    format = $2,
		    external_url = $3,
		    external_ref = $4,
		    mime_type = $5,
		    source_version = $6,
		    source_modified_at = $7,
		    status = $8,
		    error = NULL,
		    chunk_count = $9,
		    size_bytes = $10,
		    page_count = $11,
		    updated_at = now()
		WHERE id = $12
		RETURNING id, domain_id, user_id, source_id, name, format, status,
		          external_id, external_url, external_ref, mime_type,
		          description, chunk_count, size_bytes, page_count,
		          source_version, source_modified_at, error,
		          created_at, updated_at,
		          NULL, NULL, NULL, NULL`

	var sourceModifiedAt pgtype.Timestamptz
	if rec.SourceModifiedAt != nil {
		sourceModifiedAt = pgtype.Timestamptz{Time: *rec.SourceModifiedAt, Valid: true}
	}

	var chunkCount pgtype.Int4
	var sizeBytes pgtype.Int8
	var pageCount pgtype.Int4
	nextStatus := existingOrQueuedStatus(requeue, status)
	if !requeue && rec.ChunkCount != nil {
		chunkCount = pgtype.Int4{Int32: int32(*rec.ChunkCount), Valid: true}
	}
	if !requeue && rec.SizeBytes != nil {
		sizeBytes = pgtype.Int8{Int64: *rec.SizeBytes, Valid: true}
	}
	if !requeue && rec.PageCount != nil {
		pageCount = pgtype.Int4{Int32: int32(*rec.PageCount), Valid: true}
	}

	updated, err := scanRecord(tx.QueryRow(ctx, baseQ,
		rec.Name, string(rec.Format), rec.ExternalURL, rec.ExternalRef, rec.MimeType,
		rec.SourceVersion, sourceModifiedAt, string(nextStatus),
		chunkCount, sizeBytes, pageCount, id,
	))
	if err != nil {
		return domain.Record{}, fmt.Errorf("update record from source: %w", err)
	}
	return updated, nil
}

func existingOrQueuedStatus(requeue bool, current domain.RecordStatus) domain.RecordStatus {
	if requeue {
		return domain.RecordStatusQueued
	}
	if current == "" {
		return domain.RecordStatusIndexed
	}
	return current
}

func (r *recordsRepo) DeleteBySourceExternalIDs(
	ctx context.Context,
	domainID, sourceID string,
	externalIDs []string,
) (int, error) {
	if len(externalIDs) == 0 {
		return 0, nil
	}

	uniq := make([]string, 0, len(externalIDs))
	seen := make(map[string]struct{}, len(externalIDs))
	for _, externalID := range externalIDs {
		externalID = strings.TrimSpace(externalID)
		if externalID == "" {
			continue
		}
		if _, ok := seen[externalID]; ok {
			continue
		}
		seen[externalID] = struct{}{}
		uniq = append(uniq, externalID)
	}
	if len(uniq) == 0 {
		return 0, nil
	}

	tag, err := r.pool.Exec(ctx,
		`DELETE FROM records
		 WHERE domain_id = $1 AND source_id = $2 AND external_id = ANY($3)`,
		domainID, sourceID, uniq,
	)
	if err != nil {
		return 0, fmt.Errorf("delete records by source external IDs: %w", err)
	}

	return int(tag.RowsAffected()), nil
}

func (r *recordsRepo) Delete(ctx context.Context, id, domainID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM records WHERE id = $1 AND domain_id = $2`, id, domainID,
	)
	if err != nil {
		return fmt.Errorf("delete record: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// isUniqueViolation checks for PostgreSQL unique-constraint error (code 23505).
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505")
}
