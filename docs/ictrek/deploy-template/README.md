# LexAI deploy template

Copy `.env.example` to `.env`, edit secrets, then run:

```bash
./deploy.sh --platform amd
./deploy.sh --platform l4t
```

Thor uses its dedicated script:

```bash
./deploy-thor.sh
```

For an existing deployment, use the one-step updater from the deploy directory:

```bash
./update-and-deploy.sh --platform thor --check-only
./update-and-deploy.sh --platform thor
./update-and-deploy.sh --platform amd
./update-and-deploy.sh --platform l4t
```

`--check-only` is deliberately narrow: it only reads Feishu for the three LexAI
images (`lexai`, `lexai-ui`, `lexai-docreader`). If a Feishu tag is newer than
the local deployment, it reports the current tag and target tag. If the tag is
the same, it compares the remote image digest with the running container image
digest and reports a same-version image update only when the digest differs.
It does not clone the repo, sync deployment files, write `.env`, pull images, or
recreate containers.

After the user confirms an available image update, the updater pulls the latest
`docs/ictrek` from `LEXAI_DEPLOY_REPO` / `LEXAI_DEPLOY_REF`, syncs
`deploy-template` into the current directory while preserving local `.env`,
`.env.tc232`, and `.env.thor`, then pulls changed images and recreates only
managed LexAI services. Database services such as Postgres, Redis, and Neo4j are
intentionally excluded. If a vLLM model service is recreated, the follow-up
reparse step waits for the model URLs in `WEKNORA_REPARSE_WAIT_URLS` before
submitting unfinished work.

The UI update button does not run arbitrary host commands. It appears as
"检测更新" in the knowledge-base page header, calls the fixed `deploy-updater`
sidecar named by `DEPLOY_UPDATER_CONTAINER`, and starts the update only after
user confirmation. `--check-only` only reads Feishu for the three LexAI images
(`lexai`, `lexai-ui`, `lexai-docreader`) and compares same-tag remote image
digests; it does not clone the repo, sync deployment files, write `.env`, or
recreate containers. After confirmation, the sidecar pulls `docs/ictrek`,
syncs this deploy template, pulls changed images, and replaces only managed
LexAI services. If app and frontend are both updated, app is recreated and
waited healthy before frontend is recreated. Docker pull output and recreate
steps are appended to `update-and-deploy.log` and surfaced in the web dialog.
`deploy-updater` has no separate image artifact: it deliberately reuses the
LexAI app image so the update endpoint and the host-side deploy script stay in
the same release. When the app image changes, `deploy.sh` refreshes this sidecar
after the current update run finishes, otherwise the running updater would keep
executing the previous app image.
Keep `FEISHU_CONFIG_HOST_FILE` pointed at the host Feishu credential file used
for image lookup.

`deploy.sh` reads the latest image tag from each component's own Feishu column and writes image variables into `.env` before running `docker compose up -d`. The LexAI app, UI, and docreader tags are resolved independently and may be different.

All services join the `lexai` Docker network. Host ports start at 30000; service-to-service traffic uses container names inside the network.

`./deploy.sh --platform l4t` resolves component tags from the Feishu `l4t` sheet. The Ollama service is GPU-backed in these templates: `model-hub-ollama` sets `runtime: nvidia`, `NVIDIA_VISIBLE_DEVICES=all`, and `NVIDIA_DRIVER_CAPABILITIES=compute,utility`. On Jetson/Orin hosts, verify a deployment with:

```bash
docker inspect model-hub-ollama --format 'runtime={{.HostConfig.Runtime}}'
docker exec model-hub-ollama sh -lc 'ls /dev/nvhost-gpu /dev/nvmap /dev/nvhost-ctrl-gpu'
```

If `runtime` is empty or the GPU devices are missing, recreate the Ollama container from the updated compose file before testing model speed.

These deployment templates set `WEKNORA_REPARSE_INCOMPLETE_ON_START=true`. Each app container start scans knowledge rows in `failed` or `pending`; `processing` and `finalizing` rows are full-reparsed only when `processed_at is null`. A row must also have a real parse source: `file_path` for uploaded files, `source` for `file_url` / `url`, or non-empty manual `metadata.content`. Status-only rows without parseable content are skipped. This is intentional for redeploys: interrupted or failed text parsing is retried automatically after the new app is healthy, without manually clicking reparse, while text-completed documents are not forced through docreader/chunking/embedding again just because VLM/Graph/Wiki enrichment is still pending.

Startup reparse is a two-step flow. The startup scanner submits one batch reparse task through the maintenance pool, then each knowledge item is reopened through the normal reparse path. Old queued/retry tasks for that knowledge are removed, and a fresh `document:process` task is submitted to the core worker pool. Check it after every redeploy:

```bash
docker logs --since 5m lexai-thor-app-1 2>&1 \
  | grep -E 'startup-reparse|Start re-parsing knowledge|Enqueued reparse task'
```

`deploy.sh` also runs `trigger-reparse-incomplete.sh` after affected services are healthy. This covers app, docreader, model, or config redeploys where unfinished parsing must recover after the new service set is ready. The deploy script recreates `docreader` only when the docreader image or deployment config changed, waits for changed health-checked services, waits for model URLs in `WEKNORA_REPARSE_WAIT_URLS`, then submits parseable `failed` / `pending` rows, plus parseable `processing` / `finalizing` rows whose `processed_at is null`, through `POST /knowledge/batch-reparse`. This is a full-document retry for genuinely unfinished text parsing, not a stage-only retry for documents whose text has already been indexed or empty rows that cannot be parsed. Set `WEKNORA_TRIGGER_REPARSE_AFTER_DEPLOY=false` only when intentionally skipping that recovery step.

Housekeeping runs every 5 minutes in the app container. It only treats a row as drained when `pending_subtasks_count=0`, the latest attempt has no `pending/running` span, and Asynq has no queued/active task for the same knowledge and the same latest attempt. Only then does it promote `finalizing` rows to `completed`, or mark drained `summary_status=pending/processing` rows as `failed` when the knowledge row is already `completed`. Valid queued or running multimodal, Graph, Wiki, summary, or question tasks for the current attempt are not cleaned; stale tasks from older attempts do not protect the current document from recovery.

Before feature recovery and startup reparse, the app reconciles Asynq once: stale tasks from older attempts, legacy tasks without the current attempt, and byte-identical duplicate tasks are removed, then disabled-feature cleanup runs, and only genuinely missing multimodal work is recovered. Wiki trigger tasks are debounced with Asynq uniqueness while the durable per-document operations remain in `task_pending_ops`. Verify the reconciliation after a restart with `docker logs <app-container> 2>&1 | grep startup-task-reconcile`. Housekeeping only repairs document state; queue reconciliation owns stale and duplicate task removal.

Old trace attempts that still show `running` are historical rows after a newer attempt superseded them. Do not wait for old attempts; judge current progress by the latest attempt plus Asynq queue state. Stage-only recovery such as "rerun only multimodal" or "rerun only graph" requires a backend recovery entrypoint; do not describe shell-only deploys as stage-only recovery.

By default these compose templates enable `WEKNORA_SINGLE_USER_MODE=true`, so the web UI auto-creates the fixed default user space and enters the app without showing the login page. Set it to `false` to restore normal login.

New spaces default to `WEKNORA_TENANT_DEFAULT_STORAGE_QUOTA_GB=20`. Change it in `.env` or `.env.tc232` before deployment if the default storage quota should be larger. This only affects spaces created after the change; existing spaces must be updated through the system admin bulk quota action or by updating `tenants.storage_quota`.

The deployment includes a legal knowledge graph preset at `config/legal_graph_preset.json`, mounted read-only in the app container at `/app/config/legal_graph_preset.json`. LexAI is a legal-only deployment, so this preset is the default entity/relation setup for legal documents, statutes, contracts, and cases. See `../README.md` for model, Wiki, Graph, deployment, and regeneration steps; see `../legal-knowledge-graph-config.md` for the entity/relation preset details.

Wiki synthesis uses the same model as the knowledge base's main QA model by default. On `tc232` that main QA model is `lexai-vllm-qwen35-9b-awq-qa`; on thor it is `lexai-thor-vllm-qwen35-9b-qa`; on other machines it should be whichever QA model is selected for the knowledge base.

Thinking is disabled by default for LexAI QA responses. vLLM OpenAI-compatible models use `chat_template_kwargs.enable_thinking=false`. Ollama's native chat API uses `think=false`; Ollama OpenAI-compatible models should use `extra_config.thinking_control=reasoning_effort`, which sends `reasoning_effort=none`.

Knowledge graph extraction, Wiki synthesis, document summaries, table summaries, VLM/OCR multimodal parsing, and generated-question postprocessing share the main QA model. Configure worker pools, background LLM limits, model server capacity, and embedding request concurrency together; see [CONCURRENCY.md](CONCURRENCY.md). On thor, use `VLLM_MAX_MODEL_LEN=18432`, `VLLM_MAX_NUM_SEQS=7`, `WEKNORA_MAIN_QA_MODEL_CONCURRENCY=7`, `WEKNORA_CHAT_RESERVED_CONCURRENCY=3`, `WEKNORA_ASYNQ_CONCURRENCY=4`, `WEKNORA_MODEL_MAX_CONCURRENCY=1`, `WEKNORA_GRAPH_LLM_CONCURRENCY=1`, `WEKNORA_WIKI_INGEST_MAP_PARALLEL=2`, `WEKNORA_WIKI_INGEST_REDUCE_PARALLEL=2`. vLLM reports only about `3.96x` full-context concurrency at 18432 tokens on thor, so the background main-model limiter must be 1 to keep 3 usable chat slots.

On thor, keep `WEKNORA_REPARSE_WAIT_URLS=http://qwen35-9b-vllm:22222/v1/models,http://bge-m3-vllm:22223/v1/models`. Both the app startup reparse hook and `trigger-reparse-incomplete.sh` use this list so interrupted parsing is not retried before vLLM is HTTP-ready.

On thor, the default Embedding model is `lexai-thor-vllm-bge-m3-embedding`, served by `bge-m3-vllm` through the OpenAI-compatible endpoint `http://bge-m3-vllm:22223/v1`. It uses `BGE_VLLM_MAX_NUM_SEQS=8`; keep `WEKNORA_ASYNQ_CONCURRENCY=4` and `CONCURRENCY_POOL_SIZE=4` so document ingestion can use 4 embedding requests while interactive retrieval keeps about 3 service slots. Ollama `bge-m3:latest` remains configured only as a backup; see [CONCURRENCY.md](CONCURRENCY.md) for the tuning rules.

Thor Wiki generation uses the 9B QA model and keeps source text capped at 12000 characters. For Thor KBs, set `wiki_config.extraction_granularity=focused`; the deployment defaults keep Wiki ingest map/reduce parallelism at 2 unless a KB explicitly overrides `wiki_config.ingest_map_parallel` or `wiki_config.ingest_reduce_parallel`. On weaker machines, lower the Wiki map/reduce values before lowering chat reserved capacity.

For `tc232`, use the dedicated compose file. It expects the existing `qwen35-9b-awq-vllm` container to already be attached to the external `lexai` network.

```bash
cp .env.tc232.example .env.tc232
./deploy-tc232.sh
```

For `thor`, use the dedicated compose file. It creates the external `lexai` Docker network if missing, looks up the latest `thor_spark` component tags from Feishu, starts model_hub, ollama, `qwen35-9b-vllm`, and `bge-m3-vllm` on the internal network, triggers `ms://BAAI/bge-m3` through model_hub, and runs the model_hub-downloaded 9B and bge-m3 paths. Ollama remains available only as a non-default backup. See [THOR_DEPLOYMENT.md](THOR_DEPLOYMENT.md) for the reproducible host procedure.

```bash
cp .env.thor.example .env.thor
./deploy-thor.sh
```

Thor persistent data defaults to `/data/ssd/ictrek` for LexAI, model_hub, Ollama, Postgres, Redis, Neo4j, files, and docreader. The 9B vLLM mounts `/data/ssd/ictrek/models:/data/models`; the bge-m3 vLLM mounts `/data/ssd/ictrek/model_hub:/data/model_hub`. Use container paths, not host absolute symlinks.

When updating the tc232 deploy directory from this repo, use:

```bash
./sync-tc232.sh
```

Do not `rsync --delete` the whole deploy directory; remote `data/` contains the live volumes.
