CREATE TABLE IF NOT EXISTS archive_reminder_candidates (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    document_id TEXT NOT NULL,
    document_title TEXT NOT NULL DEFAULT '',
    customer_id TEXT,
    asset_id TEXT,
    assignee_id TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL,
    source_field TEXT NOT NULL,
    event_at DATETIME NOT NULL,
    suggested_offset_days INTEGER NOT NULL DEFAULT 0,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    confidence REAL NOT NULL DEFAULT 0,
    quote TEXT NOT NULL DEFAULT '',
    locator TEXT NOT NULL DEFAULT '{}',
    rule TEXT NOT NULL DEFAULT '{}',
    needs_review INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    reminder_id TEXT,
    fingerprint TEXT NOT NULL UNIQUE,
    created_by TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_archive_reminder_candidates_scope ON archive_reminder_candidates(tenant_id, status, event_at);
CREATE INDEX IF NOT EXISTS idx_archive_reminder_candidates_document ON archive_reminder_candidates(tenant_id, document_id);
UPDATE archive_reminders SET status = 'canceled' WHERE status = 'draft';
