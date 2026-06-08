-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

-- Keep existing local databases compatible with domain-scoped repositories.
ALTER TABLE sources ADD COLUMN IF NOT EXISTS domain_id TEXT NOT NULL DEFAULT '';
ALTER TABLE records ADD COLUMN IF NOT EXISTS domain_id TEXT NOT NULL DEFAULT '';
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS domain_id TEXT NOT NULL DEFAULT '';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS domain_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS sources_domain_id_idx ON sources (domain_id);
CREATE INDEX IF NOT EXISTS records_domain_id_idx ON records (domain_id);
CREATE INDEX IF NOT EXISTS chunks_domain_id_idx ON chunks (domain_id);
CREATE INDEX IF NOT EXISTS conversations_domain_id_idx ON conversations (domain_id);
