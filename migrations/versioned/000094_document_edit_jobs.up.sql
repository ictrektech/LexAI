CREATE TABLE IF NOT EXISTS document_edit_jobs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    format VARCHAR(8) NOT NULL,
    mode VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL,
    file_name VARCHAR(1024) NOT NULL,
    mime_type VARCHAR(255) NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    source_sha256 CHAR(64) NOT NULL,
    source_ref TEXT NOT NULL,
    instruction TEXT NOT NULL DEFAULT '',
    model_id VARCHAR(64) NOT NULL DEFAULT '',
    plan JSONB NOT NULL DEFAULT '{}'::JSONB,
    capabilities JSONB NOT NULL DEFAULT '{}'::JSONB,
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);
CREATE INDEX IF NOT EXISTS idx_document_edit_jobs_owner ON document_edit_jobs(tenant_id, user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_document_edit_jobs_status ON document_edit_jobs(status);

CREATE TABLE IF NOT EXISTS document_edit_artifacts (
    id VARCHAR(36) PRIMARY KEY,
    job_id VARCHAR(36) NOT NULL REFERENCES document_edit_jobs(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    kind VARCHAR(32) NOT NULL,
    file_name VARCHAR(1024) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    storage_ref TEXT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_document_edit_artifacts_job ON document_edit_artifacts(job_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_edit_artifacts_kind ON document_edit_artifacts(job_id, kind);

CREATE TABLE IF NOT EXISTS document_edit_operations (
    id VARCHAR(36) PRIMARY KEY,
    job_id VARCHAR(36) NOT NULL REFERENCES document_edit_jobs(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    operation_id VARCHAR(128) NOT NULL,
    kind VARCHAR(32) NOT NULL,
    part VARCHAR(32) NOT NULL DEFAULT '',
    anchor_sha256 CHAR(64) NOT NULL,
    expected_matches INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(16) NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    applied_at TIMESTAMP NULL
);
CREATE INDEX IF NOT EXISTS idx_document_edit_operations_job ON document_edit_operations(job_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_edit_operations_id ON document_edit_operations(job_id, operation_id);
