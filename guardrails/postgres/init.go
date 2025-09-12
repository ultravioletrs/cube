package postgres

import migrate "github.com/rubenv/sql-migrate"

func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "guardrails_initial_setup",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS guardrails_restricted_topics (
						id VARCHAR(36) UNIQUE NOT NULL,
						topic VARCHAR(255) NOT NULL UNIQUE,
						description TEXT,
						severity VARCHAR(20) CHECK (severity IN ('low', 'medium', 'high', 'critical')),
						enabled BOOLEAN,
						created_at TIMESTAMPTZ,
						updated_at TIMESTAMPTZ,
						PRIMARY KEY (id)
					)`,

					`CREATE TABLE IF NOT EXISTS guardrails_bias_patterns (
						id VARCHAR(36) UNIQUE NOT NULL,
						category VARCHAR(100) NOT NULL,
						pattern TEXT NOT NULL,
						description TEXT,
						severity VARCHAR(20) CHECK (severity IN ('low', 'medium', 'high', 'critical')),
						enabled BOOLEAN,
						created_at TIMESTAMPTZ,
						updated_at TIMESTAMPTZ,
						PRIMARY KEY (id)
					)`,

					`CREATE INDEX IF NOT EXISTS idx_guardrails_restricted_topics_enabled ON 
						guardrails_restricted_topics(enabled)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_bias_patterns_category ON 
						guardrails_bias_patterns(category)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_bias_patterns_enabled ON 
						guardrails_bias_patterns(enabled)`,
					`CREATE INDEX IF NOT EXISTS idx_guardrails_bias_patterns_severity ON 
						guardrails_bias_patterns(severity)`,

					`CREATE OR REPLACE FUNCTION update_updated_at_column()
					RETURNS TRIGGER AS $$
					BEGIN
						NEW.updated_at = NOW();
						RETURN NEW;
					END;
					$$ language 'plpgsql'`,

					`CREATE TRIGGER update_topics_updated_at
						BEFORE UPDATE ON guardrails_restricted_topics
						FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

					`CREATE TRIGGER update_patterns_updated_at
						BEFORE UPDATE ON guardrails_bias_patterns
						FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

					`CREATE TABLE IF NOT EXISTS flows (
						id VARCHAR(36) PRIMARY KEY,
						name VARCHAR(255) NOT NULL UNIQUE,
						description TEXT,
						content TEXT NOT NULL,
						type VARCHAR(50) NOT NULL CHECK (type IN ('input', 'output', 'dialog', 'subflow')),
						active BOOLEAN NOT NULL,
						version INTEGER NOT NULL,
						created_at TIMESTAMP NOT NULL,
						updated_at TIMESTAMP NOT NULL
					)`,

					`CREATE INDEX IF NOT EXISTS idx_flows_active ON flows(active)`,
					`CREATE INDEX IF NOT EXISTS idx_flows_type ON flows(type)`,

					`CREATE OR REPLACE FUNCTION update_flows_updated_at()
					RETURNS TRIGGER AS $$
					BEGIN
						NEW.updated_at = CURRENT_TIMESTAMP;
						NEW.version = OLD.version + 1;
						RETURN NEW;
					END;
					$$ LANGUAGE plpgsql`,

					`CREATE TRIGGER flows_updated_at_trigger
					BEFORE UPDATE ON flows
					FOR EACH ROW
					EXECUTE FUNCTION update_flows_updated_at()`,

					`CREATE TABLE IF NOT EXISTS kb_files (
						id VARCHAR(36) PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						content TEXT NOT NULL,
						type VARCHAR(50) NOT NULL,
						category VARCHAR(100) NOT NULL,
						tags TEXT[],
						metadata JSONB,
						active BOOLEAN NOT NULL,
						version INTEGER NOT NULL,
						created_at TIMESTAMP NOT NULL,
						updated_at TIMESTAMP NOT NULL,
						UNIQUE(name, category)
					)`,

					`CREATE INDEX IF NOT EXISTS idx_kb_files_active ON kb_files(active)`,
					`CREATE INDEX IF NOT EXISTS idx_kb_files_category ON kb_files(category)`,
					`CREATE INDEX IF NOT EXISTS idx_kb_files_tags ON kb_files USING GIN(tags)`,
					`CREATE INDEX IF NOT EXISTS idx_kb_files_metadata ON kb_files USING GIN(metadata)`,

					`CREATE INDEX IF NOT EXISTS idx_kb_files_content_fts ON kb_files USING 
						GIN(to_tsvector('english', content))`,

					`CREATE OR REPLACE FUNCTION update_kb_files_updated_at()
					RETURNS TRIGGER AS $$
					BEGIN
						NEW.updated_at = CURRENT_TIMESTAMP;
						NEW.version = OLD.version + 1;
						RETURN NEW;
					END;
					$$ LANGUAGE plpgsql`,

					`CREATE TRIGGER kb_files_updated_at_trigger
					BEFORE UPDATE ON kb_files
					FOR EACH ROW
					EXECUTE FUNCTION update_kb_files_updated_at()`,

					`INSERT INTO flows (id, name, description, content, type) VALUES
						(gen_random_uuid()::text, 'validate_message_content', 
						 'Validate message content to prevent null/empty processing errors', 
						 E'define flow validate message content\\n  $valid_message = execute 
						   validate_message_content\\n  if not $valid_message\\n    bot inform 
						   invalid_message\\n    stop\\n\\ndefine bot inform invalid_message\\n  
						   \"I didn''t receive a valid message. Please try again with a clear 
						   question or request.\"', 
						 'input'),
						(gen_random_uuid()::text, 'redaction_input_processing', 
						 'Redact sensitive information from user inputs', 
						 E'define flow redaction input processing\\n  $has_pii = execute 
						   detect_pii_in_input\\n  if $has_pii\\n    $redacted_input = execute 
						   redact_input_pii\\n    bot inform pii_redacted_input', 
						 'input'),
						(gen_random_uuid()::text, 'enhanced_input_validation', 
						 'Enhanced input validation with comprehensive safety checks', 
						 E'define flow enhanced input validation\\n  $valid_message = execute 
						   validate_message_content\\n  if not $valid_message\\n    bot inform 
						   invalid_message\\n    stop\\n  $jailbreak = execute check_jailbreak_attempt\\n  
						   if $jailbreak\\n    bot refuse inappropriate request\\n    stop\\n  $toxic = 
						   execute check_toxicity_level\\n  if $toxic\\n    bot refuse inappropriate 
						   request\\n    stop\\n  $injection = execute detect_prompt_injection\\n  
						   if $injection\\n    bot refuse inappropriate request\\n    stop', 
						 'input'),
						(gen_random_uuid()::text, 'check_input_blocking', 
						 'Check for blocked content in user inputs', 
						 E'define flow check input blocking\\n  $blocked = execute check_jailbreak_attempt\\n  
						   if $blocked\\n    bot refuse inappropriate request\\n    stop\\n  $toxic = 
						   execute check_toxicity_level\\n  if $toxic\\n    bot refuse inappropriate 
						   request\\n    stop', 
						 'input'),
						(gen_random_uuid()::text, 'redaction_output_processing', 
						 'Redact sensitive information from outputs', 
						 E'define flow redaction output processing\\n  $has_output_pii = execute 
						   detect_pii_in_output\\n  if $has_output_pii\\n    $redacted_output = 
						   execute redact_output_pii\\n    bot inform pii_redacted_output', 
						 'output'),
						(gen_random_uuid()::text, 'validate_output', 
						 'Validate output for safety and compliance', 
						 E'define flow validate output\\n  $valid_output = execute 
						   validate_output_safety\\n  if not $valid_output\\n    bot acknowledge 
						   uncertainty\\n    stop', 
						 'output')
					ON CONFLICT (name) DO NOTHING`,

					`INSERT INTO kb_files (id, name, content, type, category, tags) VALUES
						(gen_random_uuid()::text, 'content_policy.md', 
						 E'# Content Policy\\n\\n## Overview\\nThis AI assistant follows strict 
						   content guidelines to ensure safe and appropriate interactions.\\n\\n## 
						   Prohibited Content\\n- Harmful or dangerous activities\\n- Illegal 
						   activities or advice\\n- Hate speech or discrimination\\n- Personal 
						   attacks or harassment\\n- Misinformation or deception\\n- Privacy 
						   violations\\n\\n## Allowed Content\\n- Educational information\\n- 
						   Creative assistance\\n- Technical support\\n- General knowledge queries\\n- 
						   Constructive discussions', 
						 'markdown', 'policies', ARRAY['policy', 'guidelines', 'safety']),
						(gen_random_uuid()::text, 'safety_guidelines.md', 
						 E'# Safety Guidelines\\n\\n## Core Principles\\n1. **Do No Harm**: Never 
						   provide information that could cause physical, emotional, or financial 
						   harm\\n2. **Respect Privacy**: Never request or store personal information\\n3. 
						   **Be Truthful**: Always provide accurate information or acknowledge uncertainty\\n4. 
						   **Stay Professional**: Maintain appropriate boundaries in all interactions\\n\\n## 
						   Response Guidelines\\n- Decline inappropriate requests politely\\n- Redirect 
						   harmful queries to appropriate resources\\n- Acknowledge limitations when uncertain\\n- 
						   Provide balanced, factual information', 
						 'markdown', 'guidelines', ARRAY['safety', 'guidelines', 'ethics'])
					ON CONFLICT (name, category) DO NOTHING`,

					`CREATE OR REPLACE VIEW active_flows AS
					SELECT id, name, description, content, type, version, updated_at
					FROM flows
					WHERE active = true`,

					`CREATE OR REPLACE VIEW active_kb_files AS
					SELECT id, name, content, type, category, tags, metadata, version, updated_at
					FROM kb_files
					WHERE active = true`,

					`CREATE OR REPLACE FUNCTION search_kb_files(
						search_query TEXT,
						search_categories TEXT[],
						search_tags TEXT[],
						search_limit INTEGER
					)
					RETURNS TABLE (
						id VARCHAR(36),
						name VARCHAR(255),
						content TEXT,
						type VARCHAR(50),
						category VARCHAR(100),
						tags TEXT[],
						score REAL
					) AS $$
					BEGIN
						RETURN QUERY
						SELECT 
							kb.id,
							kb.name,
							kb.content,
							kb.type,
							kb.category,
							kb.tags,
							ts_rank(to_tsvector('english', kb.content), 
								plainto_tsquery('english', search_query)) AS score
						FROM kb_files kb
						WHERE kb.active = true
							AND (search_categories IS NULL OR kb.category = ANY(search_categories))
							AND (search_tags IS NULL OR kb.tags && search_tags)
							AND (search_query IS NULL OR search_query = '' OR 
								 to_tsvector('english', kb.content) @@ 
								 plainto_tsquery('english', search_query))
						ORDER BY score DESC
						LIMIT search_limit;
					END;
					$$ LANGUAGE plpgsql`,
				},
				Down: []string{
					"DROP FUNCTION IF EXISTS search_kb_files",
					"DROP VIEW IF EXISTS active_kb_files",
					"DROP VIEW IF EXISTS active_flows",
					"DROP TRIGGER IF EXISTS kb_files_updated_at_trigger ON kb_files",
					"DROP FUNCTION IF EXISTS update_kb_files_updated_at()",
					"DROP TRIGGER IF EXISTS flows_updated_at_trigger ON flows",
					"DROP FUNCTION IF EXISTS update_flows_updated_at()",
					"DROP INDEX IF EXISTS idx_kb_files_content_fts",
					"DROP INDEX IF EXISTS idx_kb_files_metadata",
					"DROP INDEX IF EXISTS idx_kb_files_tags",
					"DROP INDEX IF EXISTS idx_kb_files_category",
					"DROP INDEX IF EXISTS idx_kb_files_active",
					"DROP TABLE IF EXISTS kb_files",
					"DROP INDEX IF EXISTS idx_flows_type",
					"DROP INDEX IF EXISTS idx_flows_active",
					"DROP TABLE IF EXISTS flows",
					"DROP TRIGGER IF EXISTS update_patterns_updated_at ON guardrails_bias_patterns",
					"DROP TRIGGER IF EXISTS update_topics_updated_at ON guardrails_restricted_topics",
					"DROP FUNCTION IF EXISTS update_updated_at_column()",
					"DROP INDEX IF EXISTS idx_guardrails_bias_patterns_severity",
					"DROP INDEX IF EXISTS idx_guardrails_bias_patterns_enabled",
					"DROP INDEX IF EXISTS idx_guardrails_bias_patterns_category",
					"DROP INDEX IF EXISTS idx_guardrails_restricted_topics_enabled",
					"DROP TABLE IF EXISTS guardrails_bias_patterns",
					"DROP TABLE IF EXISTS guardrails_restricted_topics",
				},
			},
		},
	}
}
