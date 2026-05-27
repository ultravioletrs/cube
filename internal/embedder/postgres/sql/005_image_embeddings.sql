-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

CREATE TABLE IF NOT EXISTS image_embeddings (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain_id   TEXT        NOT NULL,
    user_id     TEXT        NOT NULL,
    record_id   UUID        NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    model       TEXT        NOT NULL,
    dimensions  INT         NOT NULL,
    embedding   vector      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (record_id)
);

CREATE INDEX IF NOT EXISTS image_embeddings_domain_id_idx ON image_embeddings (domain_id);
CREATE INDEX IF NOT EXISTS image_embeddings_record_id_idx ON image_embeddings (record_id);
