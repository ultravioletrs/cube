// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "guardrails_001",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS guardrails_policies (
						id VARCHAR(36) UNIQUE NOT NULL,
						name VARCHAR(255) NOT NULL UNIQUE,
						description TEXT,
						config JSONB NOT NULL,
						enabled BOOLEAN DEFAULT true,
						created_at TIMESTAMPTZ DEFAULT NOW(),
						updated_at TIMESTAMPTZ DEFAULT NOW(),
						PRIMARY KEY (id)
					)`,

					`CREATE TABLE IF NOT EXISTS guardrails_rules (
						id VARCHAR(36) UNIQUE NOT NULL,
						policy_id VARCHAR(36) NOT NULL,
						rule_type VARCHAR(50) NOT NULL, -- 'input', 'output', 'content'
						pattern TEXT NOT NULL,
						action VARCHAR(50) NOT NULL, -- 'allow', 'block', 'flag'
						severity VARCHAR(20) DEFAULT 'medium', -- 'low', 'medium', 'high', 'critical'
						metadata JSONB DEFAULT '{}',
						enabled BOOLEAN DEFAULT true,
						created_at TIMESTAMPTZ DEFAULT NOW(),
						updated_at TIMESTAMPTZ DEFAULT NOW(),
						FOREIGN KEY (policy_id) REFERENCES guardrails_policies (id) ON DELETE CASCADE,
						PRIMARY KEY (id)
					)`,

					`CREATE TABLE IF NOT EXISTS guardrails_audit_logs (
						id VARCHAR(36) UNIQUE NOT NULL,
						request_id VARCHAR(255) NOT NULL,
						policy_id VARCHAR(36),
						rule_id VARCHAR(36),
						action VARCHAR(50) NOT NULL, -- 'allowed', 'blocked', 'flagged'
						content_hash VARCHAR(64), -- SHA256 hash of sensitive content
						reason TEXT,
						severity VARCHAR(20) DEFAULT 'info',
						metadata JSONB DEFAULT '{}',
						timestamp TIMESTAMPTZ DEFAULT NOW(),
						FOREIGN KEY (policy_id) REFERENCES guardrails_policies (id) ON DELETE SET NULL,
						FOREIGN KEY (rule_id) REFERENCES guardrails_rules (id) ON DELETE SET NULL,
						PRIMARY KEY (id)
					)`,

					`CREATE TABLE IF NOT EXISTS guardrails_restricted_topics (
						id VARCHAR(36) UNIQUE NOT NULL,
						topic VARCHAR(255) NOT NULL UNIQUE,
						description TEXT,
						severity VARCHAR(20) DEFAULT 'medium',
						enabled BOOLEAN DEFAULT true,
						created_at TIMESTAMPTZ DEFAULT NOW(),
						updated_at TIMESTAMPTZ DEFAULT NOW(),
						PRIMARY KEY (id)
					)`,

					`CREATE TABLE IF NOT EXISTS guardrails_bias_patterns (
						id VARCHAR(36) UNIQUE NOT NULL,
						category VARCHAR(100) NOT NULL, -- 'gender', 'race', 'religion', etc.
						pattern TEXT NOT NULL,
						description TEXT,
						severity VARCHAR(20) DEFAULT 'medium',
						enabled BOOLEAN DEFAULT true,
						created_at TIMESTAMPTZ DEFAULT NOW(),
						updated_at TIMESTAMPTZ DEFAULT NOW(),
						PRIMARY KEY (id)
					)`,

					`CREATE TABLE IF NOT EXISTS guardrails_factuality_config (
						id VARCHAR(36) UNIQUE NOT NULL,
						confidence_threshold DECIMAL(3,2) DEFAULT 0.8,
						require_citations BOOLEAN DEFAULT false,
						fact_check_enabled BOOLEAN DEFAULT true,
						updated_at TIMESTAMPTZ DEFAULT NOW(),
						PRIMARY KEY (id)
					)`,
				},
				Down: []string{
					"DROP TABLE IF EXISTS guardrails_factuality_config",
					"DROP TABLE IF EXISTS guardrails_bias_patterns",
					"DROP TABLE IF EXISTS guardrails_restricted_topics",
					"DROP TABLE IF EXISTS guardrails_audit_logs",
					"DROP TABLE IF EXISTS guardrails_rules",
					"DROP TABLE IF EXISTS guardrails_policies",
				},
			},
			{
				Id: "guardrails_002",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_guardrails_policies_enabled ON guardrails_policies(enabled)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_rules_policy_id ON guardrails_rules(policy_id)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_rules_enabled ON guardrails_rules(enabled)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_audit_logs_timestamp ON guardrails_audit_logs(timestamp)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_audit_logs_request_id ON guardrails_audit_logs(request_id)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_restricted_topics_enabled ON guardrails_restricted_topics(enabled)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_bias_patterns_category ON guardrails_bias_patterns(category)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_bias_patterns_enabled ON guardrails_bias_patterns(enabled)`,
				},
				Down: []string{
					"DROP INDEX IF EXISTS idx_guardrails_bias_patterns_enabled",
					"DROP INDEX IF EXISTS idx_guardrails_bias_patterns_category",
					"DROP INDEX IF EXISTS idx_guardrails_restricted_topics_enabled",
					"DROP INDEX IF EXISTS idx_guardrails_audit_logs_request_id",
					"DROP INDEX IF EXISTS idx_guardrails_audit_logs_timestamp",
					"DROP INDEX IF EXISTS idx_guardrails_rules_enabled",
					"DROP INDEX IF EXISTS idx_guardrails_rules_policy_id",
					"DROP INDEX IF EXISTS idx_guardrails_policies_enabled",
				},
			},
			{
				Id: "guardrails_003",
				Up: []string{
					`INSERT INTO guardrails_factuality_config (id, confidence_threshold, require_citations, fact_check_enabled) 
					 VALUES ('default-factuality-config', 0.8, false, true)
					 ON CONFLICT (id) DO NOTHING`,

					`INSERT INTO guardrails_restricted_topics (id, topic, description, severity) VALUES 
					 ('violence-topic', 'violence', 'Content related to violence or harm', 'high'),
					 ('illegal-activities-topic', 'illegal_activities', 'Content promoting illegal activities', 'critical'),
					 ('hate-speech-topic', 'hate_speech', 'Discriminatory or hateful language', 'high'),
					 ('personal-info-topic', 'personal_information', 'Requests for personal or private information', 'medium')
					 ON CONFLICT (id) DO NOTHING`,

					`INSERT INTO guardrails_bias_patterns (id, category, pattern, description, severity) VALUES 
					 ('gender-bias-pattern', 'gender', 'stereotypical gender role assumptions', 'Patterns that assume traditional gender roles', 'medium'),
					 ('race-bias-pattern', 'race', 'racial stereotypes or generalizations', 'Content that makes racial generalizations', 'high'),
					 ('age-bias-pattern', 'age', 'age-based discrimination patterns', 'Content that discriminates based on age', 'medium'),
					 ('religion-bias-pattern', 'religion', 'religious bias or stereotypes', 'Content showing religious bias', 'medium')
					 ON CONFLICT (id) DO NOTHING`,
				},
				Down: []string{
					"DELETE FROM guardrails_bias_patterns WHERE id IN ('gender-bias-pattern', 'race-bias-pattern', 'age-bias-pattern', 'religion-bias-pattern')",
					"DELETE FROM guardrails_restricted_topics WHERE id IN ('violence-topic', 'illegal-activities-topic', 'hate-speech-topic', 'personal-info-topic')",
					"DELETE FROM guardrails_factuality_config WHERE id = 'default-factuality-config'",
				},
			},
		},
	}
}
