# Frontend

The Vue frontend exposes two authenticated application shells:

- `/legal/ai-assistant` is the default LexAI legal workspace. Conversations stay inside this shell at `/legal/ai-assistant/chat/:chatid`.
- `/legal/contract-review` lists active and archived Contract Review tasks.
- `/legal/contract-review/:reviewId` opens the persistent document/review workspace.
- `/platform/*` retains the existing knowledge-base, agent, organization, settings, and legacy chat surfaces.

The root route and successful login/onboarding flows open the LexAI workspace. The legacy `/platform` root still opens `/platform/knowledge-bases`.

## Extending the legal workspace

Legal-tool navigation is registered in `src/config/legalWorkspace.ts`. Add a typed navigation item, a child route under `/legal` in `src/router/index.ts`, and the corresponding lazy-loaded workspace component. Disabled future tools can remain in the registry without a destination.

The LexAI sidebar uses its own `lexai_legal_sidebar_collapsed` preference so it does not change the legacy platform sidebar state. Resource links intentionally leave the LexAI shell and open their existing `/platform/*` pages.

## Contract Review

Creating a review immediately persists a draft. Upload one PDF or DOCX, wait for the document to reach `ready`, select the General Contract Review playbook and represented party, then start the review. Results arrive incrementally over an authenticated SSE stream and remain available after refresh.

PDF contracts use `pdfjs-dist` for page navigation, zoom, selectable text, and issue highlights. DOCX contracts use `docx-preview`; its browser pagination can differ slightly from Microsoft Word. AI issue highlights are included in this release, while manual annotations, redlining, exports, and collaboration are extension points.

Add future playbooks to the backend playbook registry and preserve their version on each review. The typed frontend API and viewer adapter live under `src/views/legal/contract-review`.

## Verification

Run from this directory:

```bash
npm test
npm run type-check
npm run build
```

Repository policy requires choosing the build/test host first. For remote verification, copy the working tree to the selected host without using Git on that host, then run the commands from the copied `frontend` directory.
