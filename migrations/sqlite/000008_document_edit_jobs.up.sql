CREATE TABLE IF NOT EXISTS document_edit_jobs (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    format TEXT NOT NULL,
    mode TEXT NOT NULL,
    status TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT '',
    file_size INTEGER NOT NULL DEFAULT 0,
    source_sha256 TEXT NOT NULL,
    source_ref TEXT NOT NULL,
    instruction TEXT NOT NULL DEFAULT '',
    model_id TEXT NOT NULL DEFAULT '',
    plan TEXT NOT NULL DEFAULT '{}',
    capabilities TEXT NOT NULL DEFAULT '{}',
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_document_edit_jobs_owner ON document_edit_jobs(tenant_id, user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_document_edit_jobs_status ON document_edit_jobs(status);

CREATE TABLE IF NOT EXISTS document_edit_artifacts (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES document_edit_jobs(id) ON DELETE CASCADE,
    tenant_id INTEGER NOT NULL,
    kind TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    storage_ref TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_document_edit_artifacts_job ON document_edit_artifacts(job_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_edit_artifacts_kind ON document_edit_artifacts(job_id, kind);

CREATE TABLE IF NOT EXISTS document_edit_operations (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES document_edit_jobs(id) ON DELETE CASCADE,
    tenant_id INTEGER NOT NULL,
    operation_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    part TEXT NOT NULL DEFAULT '',
    anchor_sha256 TEXT NOT NULL,
    expected_matches INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    applied_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_document_edit_operations_job ON document_edit_operations(job_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_edit_operations_id ON document_edit_operations(job_id, operation_id);
