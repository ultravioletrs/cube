-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

-- Existing dev/prod volumes may have been created before embedder data was
-- scoped by Cube workspace. Add the new domain_id columns without dropping data.
ALTER TABLE sources ADD COLUMN IF NOT EXISTS domain_id TEXT;
UPDATE sources SET domain_id = 'legacy' WHERE domain_id IS NULL;
ALTER TABLE sources ALTER COLUMN domain_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS sources_domain_id_idx ON sources (domain_id);

ALTER TABLE records ADD COLUMN IF NOT EXISTS domain_id TEXT;
UPDATE records r
SET domain_id = s.domain_id
FROM sources s
WHERE r.source_id = s.id AND r.domain_id IS NULL;
UPDATE records SET domain_id = 'legacy' WHERE domain_id IS NULL;
ALTER TABLE records ALTER COLUMN domain_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS records_domain_id_idx ON records (domain_id);
CREATE UNIQUE INDEX IF NOT EXISTS records_domain_source_external_idx
    ON records (domain_id, source_id, external_id);

ALTER TABLE chunks ADD COLUMN IF NOT EXISTS domain_id TEXT;
UPDATE chunks c
SET domain_id = r.domain_id
FROM records r
WHERE c.record_id = r.id AND c.domain_id IS NULL;
UPDATE chunks SET domain_id = 'legacy' WHERE domain_id IS NULL;
ALTER TABLE chunks ALTER COLUMN domain_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS chunks_domain_id_idx ON chunks (domain_id);

ALTER TABLE conversations ADD COLUMN IF NOT EXISTS domain_id TEXT;
UPDATE conversations SET domain_id = 'legacy' WHERE domain_id IS NULL;
ALTER TABLE conversations ALTER COLUMN domain_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS conversations_domain_id_idx ON conversations (domain_id);
