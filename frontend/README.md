# Frontend

The Vue frontend exposes two authenticated application shells:

- `/legal/ai-assistant` is the default ChatSwitch legal workspace. Conversations stay inside this shell at `/legal/ai-assistant/chat/:chatid`.
- `/legal/contract-review` lists active and archived Contract Review tasks.
- `/legal/contract-review/:reviewId` opens the persistent document/review workspace.
- `/legal/smart-archive` opens the Smart Archive / Contract Archive workspace.
- `/legal/drafting/generate` is the Contract Generation placeholder; `/legal/drafting` opens Contract Editing.
- `/platform/*` retains the existing knowledge-base, agent, organization, settings, and legacy chat surfaces.

The root route and successful login/onboarding flows open the ChatSwitch workspace. The legacy `/platform` root still opens `/platform/knowledge-bases`.

## Brand assets

The shared ChatSwitch wordmark is `src/assets/img/ChatSwitch_logo.svg`. The login header and platform navigation shell use this asset. The legal workspace keeps the LexAI identity and uses `src/assets/img/LexAI_logo_exact.svg` in its sidebar, with a compact `L` monogram when collapsed so the narrow layout remains usable.

## Extending the legal workspace

Legal-tool navigation is registered in `src/config/legalWorkspace.ts`. To add a tool, define its route constant in `src/router/paths.ts`, add a typed navigation item with its `labelKey`, destination, icon, and `activeRouteNames`, register the child route under `/legal` in `src/router/index.ts`, add the corresponding lazy-loaded workspace component, and add the label to each locale file. Disabled future tools can remain in the registry without a destination.

The ChatSwitch sidebar uses its own `lexai_legal_sidebar_collapsed` preference so it does not change the legacy platform sidebar state. It starts with the legal tool navigation and does not render a separate new-chat button; the Legal Assistant entry remains the route into conversations. Resource links intentionally leave the ChatSwitch shell and open their existing `/platform/*` pages.

## Agent selection recovery

The selected Agent ID is stored in browser settings. When the current tenant's
Agent list loads, the chat surfaces validate that ID before requesting
suggested questions. If the Agent was deleted, unshared, or disabled, the
frontend clears the stale selection and falls back to an available built-in
Agent, so a stale browser setting does not produce a 404 or block chat startup.

## Legal workspace color system

All `/legal/*` pages use **Warm Legal + Vercel Neutral**: warm ivory surfaces, soft-black actions, neutral-gray AI cues, and restrained brass/risk accents. The semantic CSS tokens are declared by `src/views/legal/index.vue`; use them instead of adding page-specific hex colors:

- Surfaces: `--legal-bg-page` (`#F7F4ED`), `--legal-bg-surface` (`#FCFBF7`), and `--legal-bg-paper` for document pages.
- Text and structure: `--legal-text-primary` (`#1F1F1F`), `--legal-text-secondary` (`#6B6B6B`), `--legal-border` (`#E2DED6`), and `--legal-bg-hover`.
- Actions and AI: `--legal-brand` (`#1F1F1F`) for primary actions, `--legal-brand-hover` (`#2A2A2A`) and `--legal-brand-active` (`#171717`) for interaction states, plus `--legal-ai` (`#737373`), `--legal-ai-strong` (`#4D4D4D`), and `--legal-ai-soft` (`#F1F1EF`) for AI-generated or selected states.
- Semantics: warning/medium-risk states use `--legal-warning` (`#A9793D`) and `--legal-warning*`; high-risk and error states use `--legal-risk` (`#A6534D`) and `--legal-risk*`. Use the `*-strong` variants for small text and the base colors for icons, borders, progress, or highlights.
- Async task states use the Vercel-inspired `--legal-status-*` palette: blue queued, amber running, green completed, red failed, gray cancelled, and purple when human review is required. Each state also has `*-strong` text and `*-soft` ring/background tokens.

The legal shell maps these tokens onto TDesign variables and sets `data-workspace-theme="legal"` on the root element while mounted so teleported popups inherit the same palette. The attribute is removed when leaving `/legal/*`; do not move these overrides into the global platform theme. Statuses must retain a text or icon label and must not rely on color alone.

## Contract Review

Creating a review immediately persists a draft. Upload one PDF or DOCX, wait for the document to reach `ready`, select the General Contract Review playbook and represented party, then start the review. Results arrive incrementally over an authenticated SSE stream and remain available after refresh. Running detail workspaces also fetch a durable snapshot every 1.5 seconds, while the task list silently refreshes every 2 seconds whenever it contains an uploading or reviewing task; this prevents buffered or disconnected SSE connections from leaving stale status text on screen.

The task list supports selecting individual rows or all visible rows. Active tasks can be archived in bulk, archived tasks can be restored in bulk, and either view can permanently delete selected reviews. Running reviews are skipped by archive operations. Deleting a running review warns the user and prevents any later background result from recreating the deleted task. Bulk APIs process up to 500 unique IDs and report per-item success so one invalid task does not block the rest.

PDF contracts use `pdfjs-dist` for page navigation, zoom, selectable text, and issue highlights. DOCX contracts use `docx-preview`; its browser pagination can differ slightly from Microsoft Word. AI issue highlights accept whitespace differences and omission markers such as `...` when locating quoted source text. Manual annotations, redlining, exports, and collaboration are extension points.

Completed reviews can enter reconfiguration mode: update the playbook or represented party, cancel the pending changes, or rerun the review with the new configuration. Failed reviews can be retried without uploading the source document again.

Add future playbooks to the backend playbook registry and preserve their version on each review. The typed frontend API and viewer adapter live under `src/views/legal/contract-review`.

## Contract Drafting

The sidebar's Contract Drafting entry drills down to Contract Generation and
Contract Editing. `/legal/drafting/generate` is a placeholder and does not
start a backend task; `/legal/drafting` creates DOCX edit jobs and keeps active rows current
through an authenticated SSE snapshot stream. The task history supports client-side date,
text, engine-mode, and multi-status filters; queued and running jobs remain at
the top, followed by the most recently updated terminal jobs. Opening a row
navigates to `/legal/drafting/:jobId`, where its request, timing, operations,
errors, and downloadable artifacts are shown in a responsive detail workspace.
Each row shows a compact task ID, while the detail route uses the full UUID and
provides a copy action, so repeated uploads with the same filename remain easy
to distinguish. Browser back/forward and direct links are supported; filters,
the in-memory task snapshot, and list scroll position are restored when the
user returns during the same SPA session. Render previews are fetched only on
demand and their temporary browser URLs are released when the task changes or
the detail page is left.

The task diagnostics route `/legal/drafting/:jobId/debug` keeps every recorded
stage attempt in the timeline, but the Inspect and Planner viewers default to
the latest stage attempt so retries do not render duplicate blobs. Plain text,
annotated text, semantic snapshots, planner messages, and planner responses use
single-click tabs that load and display the selected content directly. Plan JSON
uses the same one-click viewer; historical retry data remains available through
the stage timeline and API.

## Smart Archive MVP

Smart Archive imports PDF, Word, Excel, JPG/JPEG, PNG and WEBP files. It stores
structured fields with source evidence, supports OCR and natural-language
search, and sends unverifiable documents to Review Queue. Related-party values
are cleaned before linking; new imports do not create asset rows.

The Documents tab uses server-side semantic search with import-date, document
type, multi-status, and active/archived filters. Results load 30 at a time and
append through **Load more**. Processing documents keep the loaded result range
fresh during polling. Document rows use the shared Vercel-style status palette,
and open a responsive detail drawer without changing Reminder or Review Queue
layouts.

Each import is mirrored into the read-only managed knowledge base **合同智能档案**
through a reusable parse artifact, avoiding duplicate parsing or OCR. Documents
can be archived/restored; Admin/Owner users can delete archived documents to
move them into the recycle bin. Deletion preserves the audit record while
removing the managed mirror and canceling related reminders.

Extraction creates reminder candidates only. Users confirm the evidence and
schedule, create a `draft`, then activate it. Active reminders generate
idempotent in-app notifications. While an import batch is active, progress uses
SSE plus a durable 1.5-second snapshot poll; each snapshot refreshes the
document rows, so manual refresh is unnecessary while the workspace remains
open.

### Smart Archive bulk management

The document table supports bulk archive, restore and delete with per-item
results. Restore and delete require Admin or Owner; archive requires Contributor
or above. Delete is available from archived rows and the document detail panel.

The Reminders tab supports batch deletion; active and snoozed reminders stop
scheduling immediately and their in-app notifications are removed. Deleting a
source document also cancels its reminders and clears their notifications.
Individual notifications can be deleted from the notification list, while
pending candidates can be batch-marked `ignored` and remain available for
audit.

To extend the module, add an enum/schema field in `internal/types/smart_archive.go`, persist its evidence in `archive_field_evidence`, and expose the field in `src/views/legal/SmartArchive.vue`. Keep citations bound to a document and source locator; never display an AI value without its evidence.

The backend API, extraction lifecycle, evidence model, and reminder endpoints are documented in [`docs/smart-archive.md`](../docs/smart-archive.md).

## Verification

Run from this directory:

```bash
npm test
npm run type-check
npx playwright install chromium
npm run test:e2e
npm run build
```

The legal workspace browser tests live under `tests/e2e`. They use a deterministic
mock of authentication, Contract Review, and Smart Archive APIs, so they do not
call a real model or modify the deployed archive at `10.144.144.232:30080`.
Run `npm run test:e2e -- --ui` when debugging interactions locally. Failed CI
runs retain the Playwright HTML report, trace, screenshot, and video artifacts.

The mock fixtures cover the legal workspace navigation, Contract Review upload
and result flow, Smart Archive archive/recycle-bin flow, reminder lifecycle, and
viewer mutation visibility. Backend lifecycle, HTTP error mapping, RBAC, and
tenant-isolation cases remain in the Go test packages.

Repository policy requires choosing the build/test host first. For remote verification, copy the working tree to the selected host without using Git on that host, then run the commands from the copied `frontend` directory.
