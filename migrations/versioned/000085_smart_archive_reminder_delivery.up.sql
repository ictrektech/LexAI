-- Bind each in-app notification to the durable one-shot occurrence that
-- produced it. The partial unique index is the database-level idempotency
-- guard for concurrent workers and process restarts; legacy notifications
-- remain valid with a NULL occurrence_id.
ALTER TABLE archive_notifications
    ADD COLUMN IF NOT EXISTS occurrence_id VARCHAR(36);

CREATE INDEX IF NOT EXISTS idx_archive_notifications_occurrence_lookup
    ON archive_notifications(occurrence_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_archive_notifications_occurrence_unique
    ON archive_notifications(occurrence_id)
    WHERE occurrence_id IS NOT NULL AND occurrence_id <> '';
