-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

-- GIN index on the English tsvector of chunk content.
-- Enables fast @@ operator lookups used by the hybrid BM25 keyword CTE.
CREATE INDEX IF NOT EXISTS chunks_content_fts_idx
    ON chunks USING GIN (to_tsvector('english', content));
