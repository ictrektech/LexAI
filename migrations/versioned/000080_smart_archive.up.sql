CREATE TABLE IF NOT EXISTS archive_settings (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL UNIQUE,
    managed_knowledge_base_id VARCHAR(36) NOT NULL DEFAULT '',
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    extraction_model_id VARCHAR(128) NOT NULL DEFAULT '',
    extraction_version VARCHAR(32) NOT NULL DEFAULT '1.0',
    trash_retention_days INTEGER NOT NULL DEFAULT 30,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS archive_import_batches (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    total INTEGER NOT NULL DEFAULT 0,
    completed INTEGER NOT NULL DEFAULT 0,
    failed INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(24) NOT NULL DEFAULT 'processing',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_archive_import_batches_tenant ON archive_import_batches(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS archive_customers (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name VARCHAR(512) NOT NULL,
    normalized VARCHAR(512) NOT NULL,
    aliases JSONB NOT NULL DEFAULT '[]'::JSONB,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_archive_customers_normalized ON archive_customers(tenant_id, normalized);

CREATE TABLE IF NOT EXISTS archive_documents (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    import_batch_id VARCHAR(36) NULL,
    knowledge_id VARCHAR(36) NOT NULL DEFAULT '',
    title VARCHAR(512) NOT NULL,
    file_name VARCHAR(1024) NOT NULL,
    file_type VARCHAR(16) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    file_hash VARCHAR(64) NOT NULL,
    file_path TEXT NOT NULL,
    document_type VARCHAR(32) NOT NULL DEFAULT 'other',
    business_type VARCHAR(16) NOT NULL DEFAULT 'other',
    customer_id VARCHAR(36) NULL,
    agreement_number VARCHAR(256) NOT NULL DEFAULT '',
    signed_at TIMESTAMP NULL,
    effective_at TIMESTAMP NULL,
    expires_at TIMESTAMP NULL,
    returned_at TIMESTAMP NULL,
    renewed_at TIMESTAMP NULL,
    amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    currency VARCHAR(16) NOT NULL DEFAULT '',
    extracted_text TEXT NOT NULL DEFAULT '',
    extracted_fields JSONB NOT NULL DEFAULT '{}'::JSONB,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    extraction_status VARCHAR(24) NOT NULL DEFAULT 'uploading',
    extraction_version VARCHAR(32) NOT NULL DEFAULT '1.0',
    error_message TEXT NOT NULL DEFAULT '',
    archived_at TIMESTAMP NULL,
    trashed_at TIMESTAMP NULL,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_archive_documents_hash ON archive_documents(tenant_id, file_hash) WHERE trashed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_archive_documents_listing ON archive_documents(tenant_id, trashed_at, archived_at, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_archive_documents_agreement ON archive_documents(tenant_id, agreement_number);

CREATE TABLE IF NOT EXISTS archive_assets (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name VARCHAR(512) NOT NULL DEFAULT '',
    model VARCHAR(256) NOT NULL DEFAULT '',
    serial_number VARCHAR(256) NOT NULL DEFAULT '',
    quantity INTEGER NOT NULL DEFAULT 1,
    is_quantity_only BOOLEAN NOT NULL DEFAULT FALSE,
    customer_id VARCHAR(36) NULL,
    business_type VARCHAR(16) NOT NULL DEFAULT 'other',
    status VARCHAR(16) NOT NULL DEFAULT 'unknown',
    status_override BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_archive_assets_sn ON archive_assets(tenant_id, serial_number) WHERE serial_number <> '';
CREATE INDEX IF NOT EXISTS idx_archive_assets_model ON archive_assets(tenant_id, model);

CREATE TABLE IF NOT EXISTS archive_document_assets (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    document_id VARCHAR(36) NOT NULL REFERENCES archive_documents(id) ON DELETE CASCADE,
    asset_id VARCHAR(36) NOT NULL REFERENCES archive_assets(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 1,
    link_status VARCHAR(16) NOT NULL DEFAULT 'confirmed',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_archive_document_assets_pair ON archive_document_assets(document_id, asset_id);

CREATE TABLE IF NOT EXISTS archive_document_links (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    from_document_id VARCHAR(36) NOT NULL REFERENCES archive_documents(id) ON DELETE CASCADE,
    to_document_id VARCHAR(36) NOT NULL REFERENCES archive_documents(id) ON DELETE CASCADE,
    relation VARCHAR(32) NOT NULL,
    link_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_archive_document_links_from ON archive_document_links(tenant_id, from_document_id);
CREATE INDEX IF NOT EXISTS idx_archive_document_links_to ON archive_document_links(tenant_id, to_document_id);

CREATE TABLE IF NOT EXISTS archive_field_evidence (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    document_id VARCHAR(36) NOT NULL REFERENCES archive_documents(id) ON DELETE CASCADE,
    knowledge_id VARCHAR(36) NOT NULL DEFAULT '',
    chunk_id VARCHAR(36) NOT NULL DEFAULT '',
    field_name VARCHAR(128) NOT NULL,
    value TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    quote TEXT NOT NULL DEFAULT '',
    locator_kind VARCHAR(32) NOT NULL DEFAULT 'text',
    locator JSONB NOT NULL DEFAULT '{}'::JSONB,
    source_start INTEGER NOT NULL DEFAULT 0,
    source_end INTEGER NOT NULL DEFAULT 0,
    is_manual BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_archive_field_evidence_document ON archive_field_evidence(tenant_id, document_id, field_name);
CREATE INDEX IF NOT EXISTS idx_archive_field_evidence_source ON archive_field_evidence(tenant_id, knowledge_id, chunk_id);

CREATE TABLE IF NOT EXISTS archive_reminders (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    document_id VARCHAR(36) NULL,
    customer_id VARCHAR(36) NULL,
    asset_id VARCHAR(36) NULL,
    assignee_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL,
    title VARCHAR(512) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    rule JSONB NOT NULL DEFAULT '{}'::JSONB,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    due_at TIMESTAMP NULL,
    snoozed_until TIMESTAMP NULL,
    last_occurrence_at TIMESTAMP NULL,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_archive_reminders_due ON archive_reminders(tenant_id, status, due_at);

CREATE TABLE IF NOT EXISTS archive_reminder_occurrences (
    id VARCHAR(36) PRIMARY KEY,
    reminder_id VARCHAR(36) NOT NULL REFERENCES archive_reminders(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    fingerprint VARCHAR(128) NOT NULL UNIQUE,
    due_at TIMESTAMP NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS archive_notifications (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    reminder_id VARCHAR(36) NULL,
    title VARCHAR(512) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    read_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_archive_notifications_user ON archive_notifications(tenant_id, user_id, read_at, created_at DESC);
