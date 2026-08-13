# Contract Review Workspace

Contract Review is a persistent, per-user legal workflow under `/legal/contract-review`. It does not use chat sessions or expiring chat attachments.

## Lifecycle

`draft -> uploading -> ready -> analyzing -> reviewing_clauses -> completed`

Parsing or model failures move the task to `failed`; the retry endpoint resumes parsing or restarts the structured review. Archived tasks keep their document and results. Delete removes the task from the product and deletes its stored source file.

## API

- `GET|POST /api/v1/contract-reviews`
- `POST /api/v1/contract-reviews/bulk/archive|restore|delete` (`{ "ids": [...] }`, maximum 500 IDs, per-item results)
- `GET|PATCH|DELETE /api/v1/contract-reviews/:id`
- `POST /api/v1/contract-reviews/:id/document` (`file` multipart field; PDF/DOCX only)
- `GET /api/v1/contract-reviews/:id/document/preview` (supports HTTP Range through `ServeContent`)
- `POST /api/v1/contract-reviews/:id/start`
- `POST /api/v1/contract-reviews/:id/retry`
- `GET /api/v1/contract-reviews/:id/events` (SSE snapshots plus heartbeat)
- `GET /api/v1/contract-review-playbooks`

The list UI can select the current visible tasks and archive, restore, or delete them together. Running tasks cannot be archived and are skipped by that action. Deleting a running task is allowed after an explicit warning; the scoped update path prevents an in-flight background worker from recreating a soft-deleted review.

The detail workspace uses SSE with a 1.5-second snapshot fallback. The list
refreshes every 2 seconds while work is running, so status changes do not
require a manual refresh.

Every endpoint derives tenant and user ownership from the authenticated request. API-key access is intentionally not declared.

## Models and playbooks

The structured executor reads the tenant-customized `builtin-contract-review` agent. It reuses the agent's resolved legal system prompt, model, temperature, and token settings while enforcing workspace-specific JSON schemas. If the agent has no model, the tenant default active KnowledgeQA model is used, followed by the first active KnowledgeQA model. Starting a review without any eligible model returns `MODEL_NOT_CONFIGURED`.

General Contract Review v1.0 is the only initial playbook. Add future playbooks to the typed registry and introduce dedicated prompt/schema versions without changing historical task records.

## Migrations and workers

PostgreSQL migration `000079` and SQLite migration `000002` add reviews, clauses, and issues. Document parsing runs in the core attachment queue; clause analysis runs in the enrichment summary queue. Both handlers are also registered with the Lite synchronous executor.
