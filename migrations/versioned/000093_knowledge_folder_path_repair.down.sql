-- Restore the legacy representation before removing the repaired column.
UPDATE knowledges
SET file_name = folder_path || '/' || file_name
WHERE folder_path <> ''
  AND file_name <> '';

DROP INDEX IF EXISTS idx_knowledges_folder_path;

ALTER TABLE knowledges
    DROP COLUMN IF EXISTS folder_path;
