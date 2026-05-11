-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

-- Sources represent external integration origins (Google Drive, SharePoint, etc.)
-- from which documents are ingested into the vector store.
CREATE TABLE IF NOT EXISTS sources (
    id                 UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id            TEXT        NOT NULL,
    source_type        TEXT        NOT NULL,
    name               TEXT        NOT NULL,
    config             JSONB       NOT NULL DEFAULT '{}',
    status             TEXT        NOT NULL DEFAULT 'active',
    sync_enabled       BOOLEAN     NOT NULL DEFAULT FALSE,
    auto_sync_interval INT         NOT NULL DEFAULT 0,
    last_sync_at       TIMESTAMPTZ,
    last_sync_error    TEXT,
    next_sync_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sources_user_id_idx ON sources (user_id);

-- Records represent individual indexed items linked to their source.
-- external_id / external_url / external_ref preserve the full traceability
-- chain back to the original document in the source system.
CREATE TABLE IF NOT EXISTS records (
    id                 UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id            TEXT        NOT NULL,
    source_id          UUID        NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    name               TEXT        NOT NULL,
    format             TEXT        NOT NULL,
    status             TEXT        NOT NULL DEFAULT 'queued',
    -- External reference: trace vectorized content back to its origin.
    external_id        TEXT,
    external_url       TEXT,
    external_ref       TEXT,
    mime_type          TEXT,
    -- Content metadata (populated after indexing).
    description        TEXT,
    chunk_count        INT,
    size_bytes         BIGINT,
    page_count         INT,
    -- Version tracking for idempotent re-sync.
    source_version     TEXT,
    source_modified_at TIMESTAMPTZ,
    error              TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- A given external item is indexed only once per user per source.
    UNIQUE (user_id, source_id, external_id)
);

CREATE INDEX IF NOT EXISTS records_user_id_idx   ON records (user_id);
CREATE INDEX IF NOT EXISTS records_source_id_idx ON records (source_id);

-- Chunks store text fragments and optional vectors for retrieval.
-- Final schema reflects all previous chunk table alters.
CREATE TABLE IF NOT EXISTS chunks (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     TEXT        NOT NULL,
    document_id UUID,
    record_id   UUID        REFERENCES records(id) ON DELETE CASCADE,
    content     TEXT        NOT NULL,
    chunk_index INT         NOT NULL DEFAULT 0,
    page_number INT,
    embedding   vector,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS chunks_user_id_idx   ON chunks (user_id);
CREATE INDEX IF NOT EXISTS chunks_record_id_idx ON chunks (record_id) WHERE record_id IS NOT NULL;
