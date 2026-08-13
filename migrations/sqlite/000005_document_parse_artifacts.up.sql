CREATE TABLE IF NOT EXISTS document_parse_artifacts (
    id TEXT PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    source_document_id TEXT NOT NULL DEFAULT '',
    file_hash TEXT NOT NULL,
    file_name TEXT NOT NULL DEFAULT '',
    file_type TEXT NOT NULL DEFAULT '',
    parser_version TEXT NOT NULL,
    markdown_content TEXT NOT NULL,
    result TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, file_hash, parser_version)
);
CREATE INDEX IF NOT EXISTS idx_document_parse_artifacts_scope ON document_parse_artifacts(tenant_id, parser_version, updated_at);
CREATE INDEX IF NOT EXISTS idx_document_parse_artifacts_source ON document_parse_artifacts(tenant_id, source_document_id);
