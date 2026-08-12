# Frontend

The Vue frontend exposes two authenticated application shells:

- `/legal/ai-assistant` is the default LexAI legal workspace. Conversations stay inside this shell at `/legal/ai-assistant/chat/:chatid`.
- `/legal/contract-review` lists active and archived Contract Review tasks.
- `/legal/contract-review/:reviewId` opens the persistent document/review workspace.
- `/legal/smart-archive` opens the shared Contract Archive workspace.
- `/platform/*` retains the existing knowledge-base, agent, organization, settings, and legacy chat surfaces.

The root route and successful login/onboarding flows open the LexAI workspace. The legacy `/platform` root still opens `/platform/knowledge-bases`.

## Extending the legal workspace

Legal-tool navigation is registered in `src/config/legalWorkspace.ts`. Add a typed navigation item, a child route under `/legal` in `src/router/index.ts`, and the corresponding lazy-loaded workspace component. Disabled future tools can remain in the registry without a destination.

The LexAI sidebar uses its own `lexai_legal_sidebar_collapsed` preference so it does not change the legacy platform sidebar state. Resource links intentionally leave the LexAI shell and open their existing `/platform/*` pages.

## Legal workspace color system

All `/legal/*` pages use **Warm Legal + Vercel Neutral**: warm ivory surfaces, soft-black actions, neutral-gray AI cues, and restrained brass/risk accents. The semantic CSS tokens are declared by `src/views/legal/index.vue`; use them instead of adding page-specific hex colors:

- Surfaces: `--legal-bg-page` (`#F7F4ED`), `--legal-bg-surface` (`#FCFBF7`), and `--legal-bg-paper` for document pages.
- Text and structure: `--legal-text-primary` (`#1F1F1F`), `--legal-text-secondary` (`#6B6B6B`), `--legal-border` (`#E2DED6`), and `--legal-bg-hover`.
- Actions and AI: `--legal-brand` (`#1F1F1F`) for primary actions, `--legal-brand-hover` (`#2A2A2A`) and `--legal-brand-active` (`#171717`) for interaction states, plus `--legal-ai` (`#737373`), `--legal-ai-strong` (`#4D4D4D`), and `--legal-ai-soft` (`#F1F1EF`) for AI-generated or selected states.
- Semantics: warning/medium-risk states use `--legal-warning` (`#A9793D`) and `--legal-warning*`; high-risk and error states use `--legal-risk` (`#A6534D`) and `--legal-risk*`. Use the `*-strong` variants for small text and the base colors for icons, borders, progress, or highlights.

The legal shell maps these tokens onto TDesign variables and sets `data-workspace-theme="legal"` on the root element while mounted so teleported popups inherit the same palette. The attribute is removed when leaving `/legal/*`; do not move these overrides into the global platform theme. Statuses must retain a text or icon label and must not rely on color alone.

## Contract Review

Creating a review immediately persists a draft. Upload one PDF or DOCX, wait for the document to reach `ready`, select the General Contract Review playbook and represented party, then start the review. Results arrive incrementally over an authenticated SSE stream and remain available after refresh. Running detail workspaces also fetch a durable snapshot every 1.5 seconds, while the task list silently refreshes every 2 seconds whenever it contains an uploading or reviewing task; this prevents buffered or disconnected SSE connections from leaving stale status text on screen.

The task list supports selecting individual rows or all visible rows. Active tasks can be archived in bulk, archived tasks can be restored in bulk, and either view can permanently delete selected reviews. Running reviews are skipped by archive operations. Deleting a running review warns the user and prevents any later background result from recreating the deleted task. Bulk APIs process up to 500 unique IDs and report per-item success so one invalid task does not block the rest.

PDF contracts use `pdfjs-dist` for page navigation, zoom, selectable text, and issue highlights. DOCX contracts use `docx-preview`; its browser pagination can differ slightly from Microsoft Word. AI issue highlights are included in this release, while manual annotations, redlining, exports, and collaboration are extension points.

Add future playbooks to the backend playbook registry and preserve their version on each review. The typed frontend API and viewer adapter live under `src/views/legal/contract-review`.

## Smart Archive MVP

Smart Archive is a separate, workspace-scoped module. Batch import accepts PDF, DOC/DOCX, XLS/XLSX, JPG/JPEG, PNG and WEBP files through `/api/v1/archive/import-batches`; each file is persisted, parsed, and stored with structured fields and field-level source evidence. Images remain in their original format; OCR uses a bounded derivative of large camera originals, retries transient vision failures, and preserves page-1 image evidence. Loan agreements are recognized before the generic contract marker, including borrower, device/SN and “借用期自…至…” return deadlines. If no vision model or no verifiable field is available, the source is kept in Review Queue with a reason and a Re-identify action; retrying never requires re-uploading the source file and does not fabricate reminder candidates. The UI exposes Documents, Reminders and Review Queue tabs, plus keyword/natural-language search at `/api/v1/archive/search`. An optional related-party/customer association remains available in document details and search results, but is not a required top-level workspace. Party values are cleaned before linking, role-only strings such as `乙方：` are rejected, and government procurement buyers (`采购人` or `甲方（买方）`) are recognized explicitly. Asset tabs and document-to-asset association are intentionally disabled; imported files remain document records.

Each import is also mirrored into the read-only managed knowledge base **合同智能档案**. Smart Archive persists one versioned `document_parse_artifacts` record after its reader/OCR pass; the managed Knowledge Base receives that artifact ID and reuses the normalized `ReadResult` for indexing, so importing a file does not run OCR/document parsing twice. Image mirrors only carry the multimodal flag needed by the legacy worker gate and do not invoke a second vision model. Older completed rows can be backfilled from their stored extracted text, while legacy failed image mirrors are repaired on retry/reparse; the archive document row keeps the mirror link in `knowledge_id`.

Extraction only creates persisted reminder candidates. A user confirms the evidence, offset, time and assignee before the candidate becomes a formal `draft` reminder; the user must then activate it. The assignee picker uses active workspace members and the backend validates the tenant membership. Low-confidence candidates stay pending until the source field is corrected. Only active reminders are scanned from the database and produce idempotent in-app notifications; the default due time is 09:00 in the workspace timezone. Documents use a single archive/restore lifecycle; the workspace does not expose a recycle-bin action.

Import status is refreshed through both the batch SSE stream and a durable
1.5-second batch snapshot poll. The Documents view also polls while any row is
in `uploading`, `parsing`, `extracting` or `linking`, so a buffered/disconnected
SSE connection or reopening the page cannot leave a stale status displayed.

Archiving a document keeps its source, extracted fields, knowledge-base mirror and reminders intact. “Show archived” lists archived records and supports restoring them to the active list. Document delete and move-to-trash endpoints are not exposed by the workspace.

To extend the module, add an enum/schema field in `internal/types/smart_archive.go`, persist its evidence in `archive_field_evidence`, and expose the field in `src/views/legal/SmartArchive.vue`. Keep citations bound to a document and source locator; never display an AI value without its evidence.

## Verification

Run from this directory:

```bash
npm test
npm run type-check
npm run build
```

Repository policy requires choosing the build/test host first. For remote verification, copy the working tree to the selected host without using Git on that host, then run the commands from the copied `frontend` directory.
### Smart Archive bulk document actions

The Legal Smart Archive document table supports selecting visible records and
running bulk archive or restore actions. The API returns a per-document result
so partial failures remain visible; archived documents keep their reminder
candidates and formal reminders.

The Reminders tab also supports selecting formal reminders for batch deletion
through `POST /api/v1/archive/reminders/bulk/delete`. Active and snoozed rows
are deleted only after an explicit warning and stop scheduling immediately;
notification history remains available. Pending reminder candidates can be
selected and marked `ignored` through
`POST /api/v1/archive/reminder-candidates/bulk/ignore`. Ignored candidates are
retained for audit and are available with `status=ignored`; they never create
formal reminders or notifications.
