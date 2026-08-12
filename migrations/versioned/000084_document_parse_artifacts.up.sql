CREATE TABLE IF NOT EXISTS document_parse_artifacts (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    source_document_id VARCHAR(36) NOT NULL DEFAULT '',
    file_hash VARCHAR(64) NOT NULL,
    file_name VARCHAR(1024) NOT NULL DEFAULT '',
    file_type VARCHAR(32) NOT NULL DEFAULT '',
    parser_version VARCHAR(64) NOT NULL,
    markdown_content TEXT NOT NULL,
    result JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_document_parse_artifacts_fingerprint UNIQUE (tenant_id, file_hash, parser_version)
);
CREATE INDEX IF NOT EXISTS idx_document_parse_artifacts_scope ON document_parse_artifacts(tenant_id, parser_version, updated_at);
CREATE INDEX IF NOT EXISTS idx_document_parse_artifacts_source ON document_parse_artifacts(tenant_id, source_document_id);
