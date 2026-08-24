-- document-edit lifecycle timestamps were originally created as naive
-- TIMESTAMP values. Automatic GORM timestamps were UTC, while lifecycle
-- timestamps written with time.Now() could contain the host's local clock.
-- Repair legacy rows using the adjacent automatic timestamp before promoting
-- every document-edit timestamp to timestamptz.
UPDATE document_edit_jobs
SET completed_at = updated_at
WHERE completed_at IS NOT NULL
  AND ABS(EXTRACT(EPOCH FROM (completed_at - updated_at))) > 3600;

UPDATE document_edit_stage_runs
SET started_at = created_at
WHERE ABS(EXTRACT(EPOCH FROM (started_at - created_at))) > 3600;

UPDATE document_edit_stage_runs
SET completed_at = updated_at
WHERE completed_at IS NOT NULL
  AND ABS(EXTRACT(EPOCH FROM (completed_at - updated_at))) > 3600;

UPDATE document_edit_operations
SET applied_at = created_at
WHERE applied_at IS NOT NULL
  AND ABS(EXTRACT(EPOCH FROM (applied_at - created_at))) > 3600;

-- The old processor persisted the job start in the database but kept a stale
-- in-memory job with StartedAt == nil. Recover it from the first debug stage
-- when that trace is available.
UPDATE document_edit_jobs AS jobs
SET started_at = (
    SELECT MIN(stages.started_at)
    FROM document_edit_stage_runs AS stages
    WHERE stages.job_id = jobs.id
)
WHERE jobs.started_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM document_edit_stage_runs AS stages
    WHERE stages.job_id = jobs.id
  );

ALTER TABLE document_edit_jobs
    ALTER COLUMN started_at TYPE TIMESTAMP WITH TIME ZONE
        USING started_at AT TIME ZONE 'UTC',
    ALTER COLUMN completed_at TYPE TIMESTAMP WITH TIME ZONE
        USING completed_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMP WITH TIME ZONE
        USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMP WITH TIME ZONE
        USING updated_at AT TIME ZONE 'UTC',
    ALTER COLUMN deleted_at TYPE TIMESTAMP WITH TIME ZONE
        USING deleted_at AT TIME ZONE 'UTC';

ALTER TABLE document_edit_artifacts
    ALTER COLUMN created_at TYPE TIMESTAMP WITH TIME ZONE
        USING created_at AT TIME ZONE 'UTC';

ALTER TABLE document_edit_operations
    ALTER COLUMN applied_at TYPE TIMESTAMP WITH TIME ZONE
        USING applied_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMP WITH TIME ZONE
        USING created_at AT TIME ZONE 'UTC';

ALTER TABLE document_edit_stage_runs
    ALTER COLUMN started_at TYPE TIMESTAMP WITH TIME ZONE
        USING started_at AT TIME ZONE 'UTC',
    ALTER COLUMN completed_at TYPE TIMESTAMP WITH TIME ZONE
        USING completed_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at TYPE TIMESTAMP WITH TIME ZONE
        USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMP WITH TIME ZONE
        USING updated_at AT TIME ZONE 'UTC';

ALTER TABLE document_edit_debug_blobs
    ALTER COLUMN created_at TYPE TIMESTAMP WITH TIME ZONE
        USING created_at AT TIME ZONE 'UTC';
