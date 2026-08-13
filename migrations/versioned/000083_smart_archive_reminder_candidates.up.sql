CREATE TABLE IF NOT EXISTS archive_reminder_candidates (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    document_id VARCHAR(36) NOT NULL,
    document_title VARCHAR(512) NOT NULL DEFAULT '',
    customer_id VARCHAR(36) NULL,
    asset_id VARCHAR(36) NULL,
    assignee_id VARCHAR(64) NOT NULL DEFAULT '',
    type VARCHAR(32) NOT NULL,
    source_field VARCHAR(128) NOT NULL,
    event_at TIMESTAMP NOT NULL,
    suggested_offset_days INTEGER NOT NULL DEFAULT 0,
    title VARCHAR(512) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    quote TEXT NOT NULL DEFAULT '',
    locator JSONB NOT NULL DEFAULT '{}'::JSONB,
    rule JSONB NOT NULL DEFAULT '{}'::JSONB,
    needs_review BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    reminder_id VARCHAR(36) NULL,
    fingerprint VARCHAR(160) NOT NULL UNIQUE,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_archive_reminder_candidates_scope ON archive_reminder_candidates(tenant_id, status, event_at);
CREATE INDEX IF NOT EXISTS idx_archive_reminder_candidates_document ON archive_reminder_candidates(tenant_id, document_id);

-- Drafts created by the previous parser are no longer executable suggestions.
-- Keep them for audit/history but make them explicitly canceled.
UPDATE archive_reminders SET status = 'canceled' WHERE status = 'draft';
