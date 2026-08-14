# Frontend

The Vue frontend exposes two authenticated application shells:

- `/legal/ai-assistant` is the default ChatSwitch legal workspace. Conversations stay inside this shell at `/legal/ai-assistant/chat/:chatid`.
- `/legal/contract-review` lists active and archived Contract Review tasks.
- `/legal/contract-review/:reviewId` opens the persistent document/review workspace.
- `/legal/smart-archive` opens the Smart Archive / Contract Archive workspace.
- `/platform/*` retains the existing knowledge-base, agent, organization, settings, and legacy chat surfaces.

The root route and successful login/onboarding flows open the ChatSwitch workspace. The legacy `/platform` root still opens `/platform/knowledge-bases`.

## Brand assets

The shared ChatSwitch wordmark is `src/assets/img/ChatSwitch_logo.svg`. The login header and expanded navigation shells use this asset; the legal sidebar keeps its compact `C` monogram when collapsed so the narrow layout remains usable.

## Extending the legal workspace

Legal-tool navigation is registered in `src/config/legalWorkspace.ts`. To add a tool, define its route constant in `src/router/paths.ts`, add a typed navigation item with its `labelKey`, destination, icon, and `activeRouteNames`, register the child route under `/legal` in `src/router/index.ts`, add the corresponding lazy-loaded workspace component, and add the label to each locale file. Disabled future tools can remain in the registry without a destination.

The ChatSwitch sidebar uses its own `lexai_legal_sidebar_collapsed` preference so it does not change the legacy platform sidebar state. Resource links intentionally leave the ChatSwitch shell and open their existing `/platform/*` pages.

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

Completed reviews can enter reconfiguration mode: update the playbook or represented party, cancel the pending changes, or rerun the review with the new configuration. Failed reviews can be retried without uploading the source document again.

Add future playbooks to the backend playbook registry and preserve their version on each review. The typed frontend API and viewer adapter live under `src/views/legal/contract-review`.

## Smart Archive MVP

Smart Archive imports PDF, Word, Excel, JPG/JPEG, PNG and WEBP files. It stores
structured fields with source evidence, supports OCR and natural-language
search, and sends unverifiable documents to Review Queue. Related-party values
are cleaned before linking; new imports do not create asset rows.

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
scheduling immediately while notification history remains. Pending candidates
can be batch-marked `ignored` and remain available for audit.

To extend the module, add an enum/schema field in `internal/types/smart_archive.go`, persist its evidence in `archive_field_evidence`, and expose the field in `src/views/legal/SmartArchive.vue`. Keep citations bound to a document and source locator; never display an AI value without its evidence.

The backend API, extraction lifecycle, evidence model, and reminder endpoints are documented in [`docs/smart-archive.md`](../docs/smart-archive.md).

## Verification

Run from this directory:

```bash
npm test
npm run type-check
npm run build
```

Repository policy requires choosing the build/test host first. For remote verification, copy the working tree to the selected host without using Git on that host, then run the commands from the copied `frontend` directory.
