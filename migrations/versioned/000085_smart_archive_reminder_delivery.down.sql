DROP INDEX IF EXISTS idx_archive_notifications_occurrence_unique;
DROP INDEX IF EXISTS idx_archive_notifications_occurrence_lookup;
ALTER TABLE archive_notifications DROP COLUMN IF EXISTS occurrence_id;
