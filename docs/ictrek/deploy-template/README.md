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

By default these compose templates enable `WEKNORA_SINGLE_USER_MODE=true`, so the web UI auto-creates the fixed default user space and enters the app without showing the login page. Set it to `false` to restore normal login.

New spaces default to `WEKNORA_TENANT_DEFAULT_STORAGE_QUOTA_GB=20`. Change it in `.env` or `.env.tc232` before deployment if the default storage quota should be larger. This only affects spaces created after the change; existing spaces must be updated through the system admin bulk quota action or by updating `tenants.storage_quota`.

The deployment includes a legal knowledge graph preset at `config/legal_graph_preset.json`, mounted read-only in the app container at `/app/config/legal_graph_preset.json`. LexAI is a legal-only deployment, so this preset is the default entity/relation setup for legal documents, statutes, contracts, and cases. See `../README.md` for model, Wiki, Graph, deployment, and regeneration steps; see `../legal-knowledge-graph-config.md` for the entity/relation preset details.

Wiki synthesis uses the same model as the knowledge base's main QA model by default. On `tc232` that main QA model is `lexai-vllm-qwen35-9b-awq-qa`; on thor it is `lexai-thor-vllm-qwen35-9b-qa`; on other machines it should be whichever QA model is selected for the knowledge base.

Thinking is disabled by default for LexAI QA responses. vLLM OpenAI-compatible models use `chat_template_kwargs.enable_thinking=false`. Ollama's native chat API uses `think=false`; Ollama OpenAI-compatible models should use `extra_config.thinking_control=reasoning_effort`, which sends `reasoning_effort=none`.

Knowledge graph extraction, Wiki synthesis, document summaries, table summaries, and generated-question postprocessing share the main QA model. Configure queue weights, background LLM limits, model server capacity, and embedding request concurrency together; see [CONCURRENCY.md](CONCURRENCY.md). On thor, use `VLLM_MAX_NUM_SEQS=8`, `WEKNORA_MAIN_QA_MODEL_CONCURRENCY=8`, `WEKNORA_CHAT_RESERVED_CONCURRENCY=3`, `WEKNORA_GRAPH_LLM_CONCURRENCY=2`, `WEKNORA_WIKI_INGEST_MAP_PARALLEL=2`, `WEKNORA_WIKI_INGEST_REDUCE_PARALLEL=2`, `WEKNORA_ASYNQ_QUEUE_PARSE=5`, `WEKNORA_ASYNQ_QUEUE_GRAPH=2`, and `WEKNORA_ASYNQ_QUEUE_QUESTION=2`.

On thor, the default Embedding model is `lexai-thor-vllm-bge-m3-embedding`, served by `bge-m3-vllm` through the OpenAI-compatible endpoint `http://bge-m3-vllm:22223/v1`. It uses `BGE_VLLM_MAX_NUM_SEQS=8`; keep `WEKNORA_ASYNQ_CONCURRENCY=9` and `CONCURRENCY_POOL_SIZE=5` so document ingestion can use 5 embedding requests while interactive retrieval keeps about 3 service slots. Ollama `bge-m3:latest` remains configured only as a backup; see [CONCURRENCY.md](CONCURRENCY.md) for the tuning rules.

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
