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
	domainID string,
	q domain.RetrievalQuery,
) ([]domain.ChunkMatch, error) {
	terms := searchTerms(q.Query)
	if len(terms) == 0 {
		return []domain.ChunkMatch{}, nil
	}

	args := []any{domainID}
	next := 2

	var where []string
	where = append(where, "c.domain_id = $1", "r.domain_id = $1", "r.status = 'indexed'")

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

// HybridSearchChunks combines vector similarity and keyword search via
// Reciprocal Rank Fusion (RRF).  When no meaningful keywords can be extracted
// from q.Query the keyword CTE is omitted and the result is pure vector search:
//
//	SELECT v.id AS chunk_id, 1.0 / (60.0 + v.rank) AS score
//	FROM vector_ranked v
func (r *ChunksRepository) HybridSearchChunks(
	ctx context.Context,
	domainID string,
	queryVec []float32,
	q domain.RetrievalQuery,
) ([]ChunkSearchResult, error) {
	topK := q.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 100 {
		topK = 100
	}
	innerLimit := topK * 5

	terms := searchTerms(q.Query)
	vec := float32SliceToPGVector(queryVec)

	// $1=domainID  $2=queryVec  $3=innerLimit
	args := []any{domainID, vec, innerLimit}
	next := 4

	// Optional record-ID filter shared by both CTEs.
	var recordIDFilter string
	if len(q.RecordIDs) > 0 {
		phs := make([]string, 0, len(q.RecordIDs))
		for _, id := range q.RecordIDs {
			if id = strings.TrimSpace(id); id == "" {
				continue
			}
			phs = append(phs, fmt.Sprintf("$%d", next))
			args = append(args, id)
			next++
		}
		if len(phs) > 0 {
			recordIDFilter = " AND c.record_id IN (" + strings.Join(phs, ",") + ")"
		}
	}

	var sb strings.Builder

	// Vector CTE — always present.
	sb.WriteString(`WITH vector_ranked AS (
    SELECT c.id, ROW_NUMBER() OVER (ORDER BY c.embedding <-> $2::vector) AS rank
    FROM chunks c
    JOIN records rec ON rec.id = c.record_id
    WHERE c.domain_id = $1 AND c.embedding IS NOT NULL AND rec.status = 'indexed'`)
	sb.WriteString(recordIDFilter)
	sb.WriteString(` LIMIT $3
)`)

	if len(terms) == 0 {
		// No meaningful words in query: degrade to pure vector scoring.
		sb.WriteString(`,
rrf AS (
    SELECT id AS chunk_id, 1.0 / (60.0 + rank::float) AS score
    FROM vector_ranked
)`)
	} else {
		// Keyword CTE — BM25-ranked via PostgreSQL full-text search.
		// websearch_to_tsquery with OR-joined terms gives recall even when a
		// query term (e.g. "built") is absent from the document. The GIN index
		// on to_tsvector('english', content) makes the @@ scan fast.
		orQuery := strings.Join(terms, " OR ")
		queryPH := fmt.Sprintf("$%d", next)
		next++
		args = append(args, orQuery)

		sb.WriteString(`,
keyword_ranked AS (
    SELECT id, ROW_NUMBER() OVER (ORDER BY fts_rank DESC, id) AS rank
    FROM (
        SELECT c.id,
               ts_rank_cd(to_tsvector('english', c.content),
                          websearch_to_tsquery('english', ` + queryPH + `)) AS fts_rank
        FROM chunks c
        JOIN records rec ON rec.id = c.record_id
        WHERE c.domain_id = $1
          AND rec.status = 'indexed'
          AND to_tsvector('english', c.content) @@ websearch_to_tsquery('english', ` + queryPH + `)`)
		sb.WriteString(recordIDFilter)
		sb.WriteString(`
        ORDER BY fts_rank DESC
        LIMIT $3
    ) t
),
rrf AS (
    SELECT COALESCE(v.id, k.id) AS chunk_id,
           COALESCE(1.0 / (60.0 + v.rank::float), 0.0)
         + COALESCE(1.0 / (60.0 + k.rank::float), 0.0) AS score
    FROM vector_ranked v
    FULL OUTER JOIN keyword_ranked k ON k.id = v.id
)`)
	}

	topKPH := fmt.Sprintf("$%d", next)
	args = append(args, topK)

	sb.WriteString(`
SELECT c.content, c.record_id, rec.name, COALESCE(rec.external_url, ''), c.chunk_index
FROM rrf
JOIN chunks c ON c.id = rrf.chunk_id
JOIN records rec ON rec.id = c.record_id
ORDER BY rrf.score DESC
LIMIT `)
	sb.WriteString(topKPH)

	rows, err := r.db.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("hybrid search chunks: %w", err)
	}
	defer rows.Close()

	var results []ChunkSearchResult
	for rows.Next() {
		var res ChunkSearchResult
		if err := rows.Scan(&res.Content, &res.RecordID, &res.RecordName, &res.ExternalURL, &res.ChunkIndex); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		results = append(results, res)
	}
	return results, rows.Err()
}

// searchTerms returns the meaningful tokens in query, used only to decide
// whether to include the FTS keyword CTE.  Stop-word removal and stemming are
// handled by plainto_tsquery inside the SQL query itself.
func searchTerms(query string) []string {
	fields := strings.Fields(strings.TrimSpace(query))
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
