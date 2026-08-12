-- Migration 80 was already deployed in some environments before source
-- linkage was added to archive evidence. Keep this additive and idempotent so
-- both the earlier and current migration-80 schemas converge.
ALTER TABLE archive_field_evidence
    ADD COLUMN IF NOT EXISTS knowledge_id VARCHAR(36) NOT NULL DEFAULT '';

ALTER TABLE archive_field_evidence
    ADD COLUMN IF NOT EXISTS chunk_id VARCHAR(36) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_archive_field_evidence_source
    ON archive_field_evidence(tenant_id, knowledge_id, chunk_id);
