// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func (r *ChunksRepository) KeywordSearchChunks(
	ctx context.Context,
	userID string,
	q domain.RetrievalQuery,
) ([]domain.ChunkMatch, error) {
	terms := searchTerms(q.Query)
	if len(terms) == 0 {
		return []domain.ChunkMatch{}, nil
	}

	args := []any{userID}
	next := 2

	var where []string
	where = append(where, "c.user_id = $1", "r.user_id = $1", "r.status = 'indexed'")

	if len(q.RecordIDs) > 0 {
		placeholders := make([]string, 0, len(q.RecordIDs))
		for _, id := range q.RecordIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			placeholders = append(placeholders, fmt.Sprintf("$%d", next))
			args = append(args, id)
			next++
		}
		if len(placeholders) > 0 {
			where = append(where, "r.id IN ("+strings.Join(placeholders, ",")+")")
		}
	}

	if len(q.SourceIDs) > 0 {
		placeholders := make([]string, 0, len(q.SourceIDs))
		for _, id := range q.SourceIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			placeholders = append(placeholders, fmt.Sprintf("$%d", next))
			args = append(args, id)
			next++
		}
		if len(placeholders) > 0 {
			where = append(where, "r.source_id IN ("+strings.Join(placeholders, ",")+")")
		}
	}

	likeExprs := make([]string, 0, len(terms))
	rankExprs := make([]string, 0, len(terms))
	for _, term := range terms {
		ph := fmt.Sprintf("$%d", next)
		next++
		args = append(args, "%"+term+"%")
		likeExprs = append(likeExprs, "c.content ILIKE "+ph)
		rankExprs = append(rankExprs, "CASE WHEN c.content ILIKE "+ph+" THEN 1 ELSE 0 END")
	}
	where = append(where, "("+strings.Join(likeExprs, " OR ")+")")

	limit := q.TopK
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}
	args = append(args, limit)
	limitPH := fmt.Sprintf("$%d", next)

	query := `
		SELECT c.id, r.id, r.name, r.format, s.id, s.name,
		       c.chunk_index, c.page_number, c.content
		FROM chunks c
		JOIN records r ON r.id = c.record_id
		LEFT JOIN sources s ON s.id = r.source_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY (` + strings.Join(rankExprs, " + ") + `) DESC, r.updated_at DESC, c.chunk_index ASC
		LIMIT ` + limitPH

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()

	matches := make([]domain.ChunkMatch, 0, limit)
	for rows.Next() {
		var (
			m          domain.ChunkMatch
			format     string
			sourceID   pgtype.Text
			sourceName pgtype.Text
			pageNumber pgtype.Int4
		)
		if err := rows.Scan(
			&m.ChunkID, &m.RecordID, &m.RecordName, &format,
			&sourceID, &sourceName, &m.ChunkIndex, &pageNumber, &m.Content,
		); err != nil {
			return nil, fmt.Errorf("scan chunk match: %w", err)
		}
		m.RecordFormat = domain.RecordFormat(format)
		if sourceID.Valid {
			m.SourceID = sourceID.String
		}
		if sourceName.Valid {
			m.SourceName = sourceName.String
		}
		if pageNumber.Valid {
			n := int(pageNumber.Int32)
			m.PageNumber = &n
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chunk matches: %w", err)
	}

	return matches, nil
}

func searchTerms(query string) []string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(fields) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(fields))
	terms := make([]string, 0, len(fields))
	for _, token := range fields {
		token = strings.Trim(token, ".,!?;:\"'`()[]{}")
		if len(token) < 2 {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		terms = append(terms, token)
		if len(terms) >= 8 {
			break
		}
	}
	return terms
}
