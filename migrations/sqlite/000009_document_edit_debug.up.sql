ALTER TABLE document_edit_jobs ADD COLUMN comparison_group_id TEXT NOT NULL DEFAULT '';
ALTER TABLE document_edit_jobs ADD COLUMN comparison_parent_id TEXT NOT NULL DEFAULT '';
ALTER TABLE document_edit_jobs ADD COLUMN comparison_strategy TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_document_edit_jobs_comparison_group ON document_edit_jobs(comparison_group_id);
CREATE INDEX IF NOT EXISTS idx_document_edit_jobs_comparison_parent ON document_edit_jobs(comparison_parent_id);

ALTER TABLE document_edit_operations ADD COLUMN actual_matches INTEGER;
ALTER TABLE document_edit_operations ADD COLUMN engine_name TEXT NOT NULL DEFAULT '';
ALTER TABLE document_edit_operations ADD COLUMN engine_message TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS document_edit_stage_runs (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES document_edit_jobs(id) ON DELETE CASCADE,
    tenant_id INTEGER NOT NULL,
    stage TEXT NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 1,
    engine_name TEXT NOT NULL DEFAULT '',
    engine_version TEXT NOT NULL DEFAULT '',
    protocol_version TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    input_summary TEXT NOT NULL DEFAULT '{}',
    output_summary TEXT NOT NULL DEFAULT '{}',
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_document_edit_stage_runs_job ON document_edit_stage_runs(job_id, started_at, created_at);

CREATE TABLE IF NOT EXISTS document_edit_debug_blobs (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES document_edit_jobs(id) ON DELETE CASCADE,
    tenant_id INTEGER NOT NULL,
    stage_run_id TEXT NOT NULL REFERENCES document_edit_stage_runs(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    content_type TEXT NOT NULL,
    storage_ref TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_document_edit_debug_blobs_job ON document_edit_debug_blobs(job_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_edit_debug_blobs_stage_kind ON document_edit_debug_blobs(stage_run_id, kind);
