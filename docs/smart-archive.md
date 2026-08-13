# Smart Archive MVP

Smart Archive is available at `/legal/smart-archive`. It imports PDF, Word,
Excel, JPG/JPEG, PNG and WEBP files, extracts structured fields with evidence,
and optionally links a related customer. New imports do not create asset rows.

## API

- Settings: `GET/PATCH /api/v1/archive/settings`
- Import: `POST /archive/import-batches`, `GET /archive/import-batches/:id`,
  `GET /archive/import-batches/:id/events`
- Documents: `GET/PATCH /archive/documents/:id`, plus `retry-extraction`,
  `archive`, `restore`, `DELETE`, `evidence` and `preview`
- Bulk documents: `POST /archive/documents/bulk/archive|restore|delete`
- Search and entities: `POST /archive/search`, `GET/PATCH /archive/customers/:id`
- Reminder candidates: list, create reminder, and `bulk/ignore`
- Reminders: list/create/update/delete and `POST /archive/reminders/bulk/delete`
- Notifications: list and mark read

All paths above are under `/api/v1`. Bulk requests use `{ "ids": [...] }` and
return per-item results.

Viewer can read, search, preview and mark notifications read. Contributor can
import, correct fields, retry, archive and manage reminders. Restoring archived
documents requires Admin or Owner. Every operation is tenant-scoped.

## Extraction and evidence

- Import is idempotent by file hash and extraction version.
- Images keep their original bytes; OCR uses an active vision model and a
  bounded derivative. Missing OCR capability sends the document to Review Queue.
- Evidence stores the source quote, character range and a PDF, Office,
  spreadsheet or image locator. Unlocatable values are not presented as facts.
- Related-party names remove Markdown/OCR decoration and reject role-only values
  such as `乙方：`. Government procurement documents recognize `采购人` and
  `甲方（买方）`. Startup backfill repairs old AI values but preserves manual edits.
- Each import is mirrored into the read-only managed knowledge base
  **合同智能档案** through a reusable parse artifact, avoiding duplicate OCR.

Archiving keeps the source, evidence, knowledge-base mirror, reminder candidates
and reminders. Admin/Owner users can move archived documents to the recycle bin
with delete; the operation cancels related pending reminders and removes the
managed knowledge mirror. Bulk delete accepts up to 500 unique IDs and reports
per-item results. Trash cleanup follows the configured retention period.

## Reminders

Extraction creates reminder candidates, never active reminders. A user must:

1. Confirm the date, offset, time and assignee.
2. Create a formal `draft` reminder.
3. Activate it.

Supported MVP events are expiry, return, payment, delivery, renewal and missing
return records. Active reminders use the workspace timezone, persist UTC due
times and create idempotent in-app notifications. The scheduler wakes for the
next due item, scans immediately after startup and runs a five-minute recovery
scan so restarts do not lose reminders.

The Reminders tab can batch-delete up to 500 formal reminders. Deleting an
active or snoozed reminder stops future scheduling but keeps notification
history. Pending candidates can be batch-marked `ignored`; ignored records stay
queryable for audit and never create notifications.

Import progress uses SSE plus a 1.5-second snapshot fallback. The document list
also polls while a row is uploading, parsing, extracting or linking, so status
changes do not require a manual refresh.

## Migrations

- PostgreSQL: `000080`–`000085`
- SQLite: `000003`–`000006`

Use the normal application migration runner. Periodic reminders, external
notifications, custom schemas and ERP synchronization are out of scope.
