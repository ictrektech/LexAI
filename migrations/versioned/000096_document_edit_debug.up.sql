ALTER TABLE document_edit_jobs ADD COLUMN IF NOT EXISTS comparison_group_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE document_edit_jobs ADD COLUMN IF NOT EXISTS comparison_parent_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE document_edit_jobs ADD COLUMN IF NOT EXISTS comparison_strategy VARCHAR(16) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_document_edit_jobs_comparison_group ON document_edit_jobs(comparison_group_id);
CREATE INDEX IF NOT EXISTS idx_document_edit_jobs_comparison_parent ON document_edit_jobs(comparison_parent_id);

ALTER TABLE document_edit_operations ADD COLUMN IF NOT EXISTS actual_matches INTEGER NULL;
ALTER TABLE document_edit_operations ADD COLUMN IF NOT EXISTS engine_name VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE document_edit_operations ADD COLUMN IF NOT EXISTS engine_message TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS document_edit_stage_runs (
    id VARCHAR(36) PRIMARY KEY,
    job_id VARCHAR(36) NOT NULL REFERENCES document_edit_jobs(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    stage VARCHAR(32) NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 1,
    engine_name VARCHAR(64) NOT NULL DEFAULT '',
    engine_version VARCHAR(64) NOT NULL DEFAULT '',
    protocol_version VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP NULL,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    input_summary JSONB NOT NULL DEFAULT '{}'::JSONB,
    output_summary JSONB NOT NULL DEFAULT '{}'::JSONB,
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_document_edit_stage_runs_job ON document_edit_stage_runs(job_id, started_at, created_at);

CREATE TABLE IF NOT EXISTS document_edit_debug_blobs (
    id VARCHAR(36) PRIMARY KEY,
    job_id VARCHAR(36) NOT NULL REFERENCES document_edit_jobs(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    stage_run_id VARCHAR(36) NOT NULL REFERENCES document_edit_stage_runs(id) ON DELETE CASCADE,
    kind VARCHAR(64) NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    storage_ref TEXT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_document_edit_debug_blobs_job ON document_edit_debug_blobs(job_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_edit_debug_blobs_stage_kind ON document_edit_debug_blobs(stage_run_id, kind);
