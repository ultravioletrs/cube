-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

-- Keep compatibility with historical schemas that predate vector/page metadata.
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS embedding vector(768);
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS page_number INT;
