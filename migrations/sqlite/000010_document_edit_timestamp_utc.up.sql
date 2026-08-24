-- SQLite stores timestamps without a timezone. Repair legacy lifecycle
-- values that were written with the host's local clock; automatic GORM
-- timestamps in updated_at/created_at remain the UTC reference values.
UPDATE document_edit_jobs
SET completed_at = updated_at
WHERE completed_at IS NOT NULL
  AND ABS(strftime('%s', completed_at) - strftime('%s', updated_at)) > 3600;

UPDATE document_edit_stage_runs
SET started_at = created_at
WHERE ABS(strftime('%s', started_at) - strftime('%s', created_at)) > 3600;

UPDATE document_edit_stage_runs
SET completed_at = updated_at
WHERE completed_at IS NOT NULL
  AND ABS(strftime('%s', completed_at) - strftime('%s', updated_at)) > 3600;

UPDATE document_edit_operations
SET applied_at = created_at
WHERE applied_at IS NOT NULL
  AND ABS(strftime('%s', applied_at) - strftime('%s', created_at)) > 3600;

UPDATE document_edit_jobs
SET started_at = (
    SELECT MIN(stages.started_at)
    FROM document_edit_stage_runs AS stages
    WHERE stages.job_id = document_edit_jobs.id
)
WHERE started_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM document_edit_stage_runs AS stages
    WHERE stages.job_id = document_edit_jobs.id
  );
