-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE records ADD COLUMN IF NOT EXISTS ingest_stage TEXT;
