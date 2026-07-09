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

`deploy.sh` reads the latest image tag from each component's own Feishu column and writes image variables into `.env` before running `docker compose up -d`. The LexAI app, UI, and docreader tags are resolved independently and may be different.

All services join the `lexai` Docker network. Host ports start at 30000; service-to-service traffic uses container names inside the network.

`./deploy.sh --platform l4t` resolves component tags from the Feishu `l4t` sheet. The Ollama service is GPU-backed in these templates: `model-hub-ollama` sets `runtime: nvidia`, `NVIDIA_VISIBLE_DEVICES=all`, and `NVIDIA_DRIVER_CAPABILITIES=compute,utility`. On Jetson/Orin hosts, verify a deployment with:

```bash
docker inspect model-hub-ollama --format 'runtime={{.HostConfig.Runtime}}'
docker exec model-hub-ollama sh -lc 'ls /dev/nvhost-gpu /dev/nvmap /dev/nvhost-ctrl-gpu'
```

If `runtime` is empty or the GPU devices are missing, recreate the Ollama container from the updated compose file before testing model speed.

These deployment templates set `WEKNORA_REPARSE_INCOMPLETE_ON_START=true`. Each app container start scans knowledge rows in `failed`, `pending`, or `processing`; `finalizing` rows are full-reparsed only when `processed_at is null`. This is intentional for redeploys: interrupted or failed parsing is retried automatically after the new app is healthy, without manually clicking reparse, while text-completed documents are not forced through docreader/chunking/embedding again just because VLM/Graph/Wiki enrichment is still pending.

Startup reparse is a two-step flow. The startup scanner submits one batch task to the `critical` queue so it is not stuck behind stale document tasks from the previous container. Each knowledge item is then reopened through the normal reparse path, old queued/retry tasks for that knowledge are removed, and a fresh `document:process` task is submitted to the `parse` queue. Check it after every redeploy:

```bash
docker logs --since 5m lexai-thor-app-1 2>&1 \
  | grep -E 'startup-reparse|Start re-parsing knowledge|Enqueued reparse task'
```

`deploy.sh` also runs `trigger-reparse-incomplete.sh` after compose is healthy. This covers frontend-only or config-only redeploys where the app startup hook would not run. By default the deploy script recreates `docreader`, waits for it to become healthy, recreates `app`, waits for model URLs in `WEKNORA_REPARSE_WAIT_URLS`, then submits current `failed` / `pending` / `processing` knowledge rows, plus `finalizing` rows whose `processed_at is null`, through `POST /knowledge/batch-reparse`. This is a full-document retry for genuinely unfinished text parsing, not a stage-only retry for documents whose text has already been indexed. Set `WEKNORA_RECREATE_DOCREADER_ON_DEPLOY=false` or `WEKNORA_TRIGGER_REPARSE_AFTER_DEPLOY=false` only when intentionally skipping those steps.

Housekeeping runs every 5 minutes in the app container. It only treats a row as drained when `pending_subtasks_count=0`, the latest attempt has no `pending/running` span, and Asynq has no queued/active task for that knowledge. Only then does it promote `finalizing` rows to `completed`, or mark drained `summary_status=pending/processing` rows as `failed` when the knowledge row is already `completed`. Valid queued or running multimodal, Graph, Wiki, summary, or question tasks are not cleaned.

Old trace attempts that still show `running` are historical rows after a newer attempt superseded them. Do not wait for old attempts; judge current progress by the latest attempt plus Asynq queue state. Stage-only recovery such as "rerun only multimodal" or "rerun only graph" requires a backend recovery entrypoint; do not describe shell-only deploys as stage-only recovery.

By default these compose templates enable `WEKNORA_SINGLE_USER_MODE=true`, so the web UI auto-creates the fixed default user space and enters the app without showing the login page. Set it to `false` to restore normal login.

New spaces default to `WEKNORA_TENANT_DEFAULT_STORAGE_QUOTA_GB=20`. Change it in `.env` or `.env.tc232` before deployment if the default storage quota should be larger. This only affects spaces created after the change; existing spaces must be updated through the system admin bulk quota action or by updating `tenants.storage_quota`.

The deployment includes a legal knowledge graph preset at `config/legal_graph_preset.json`, mounted read-only in the app container at `/app/config/legal_graph_preset.json`. LexAI is a legal-only deployment, so this preset is the default entity/relation setup for legal documents, statutes, contracts, and cases. See `../README.md` for model, Wiki, Graph, deployment, and regeneration steps; see `../legal-knowledge-graph-config.md` for the entity/relation preset details.

Wiki synthesis uses the same model as the knowledge base's main QA model by default. On `tc232` that main QA model is `lexai-vllm-qwen35-9b-awq-qa`; on thor it is `lexai-thor-vllm-qwen35-9b-qa`; on other machines it should be whichever QA model is selected for the knowledge base.

Thinking is disabled by default for LexAI QA responses. vLLM OpenAI-compatible models use `chat_template_kwargs.enable_thinking=false`. Ollama's native chat API uses `think=false`; Ollama OpenAI-compatible models should use `extra_config.thinking_control=reasoning_effort`, which sends `reasoning_effort=none`.

Knowledge graph extraction, Wiki synthesis, document summaries, table summaries, VLM/OCR multimodal parsing, and generated-question postprocessing share the main QA model. Configure queue weights, background LLM limits, model server capacity, and embedding request concurrency together; see [CONCURRENCY.md](CONCURRENCY.md). On thor, use `VLLM_MAX_MODEL_LEN=18000`, `VLLM_MAX_NUM_SEQS=7`, `WEKNORA_MAIN_QA_MODEL_CONCURRENCY=7`, `WEKNORA_CHAT_RESERVED_CONCURRENCY=3`, `WEKNORA_ASYNQ_CONCURRENCY=4`, `WEKNORA_GRAPH_LLM_CONCURRENCY=2`, `WEKNORA_WIKI_INGEST_MAP_PARALLEL=2`, `WEKNORA_WIKI_INGEST_REDUCE_PARALLEL=2`, `WEKNORA_ASYNQ_QUEUE_PARSE=5`, `WEKNORA_ASYNQ_QUEUE_MULTIMODAL=3`, `WEKNORA_ASYNQ_QUEUE_GRAPH=1`, and `WEKNORA_ASYNQ_QUEUE_QUESTION=2`.

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
