-- Repair deployments that reached a newer migration version before the
-- knowledge-folder migration was introduced at version 000079.
--
-- Some deployments already recorded version 86 (and were later renumbered to
-- 92), so golang-migrate correctly skipped the newly-added 000079 migration.
-- Keep this repair at the next version instead of changing an applied
-- migration file.
ALTER TABLE knowledges
    ADD COLUMN IF NOT EXISTS folder_path VARCHAR(1024) NOT NULL DEFAULT '';

-- Recover the legacy folder representation used by older uploads. The
-- folder_path = '' guard keeps this repair safe if it is re-run manually.
UPDATE knowledges
SET folder_path = LEFT(
        REGEXP_REPLACE(file_name, '/[^/]*$', ''),
        1024
    ),
    file_name = REGEXP_REPLACE(file_name, '^.*/', '')
WHERE folder_path = ''
  AND file_name LIKE '%/%';

CREATE INDEX IF NOT EXISTS idx_knowledges_folder_path
    ON knowledges (tenant_id, knowledge_base_id, folder_path);
