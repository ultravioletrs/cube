// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	smqErrors "github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/ultraviolet/cube/guardrails"
)

var _ guardrails.Repository = (*repository)(nil)

type repository struct {
	db postgres.Database
}

func (r *repository) CreateFlow(ctx context.Context, flow *guardrails.Flow) error {
	query := `INSERT INTO flows (id, name, description, content, type, active, version, 
		created_at, updated_at) VALUES (:id, :name, :description, :content, :type, 
		:active, :version, :created_at, :updated_at)`

	createdAt := time.Now()
	updatedAt := createdAt

	_, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":          flow.ID,
		"name":        flow.Name,
		"description": flow.Description,
		"content":     flow.Content,
		"type":        flow.Type,
		"active":      flow.Active,
		"version":     flow.Version,
		"created_at":  createdAt,
		"updated_at":  updatedAt,
	})
	if err != nil {
		return smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	return nil
}

func (r *repository) GetFlow(ctx context.Context, id string) (*guardrails.Flow, error) {
	flow := &guardrails.Flow{
		ID: id,
	}
	query := `SELECT id, name, description, content, type, active, version, 
		created_at, updated_at FROM flows WHERE id = $1`

	if err := r.db.QueryRowxContext(ctx, query, id).StructScan(flow); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, guardrails.ErrNotFound
		}

		return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	return flow, nil
}

func (r *repository) GetFlows(ctx context.Context, pm *guardrails.PageMetadata) ([]*guardrails.Flow, error) {
	oq := getOrderQuery(pm.Order)
	dq := getDirQuery(pm.Dir)

	args := []interface{}{}
	argIdx := 1

	query := `SELECT id, name, description, content, type, active, version, 
		created_at, updated_at FROM flows WHERE active = true`

	if pm.Name != "" {
		query += fmt.Sprintf(" AND type = $%d", argIdx)

		args = append(args, pm.Name)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY %s %s", oq, dq)

	if pm.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)

		args = append(args, pm.Limit)
		argIdx++
	}

	if pm.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)

		args = append(args, pm.Offset)
	}

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}
	defer rows.Close()

	var flows []*guardrails.Flow
	for rows.Next() {
		var flow guardrails.Flow
		if err := rows.StructScan(&flow); err != nil {
			return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
		}

		flows = append(flows, &flow)
	}

	if err := rows.Err(); err != nil {
		return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	return flows, nil
}

func (r *repository) UpdateFlow(ctx context.Context, flow *guardrails.Flow) error {
	query := `UPDATE flows SET name = :name, description = :description, 
		content = :content, type = :type, active = :active WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":          flow.ID,
		"name":        flow.Name,
		"description": flow.Description,
		"content":     flow.Content,
		"type":        flow.Type,
		"active":      flow.Active,
	})
	if err != nil {
		return smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	return nil
}

func (r *repository) DeleteFlow(ctx context.Context, id string) error {
	query := `DELETE FROM flows WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return guardrails.ErrNotFound
	}

	return nil
}

func (r *repository) CreateKBFile(ctx context.Context, file *guardrails.KBFile) error {
	query := `INSERT INTO kb_files (id, name, content, type, category, tags, 
		metadata, active, version, created_at, updated_at) VALUES (:id, :name, 
		:content, :type, :category, :tags, :metadata, :active, :version, 
		:created_at, :updated_at)`

	createdAt := time.Now()
	updatedAt := createdAt

	tagsArray := "{" + stringSliceToPostgresArray(file.Tags) + "}"

	metadataJSON, err := json.Marshal(file.Metadata)
	if err != nil {
		return smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	_, err = r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":         file.ID,
		"name":       file.Name,
		"content":    file.Content,
		"type":       file.Type,
		"category":   file.Category,
		"tags":       tagsArray,
		"metadata":   metadataJSON,
		"active":     file.Active,
		"version":    file.Version,
		"created_at": createdAt,
		"updated_at": updatedAt,
	})
	if err != nil {
		return smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	return nil
}

func (r *repository) GetKBFile(ctx context.Context, id string) (*guardrails.KBFile, error) {
	var file struct {
		ID        string          `db:"id"`
		Name      string          `db:"name"`
		Content   string          `db:"content"`
		Type      string          `db:"type"`
		Category  string          `db:"category"`
		Tags      sql.NullString  `db:"tags"`
		Metadata  json.RawMessage `db:"metadata"`
		Active    bool            `db:"active"`
		Version   int             `db:"version"`
		CreatedAt time.Time       `db:"created_at"`
		UpdatedAt time.Time       `db:"updated_at"`
	}

	query := `SELECT id, name, content, type, category, 
		array_to_string(tags, ',') as tags, metadata, active, version, 
		created_at, updated_at FROM kb_files WHERE id = $1`

	if err := r.db.QueryRowxContext(ctx, query, id).StructScan(&file); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, guardrails.ErrNotFound
		}

		return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	var tags []string
	if file.Tags.Valid && file.Tags.String != "" {
		tags = stringToSlice(file.Tags.String)
	}

	var metadata map[string]interface{}
	if len(file.Metadata) > 0 {
		if err := json.Unmarshal(file.Metadata, &metadata); err != nil {
			return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
		}
	}

	return &guardrails.KBFile{
		ID:        file.ID,
		Name:      file.Name,
		Content:   file.Content,
		Type:      file.Type,
		Category:  file.Category,
		Tags:      tags,
		Metadata:  metadata,
		Active:    file.Active,
		Version:   file.Version,
		CreatedAt: file.CreatedAt.Format(time.RFC3339),
		UpdatedAt: file.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (r *repository) GetKBFiles(ctx context.Context, pm *guardrails.PageMetadata) ([]*guardrails.KBFile, error) {
	oq := getOrderQuery(pm.Order)
	dq := getDirQuery(pm.Dir)

	args := []interface{}{}
	argIdx := 1

	query := `SELECT id, name, content, type, category, 
		array_to_string(tags, ',') as tags, metadata, active, version, 
		created_at, updated_at FROM kb_files WHERE active = true`

	if pm.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIdx)

		args = append(args, pm.Category)
		argIdx++
	}

	if pm.User != "" {
		tags := strings.Split(pm.User, ",")
		if len(tags) > 0 {
			query += fmt.Sprintf(" AND tags && $%d", argIdx)
			tagsArray := "{" + stringSliceToPostgresArray(tags) + "}"
			args = append(args, tagsArray)
			argIdx++
		}
	}

	query += fmt.Sprintf(" ORDER BY %s %s", oq, dq)

	if pm.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)

		args = append(args, pm.Limit)
		argIdx++
	}

	if pm.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)

		args = append(args, pm.Offset)
	}

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}
	defer rows.Close()

	var kbFiles []*guardrails.KBFile

	for rows.Next() {
		var file struct {
			ID        string          `db:"id"`
			Name      string          `db:"name"`
			Content   string          `db:"content"`
			Type      string          `db:"type"`
			Category  string          `db:"category"`
			Tags      sql.NullString  `db:"tags"`
			Metadata  json.RawMessage `db:"metadata"`
			Active    bool            `db:"active"`
			Version   int             `db:"version"`
			CreatedAt time.Time       `db:"created_at"`
			UpdatedAt time.Time       `db:"updated_at"`
		}

		if err := rows.StructScan(&file); err != nil {
			return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
		}

		var tags []string
		if file.Tags.Valid && file.Tags.String != "" {
			tags = stringToSlice(file.Tags.String)
		}

		var metadata map[string]interface{}
		if len(file.Metadata) > 0 {
			if err := json.Unmarshal(file.Metadata, &metadata); err != nil {
				return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
			}
		}

		kbFiles = append(kbFiles, &guardrails.KBFile{
			ID:        file.ID,
			Name:      file.Name,
			Content:   file.Content,
			Type:      file.Type,
			Category:  file.Category,
			Tags:      tags,
			Metadata:  metadata,
			Active:    file.Active,
			Version:   file.Version,
			CreatedAt: file.CreatedAt.Format(time.RFC3339),
			UpdatedAt: file.UpdatedAt.Format(time.RFC3339),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	return kbFiles, nil
}

func (r *repository) UpdateKBFile(ctx context.Context, file *guardrails.KBFile) error {
	query := `UPDATE kb_files SET name = :name, content = :content, type = :type, 
		category = :category, tags = :tags, metadata = :metadata, 
		active = :active WHERE id = :id`

	tagsArray := "{" + stringSliceToPostgresArray(file.Tags) + "}"

	metadataJSON, err := json.Marshal(file.Metadata)
	if err != nil {
		return smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	_, err = r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":       file.ID,
		"name":     file.Name,
		"content":  file.Content,
		"type":     file.Type,
		"category": file.Category,
		"tags":     tagsArray,
		"metadata": metadataJSON,
		"active":   file.Active,
	})
	if err != nil {
		return smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	return nil
}

func (r *repository) DeleteKBFile(ctx context.Context, id string) error {
	query := `DELETE FROM kb_files WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	if rowsAffected == 0 {
		return guardrails.ErrNotFound
	}

	return nil
}

func (r *repository) SearchKBFiles(
	ctx context.Context,
	searchQuery string,
	categories, tags []string,
	limit int,
) ([]*guardrails.KBFile, error) {
	sqlQuery := `SELECT * FROM search_kb_files($1, $2, $3, $4)`

	var categoriesArray interface{}
	if len(categories) > 0 {
		categoriesArray = "{" + stringSliceToPostgresArray(categories) + "}"
	} else {
		categoriesArray = nil
	}

	var tagsArray interface{}
	if len(tags) > 0 {
		tagsArray = "{" + stringSliceToPostgresArray(tags) + "}"
	} else {
		tagsArray = nil
	}

	if limit <= 0 {
		limit = 10
	}

	rows, err := r.db.QueryxContext(ctx, sqlQuery, searchQuery,
		categoriesArray, tagsArray, limit)
	if err != nil {
		return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}
	defer rows.Close()

	var kbFiles []*guardrails.KBFile

	for rows.Next() {
		var file struct {
			ID       string         `db:"id"`
			Name     string         `db:"name"`
			Content  string         `db:"content"`
			Type     string         `db:"type"`
			Category string         `db:"category"`
			Tags     sql.NullString `db:"tags"`
			Score    float32        `db:"score"`
		}

		if err := rows.StructScan(&file); err != nil {
			return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
		}

		var fileTags []string
		if file.Tags.Valid && file.Tags.String != "" {
			fileTags = stringToSlice(file.Tags.String)
		}

		kbFiles = append(kbFiles, &guardrails.KBFile{
			ID:       file.ID,
			Name:     file.Name,
			Content:  file.Content,
			Type:     file.Type,
			Category: file.Category,
			Tags:     fileTags,
			Active:   true,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, smqErrors.Wrap(guardrails.ErrViewEntity, err)
	}

	return kbFiles, nil
}

func NewRepository(db postgres.Database) guardrails.Repository {
	return &repository{
		db: db,
	}
}

func stringSliceToPostgresArray(slice []string) string {
	if len(slice) == 0 {
		return ""
	}

	result := ""

	for i, s := range slice {
		if i > 0 {
			result += ","
		}

		escaped := strings.ReplaceAll(s, `"`, `\"`)
		result += `"` + escaped + `"`
	}

	return result
}

func stringToSlice(s string) []string {
	if s == "" {
		return []string{}
	}

	return strings.Split(s, ",")
}

func getOrderQuery(order string) string {
	switch order {
	case "name":
		return "name"
	default:
		return "created_at"
	}
}

func getDirQuery(dir string) string {
	switch dir {
	case "asc":
		return "ASC"
	default:
		return "DESC"
	}
}
