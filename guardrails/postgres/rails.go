// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/ultraviolet/cube/guardrails"
)

var _ guardrails.Repository = (*repository)(nil)

type repository struct {
	db postgres.Database
}

// NewRepository creates a new guardrails repository
func NewRepository(db postgres.Database) guardrails.Repository {
	return &repository{
		db: db,
	}
}

// Policy represents a guardrails policy in the database
type Policy struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Rules       string    `db:"rules" json:"rules"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Rule represents a guardrails rule in the database
type Rule struct {
	ID        string          `db:"id" json:"id"`
	PolicyID  string          `db:"policy_id" json:"policy_id"`
	RuleType  string          `db:"rule_type" json:"rule_type"`
	Pattern   string          `db:"pattern" json:"pattern"`
	Action    string          `db:"action" json:"action"`
	Severity  string          `db:"severity" json:"severity"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata"`
	Enabled   bool            `db:"enabled" json:"enabled"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string    `db:"id" json:"id"`
	Action    string    `db:"action" json:"action"`
	Resource  string    `db:"resource" json:"resource"`
	UserID    string    `db:"user_id" json:"user_id"`
	Details   string    `db:"details" json:"details"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

// RestrictedTopic represents a restricted topic
type RestrictedTopic struct {
	ID          string    `db:"id" json:"id"`
	Topic       string    `db:"topic" json:"topic"`
	Description string    `db:"description" json:"description"`
	Severity    string    `db:"severity" json:"severity"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// BiasPattern represents a bias pattern
type BiasPattern struct {
	ID          string    `db:"id" json:"id"`
	Category    string    `db:"category" json:"category"`
	Pattern     string    `db:"pattern" json:"pattern"`
	Description string    `db:"description" json:"description"`
	Severity    string    `db:"severity" json:"severity"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// FactualityConfig represents factuality configuration
type FactualityConfig struct {
	ID                  string    `db:"id" json:"id"`
	Enabled             bool      `db:"enabled" json:"enabled"`
	ConfidenceThreshold float64   `db:"confidence_threshold" json:"confidence_threshold"`
	RequireCitations    bool      `db:"require_citations" json:"require_citations"`
	ExternalAPIURL      string    `db:"external_api_url" json:"external_api_url"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}

func (r *repository) CreatePolicy(ctx context.Context, policy guardrails.Policy) error {
	query := `
		INSERT INTO guardrails_policies (id, name, description, rules, enabled)
		VALUES (:id, :name, :description, :rules, :enabled)`

	_, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":          policy.ID,
		"name":        policy.Name,
		"description": policy.Description,
		"rules":       policy.Rules,
		"enabled":     policy.Enabled,
	})
	if err != nil {
		return postgres.HandleError(err, service.ErrCreateEntity)
	}

	return nil
}

func (r *repository) GetPolicy(ctx context.Context, id string) (guardrails.Policy, error) {
	var dbPolicy Policy
	query := `SELECT id, name, description, rules, enabled, created_at, updated_at 
			  FROM guardrails_policies WHERE id = $1`

	if err := r.db.QueryRowxContext(ctx, query, id).StructScan(&dbPolicy); err != nil {
		return guardrails.Policy{}, fmt.Errorf("failed to get policy: %w", err)
	}

	return guardrails.Policy{
		ID:          dbPolicy.ID,
		Name:        dbPolicy.Name,
		Description: dbPolicy.Description,
		Rules:       dbPolicy.Rules,
		Enabled:     dbPolicy.Enabled,
		CreatedAt:   dbPolicy.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   dbPolicy.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (r *repository) ListPolicies(ctx context.Context, limit, offset int) ([]guardrails.Policy, error) {
	var dbPolicies []Policy
	query := `SELECT id, name, description, rules, enabled, created_at, updated_at 
			  FROM guardrails_policies ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryxContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dbPolicy Policy
		if err := rows.StructScan(&dbPolicy); err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}
		dbPolicies = append(dbPolicies, dbPolicy)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate policies: %w", err)
	}

	policies := make([]guardrails.Policy, len(dbPolicies))
	for i, dbPolicy := range dbPolicies {
		policies[i] = guardrails.Policy{
			ID:          dbPolicy.ID,
			Name:        dbPolicy.Name,
			Description: dbPolicy.Description,
			Rules:       dbPolicy.Rules,
			Enabled:     dbPolicy.Enabled,
			CreatedAt:   dbPolicy.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   dbPolicy.UpdatedAt.Format(time.RFC3339),
		}
	}

	return policies, nil
}

func (r *repository) UpdatePolicy(ctx context.Context, policy guardrails.Policy) error {
	query := `
		UPDATE guardrails_policies 
		SET name = :name, description = :description, rules = :rules, 
			enabled = :enabled, updated_at = NOW()
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":          policy.ID,
		"name":        policy.Name,
		"description": policy.Description,
		"rules":       policy.Rules,
		"enabled":     policy.Enabled,
	})

	return err
}

func (r *repository) DeletePolicy(ctx context.Context, id string) error {
	query := `DELETE FROM guardrails_policies WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *repository) GetRestrictedTopics(ctx context.Context) ([]string, error) {
	var topics []string
	query := `SELECT topic FROM guardrails_restricted_topics WHERE enabled = true`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get restricted topics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var topic string
		if err := rows.Scan(&topic); err != nil {
			return nil, fmt.Errorf("failed to scan topic: %w", err)
		}
		topics = append(topics, topic)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate topics: %w", err)
	}

	return topics, nil
}

func (r *repository) UpdateRestrictedTopics(ctx context.Context, topics []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM guardrails_restricted_topics"); err != nil {
		return fmt.Errorf("failed to clear existing topics: %w", err)
	}

	for _, topic := range topics {
		query := `INSERT INTO guardrails_restricted_topics (topic, enabled) VALUES ($1, true)`
		if _, err := tx.ExecContext(ctx, query, topic); err != nil {
			return fmt.Errorf("failed to insert topic %s: %w", topic, err)
		}
	}

	return tx.Commit()
}

func (r *repository) AddRestrictedTopic(ctx context.Context, topic string) error {
	query := `INSERT INTO guardrails_restricted_topics (topic, enabled) VALUES ($1, true)
			  ON CONFLICT (topic) DO UPDATE SET enabled = true, updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, topic)
	return err
}

func (r *repository) RemoveRestrictedTopic(ctx context.Context, topic string) error {
	query := `DELETE FROM guardrails_restricted_topics WHERE topic = $1`
	_, err := r.db.ExecContext(ctx, query, topic)
	return err
}

func (r *repository) GetBiasPatterns(ctx context.Context) (map[string][]guardrails.BiasPattern, error) {
	var dbPatterns []BiasPattern
	query := `SELECT category, pattern, description, severity FROM guardrails_bias_patterns WHERE enabled = true`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get bias patterns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dbPattern BiasPattern
		if err := rows.StructScan(&dbPattern); err != nil {
			return nil, fmt.Errorf("failed to scan bias pattern: %w", err)
		}
		dbPatterns = append(dbPatterns, dbPattern)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate bias patterns: %w", err)
	}

	patterns := make(map[string][]guardrails.BiasPattern)
	for _, dbPattern := range dbPatterns {
		patterns[dbPattern.Category] = append(patterns[dbPattern.Category], guardrails.BiasPattern{
			Pattern:     dbPattern.Pattern,
			Description: dbPattern.Description,
			Severity:    dbPattern.Severity,
		})
	}

	return patterns, nil
}

func (r *repository) UpdateBiasPatterns(ctx context.Context, patterns map[string][]guardrails.BiasPattern) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM guardrails_bias_patterns"); err != nil {
		return fmt.Errorf("failed to clear existing patterns: %w", err)
	}

	for category, categoryPatterns := range patterns {
		for _, pattern := range categoryPatterns {
			query := `INSERT INTO guardrails_bias_patterns (category, pattern, description, severity, enabled) 
					  VALUES ($1, $2, $3, $4, true)`
			if _, err := tx.ExecContext(ctx, query, category, pattern.Pattern, pattern.Description, pattern.Severity); err != nil {
				return fmt.Errorf("failed to insert bias pattern: %w", err)
			}
		}
	}

	return tx.Commit()
}

func (r *repository) GetFactualityConfig(ctx context.Context) (guardrails.FactualityConfig, error) {
	var dbConfig FactualityConfig
	query := `SELECT enabled, confidence_threshold, require_citations, external_api_url 
			  FROM guardrails_factuality_config LIMIT 1`

	if err := r.db.QueryRowxContext(ctx, query).StructScan(&dbConfig); err != nil {
		return guardrails.FactualityConfig{}, fmt.Errorf("failed to get factuality config: %w", err)
	}

	return guardrails.FactualityConfig{
		Enabled:             dbConfig.Enabled,
		ConfidenceThreshold: dbConfig.ConfidenceThreshold,
		RequireCitations:    dbConfig.RequireCitations,
		ExternalAPIURL:      dbConfig.ExternalAPIURL,
	}, nil
}

func (r *repository) UpdateFactualityConfig(ctx context.Context, config guardrails.FactualityConfig) error {
	query := `UPDATE guardrails_factuality_config 
			  SET enabled = $1, confidence_threshold = $2, require_citations = $3, 
				  external_api_url = $4, updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, config.Enabled, config.ConfidenceThreshold, config.RequireCitations, config.ExternalAPIURL)
	return err
}

func (r *repository) CreateAuditLog(ctx context.Context, log guardrails.AuditLog) error {
	query := `
		INSERT INTO guardrails_audit_logs (id, action, resource, user_id, details)
		VALUES (:id, :action, :resource, :user_id, :details)`

	_, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":       log.ID,
		"action":   log.Action,
		"resource": log.Resource,
		"user_id":  log.UserID,
		"details":  log.Details,
	})

	return err
}

func (r *repository) GetAuditLogs(ctx context.Context, limit int) ([]guardrails.AuditLog, error) {
	var dbLogs []AuditLog
	query := `SELECT id, action, resource, user_id, details, timestamp
			  FROM guardrails_audit_logs ORDER BY timestamp DESC LIMIT $1`

	rows, err := r.db.QueryxContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dbLog AuditLog
		if err := rows.StructScan(&dbLog); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		dbLogs = append(dbLogs, dbLog)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate audit logs: %w", err)
	}

	logs := make([]guardrails.AuditLog, len(dbLogs))
	for i, dbLog := range dbLogs {
		logs[i] = guardrails.AuditLog{
			ID:        dbLog.ID,
			Action:    dbLog.Action,
			Resource:  dbLog.Resource,
			UserID:    dbLog.UserID,
			Details:   dbLog.Details,
			Timestamp: dbLog.Timestamp.Format(time.RFC3339),
		}
	}

	return logs, nil
}

func (r *repository) ExportConfig(ctx context.Context) ([]byte, error) {
	config := make(map[string]interface{})

	topics, err := r.GetRestrictedTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export restricted topics: %w", err)
	}
	config["restricted_topics"] = topics

	patterns, err := r.GetBiasPatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export bias patterns: %w", err)
	}
	config["bias_patterns"] = patterns

	factualityConfig, err := r.GetFactualityConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export factuality config: %w", err)
	}
	config["factuality"] = factualityConfig

	return json.MarshalIndent(config, "", "  ")
}

func (r *repository) ImportConfig(ctx context.Context, data []byte) error {
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config data: %w", err)
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	if topicsRaw, ok := config["restricted_topics"]; ok {
		topicsBytes, _ := json.Marshal(topicsRaw)
		var topics []string
		if err := json.Unmarshal(topicsBytes, &topics); err == nil {
			if err := r.UpdateRestrictedTopics(ctx, topics); err != nil {
				return fmt.Errorf("failed to import restricted topics: %w", err)
			}
		}
	}

	if patternsRaw, ok := config["bias_patterns"]; ok {
		patternsBytes, _ := json.Marshal(patternsRaw)
		var patterns map[string][]guardrails.BiasPattern
		if err := json.Unmarshal(patternsBytes, &patterns); err == nil {
			if err := r.UpdateBiasPatterns(ctx, patterns); err != nil {
				return fmt.Errorf("failed to import bias patterns: %w", err)
			}
		}
	}

	if factualityRaw, ok := config["factuality"]; ok {
		factualityBytes, _ := json.Marshal(factualityRaw)
		var factualityConfig guardrails.FactualityConfig
		if err := json.Unmarshal(factualityBytes, &factualityConfig); err == nil {
			if err := r.UpdateFactualityConfig(ctx, factualityConfig); err != nil {
				return fmt.Errorf("failed to import factuality config: %w", err)
			}
		}
	}

	return tx.Commit()
}
