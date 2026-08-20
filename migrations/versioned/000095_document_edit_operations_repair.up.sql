-- Repair migration for deployments where 000094 was marked applied before
-- document_edit_operations was present in the migration payload.
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

CREATE INDEX IF NOT EXISTS idx_document_edit_operations_job
    ON document_edit_operations(job_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_document_edit_operations_id
    ON document_edit_operations(job_id, operation_id);
