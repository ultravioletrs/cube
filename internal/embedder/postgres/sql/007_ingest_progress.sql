-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE records ADD COLUMN IF NOT EXISTS ingest_total_chunks INT;
ALTER TABLE records ADD COLUMN IF NOT EXISTS ingest_indexed_chunks INT;
