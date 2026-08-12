DROP INDEX IF EXISTS idx_archive_field_evidence_source;

ALTER TABLE archive_field_evidence DROP COLUMN IF EXISTS chunk_id;
ALTER TABLE archive_field_evidence DROP COLUMN IF EXISTS knowledge_id;
