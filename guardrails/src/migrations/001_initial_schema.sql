-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

-- Migration: 001_initial_schema
-- Description: Create initial guardrails configuration tables

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Guardrail configurations table
-- Stores the base configuration content (config.yml, prompts.yml, colang)
CREATE TABLE IF NOT EXISTS guardrail_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    config_yaml TEXT NOT NULL,
    prompts_yaml TEXT NOT NULL DEFAULT '',
    colang TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Guardrail versions table
-- Tracks versioning and activation state
CREATE TABLE IF NOT EXISTS guardrail_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    config_id UUID NOT NULL REFERENCES guardrail_configs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    revision INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(config_id, revision)
);

-- Materialized guardrail table
-- Denormalized view for efficient runtime loading
CREATE TABLE IF NOT EXISTS guardrail_materialized (
    version_id UUID PRIMARY KEY REFERENCES guardrail_versions(id) ON DELETE CASCADE,
    config_yaml TEXT NOT NULL,
    prompts_yaml TEXT NOT NULL DEFAULT '',
    colang TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_guardrail_configs_name ON guardrail_configs(name);
CREATE INDEX IF NOT EXISTS idx_guardrail_versions_config_id ON guardrail_versions(config_id);
CREATE INDEX IF NOT EXISTS idx_guardrail_versions_is_active ON guardrail_versions(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_guardrail_versions_revision ON guardrail_versions(config_id, revision DESC);

-- Ensure only one active version at a time
CREATE UNIQUE INDEX IF NOT EXISTS idx_guardrail_versions_single_active
    ON guardrail_versions(is_active) WHERE is_active = true;

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to configs table
DROP TRIGGER IF EXISTS update_guardrail_configs_updated_at ON guardrail_configs;
CREATE TRIGGER update_guardrail_configs_updated_at
    BEFORE UPDATE ON guardrail_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to materialized table
DROP TRIGGER IF EXISTS update_guardrail_materialized_updated_at ON guardrail_materialized;
CREATE TRIGGER update_guardrail_materialized_updated_at
    BEFORE UPDATE ON guardrail_materialized
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
