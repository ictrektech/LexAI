# Smart Archive MVP

Smart Archive is a workspace-scoped legal tool at `/legal/smart-archive`.
It stores historical PDF, DOC/DOCX, XLS/XLSX, JPG/JPEG, PNG and WEBP documents,
extracts a fixed set of contract fields, optionally links related parties by normalized values, and keeps
field-level source evidence for every accepted candidate. Asset extraction and
document-to-asset association are intentionally disabled; legacy asset tables
remain only for backwards-compatible migrations.

## API

- `GET/PATCH /api/v1/archive/settings`
- `POST /api/v1/archive/import-batches`, `GET /api/v1/archive/import-batches/:id`, `GET /api/v1/archive/import-batches/:id/events`
- `GET /api/v1/archive/documents`, `GET/PATCH /api/v1/archive/documents/:id`
- `POST /api/v1/archive/documents/:id/archive`, `POST /api/v1/archive/documents/:id/restore`
- `POST /api/v1/archive/documents/bulk/archive`, `/bulk/restore` accept
  `{ "ids": ["..."] }` and return per-document success/failure results.
  Archive preserves associated reminder candidates and formal reminders.
- `GET /api/v1/archive/documents/:id/evidence`, `GET /api/v1/archive/documents/:id/preview`
- The preview endpoint streams the original source with `application/pdf`,
  Office, `image/jpeg`, `image/png` or `image/webp` as appropriate.
- `POST /api/v1/archive/search` with `query`, `filters`, `page`, and `page_size`
- `GET /api/v1/archive/customers`, `PATCH /api/v1/archive/customers/:id`
- `GET /api/v1/archive/reminders`, `POST /api/v1/archive/reminders`,
  `PATCH/DELETE /api/v1/archive/reminders/:id`
- `GET /api/v1/archive/reminder-candidates`, `POST /api/v1/archive/reminder-candidates/:id/create`
- `GET /api/v1/archive/notifications` and `POST /api/v1/archive/notifications/:id/read`

Legacy asset endpoints and tables are retained only for existing deployments;
they are not linked from the UI, and new imports do not create asset rows or
document-to-asset links.

Failed or low-confidence documents can be retried with
`POST /api/v1/archive/documents/:id/retry-extraction`. Archive and restore are
the only public document-lifecycle operations.

In the Documents view, enable “Show archived” to list archived records. Each
archived row provides Restore, which clears `archived_at`.
The document table also supports selecting visible rows with checkboxes. The
bulk toolbar applies archive or restore and reports partial failures
individually instead of hiding them.

All endpoints scope records to the authenticated workspace. Import returns a
batch ID; the events endpoint emits progress snapshots while extraction and
optional related-party linking continues in the background.

## Extraction and evidence

The first schema recognizes contract, loan, outbound, return, renewal,
payment and delivery documents. Dates, agreement numbers, customers and
amounts are stored on `archive_documents` and their evidence rows in
`archive_field_evidence`. A locator includes character offsets and
parser-specific page/paragraph/sheet hints. Candidates remain visible until a
user corrects them; manual corrections are preserved.

Related-party values are normalized before customer linking. Markdown and OCR
presentation markers, bullets, table pipes, and role-only values such as
`乙方：` are never stored as customer names. Government procurement documents
also recognize `采购人` and `甲方（买方）` explicitly. Startup backfill repairs
older machine-extracted party values and evidence while preserving manual
corrections.

Standalone JPG/JPEG, PNG and WEBP files are kept as their original bytes. The
image branch selects the archive extraction model when it is configured, or an
active tenant vision model otherwise, and sends a bounded OCR derivative (the
camera original is never overwritten) through the existing OCR prompt before
reusing the same field schema and reminder-candidate flow. A transient vision
request is retried once. Loan agreements are classified before generic
“contract/agreement” markers, and date ranges such as “借用期自 … 至 …” use
the end date as the return deadline; the borrower, device model/SN and table
rows are retained as evidence where the OCR contains them. Markdown emphasis
returned by the OCR model (for example `**2025 年 11 月 1 日**`) is treated as
presentation markup and removed from archive OCR text, field values and
evidence quotes; the uploaded image remains unchanged.
Image evidence is marked with locator kind `image`, page `1`, the OCR quote and
character range; coordinates are omitted unless a future provider supplies
them. If no active vision model is available, the original remains durable and
the document enters the review queue without guessed fields or reminders. A
`needs_review` row now shows the reason and exposes **Re-identify** (重新识别);
retrying uses the persisted original and, after successful extraction,
regenerates the return and missing-return reminder candidates without creating
formal reminders automatically.

The first workspace request creates a dedicated managed document knowledge
base. Smart Archive now writes one durable `document_parse_artifacts` row per
file hash and parser version immediately after its reader/OCR pass. The
managed Knowledge Base receives the artifact ID and reuses the stored
`ReadResult` for chunking and indexing, so the same source is not parsed/OCR'd
twice. Image mirrors only carry the multimodal flag needed by the legacy
worker gate; they do not invoke a second vision model. Older rows without an
artifact are backfilled from their already extracted text when possible, and
legacy failed image mirrors are repaired on retry. The archive tables remain
the structured source of truth; an artifact/indexing failure is reported
separately from field extraction.
Field evidence keeps the archive document, knowledge ID (when indexed),
optional chunk ID, character range and a format-specific locator. If an older
completed archive row has no knowledge ID, the next Smart Archive settings
load performs an idempotent best-effort backfill from the durable original
file. A normal knowledge-base edit/delete request is blocked for this managed
KB. The managed KB is nevertheless exposed from `/platform/knowledge-bases`
as a read-only document view, so users can open the parsed knowledge records
and original previews; imports, parsing settings, tags and deletion remain
available only from Smart Archive.

## Reminders

Extraction creates reminder candidates only. Users must confirm the source
date, offset, time and assignee and explicitly create a formal draft reminder;
they must then activate it before scheduling. Supported MVP types are one-time
expiry/return/payment/delivery/renewal reminders and a missing-return
condition. Active rows are scheduled from the database, occurrences are
deduplicated by fingerprint, and in-app notifications are written for the
assignee. The runner targets the next persisted due time and also performs a
bounded five-minute compensation scan, plus an immediate scan after startup.
The database transaction records the occurrence, notification and delivery
cursor together; a failed write is rolled back and retried, while concurrent
workers are protected by the occurrence fingerprint and notification
occurrence key. This means an in-memory timer or process restart cannot erase
a due reminder (delivery is at-least-once at the database boundary, with
idempotent in-app notification creation). Date reminders default to 09:00 in the workspace timezone
(`Asia/Shanghai`) and are stored as UTC. Use `GET
/api/v1/archive/reminder-candidates` to list pending suggestions (use
`?status=pending|created|superseded|ignored` for history) and `POST
/api/v1/archive/reminder-candidates/:id/create` to create a draft reminder;
the latter is transactional and idempotent. The assignee must be an active
workspace member; the UI loads that member list rather than accepting an
unvalidated free-form recipient. A low-confidence candidate remains pending
until its source field is corrected and is not executable.

Import progress is delivered through the batch SSE endpoint and backed by a
1.5-second snapshot poll. The Documents view additionally polls while a row
is in an active parsing state, covering SSE buffering, transient disconnects,
and pages reopened during a background import.

The public document API no longer exposes delete or move-to-trash operations.
Existing rows placed in trash by an older deployment remain subject to the
legacy retention cleanup so upgrades do not leak stored source files or stale
managed-knowledge mirrors.

Periodic reminders, email/Feishu delivery, custom schemas and external ERP
synchronization are intentionally deferred.

The Reminders tab supports contributor-only batch management. `POST
/api/v1/archive/reminders/bulk/delete` accepts `{ "ids": ["..."] }` and
deletes up to 500 tenant-scoped formal reminders in one request, returning a
per-item result for partial failures. Deleting an active or snoozed reminder
immediately removes it from the scheduler; notification history is retained.
Pending suggestions can be reviewed in bulk with `POST
/api/v1/archive/reminder-candidates/bulk/ignore`. Ignoring changes the
candidate status to `ignored` (it is not a physical delete), never creates a
formal reminder, and can be inspected with
`GET /api/v1/archive/reminder-candidates?status=ignored`.
