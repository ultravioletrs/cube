-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

-- Trigram index on record name to support fast case-insensitive ILIKE
-- substring search from the chat "Customize records" panel at scale.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS records_name_trgm_idx
    ON records USING GIN (name gin_trgm_ops);
