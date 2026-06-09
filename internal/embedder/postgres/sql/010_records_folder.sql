-- Copyright (c) Ultraviolet
-- SPDX-License-Identifier: Apache-2.0

-- Folder structure for records. folder_path is the human-readable containing
-- folder path within the source (e.g. /Docs/2024/Q3); folder_id is the
-- immediate parent folder ID in the source system. Populated on sync for
-- folder-tree ingests (Google Drive); NULL for existing/flat records.
ALTER TABLE records ADD COLUMN IF NOT EXISTS folder_path TEXT;
ALTER TABLE records ADD COLUMN IF NOT EXISTS folder_id TEXT;

-- text_pattern_ops supports fast prefix (LIKE '/Docs/2024%') folder filtering.
CREATE INDEX IF NOT EXISTS records_folder_path_idx
    ON records (folder_path text_pattern_ops);
