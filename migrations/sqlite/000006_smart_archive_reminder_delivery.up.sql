ALTER TABLE archive_notifications ADD COLUMN occurrence_id TEXT;
CREATE INDEX IF NOT EXISTS idx_archive_notifications_occurrence_lookup ON archive_notifications(occurrence_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_archive_notifications_occurrence_unique ON archive_notifications(occurrence_id) WHERE occurrence_id IS NOT NULL AND occurrence_id <> '';
