# LexAI deploy template

Copy `.env.example` to `.env`, edit secrets, then run:

```bash
./deploy.sh --platform amd
./deploy.sh --platform l4t
./deploy.sh --platform thor
```

`deploy.sh` reads the latest image tag from each component's own Feishu column and writes image variables into `.env` before running `docker compose up -d`. The LexAI app, UI, and docreader tags are resolved independently and may be different.

All services join the `lexai` Docker network. Host ports start at 30000; service-to-service traffic uses container names inside the network.

By default these compose templates enable `WEKNORA_SINGLE_USER_MODE=true`, so the web UI auto-creates the fixed default user space and enters the app without showing the login page. Set it to `false` to restore normal login.

New spaces default to `WEKNORA_TENANT_DEFAULT_STORAGE_QUOTA_GB=20`. Change it in `.env` or `.env.tc232` before deployment if the default storage quota should be larger. This only affects spaces created after the change; existing spaces must be updated through the system admin bulk quota action or by updating `tenants.storage_quota`.

The deployment includes a legal knowledge graph preset at `config/legal_graph_preset.json`, mounted read-only in the app container at `/app/config/legal_graph_preset.json`. LexAI is a legal-only deployment, so this preset is the default entity/relation setup for legal documents, statutes, contracts, and cases. See `../README.md` for model, Wiki, Graph, deployment, and regeneration steps; see `../legal-knowledge-graph-config.md` for the entity/relation preset details.

Wiki synthesis uses the same model as the knowledge base's main QA model by default. On `tc232` that main QA model is `lexai-vllm-qwen35-9b-awq-qa`; on other machines it should be whichever QA model is selected for the knowledge base.

Thinking is disabled by default for LexAI QA responses. vLLM OpenAI-compatible models use `chat_template_kwargs.enable_thinking=false`. Ollama's native chat API uses `think=false`; Ollama OpenAI-compatible models should use `extra_config.thinking_control=reasoning_effort`, which sends `reasoning_effort=none`.

Knowledge graph extraction shares the main QA model. On thor, vLLM is capped at 6 sequences, so set `WEKNORA_GRAPH_LLM_CONCURRENCY=3` to leave capacity for interactive chat.

On thor, bge-m3 runs as a separate OpenAI-compatible vLLM embedding service, `bge-m3-vllm`, with `BGE_VLLM_MAX_NUM_SEQS=6`. Set `WEKNORA_ASYNQ_CONCURRENCY=3` so document ingestion can use up to half of that service while interactive retrieval still has capacity.

For `tc232`, use the dedicated compose file. It expects the existing `qwen35-9b-awq-vllm` container to already be attached to the external `lexai` network.

```bash
cp .env.tc232.example .env.tc232
./deploy-tc232.sh
```

For `thor`, use the dedicated compose file. It creates the external `lexai` Docker network if missing, looks up the latest `thor_spark` component tags from Feishu, starts model_hub, ollama, `qwen35-9b-vllm`, and `bge-m3-vllm` on the internal network, triggers `ollama://bge-m3:latest` and `ms://BAAI/bge-m3` through model_hub, and runs the model_hub-downloaded 9B and bge-m3 paths. See [THOR_DEPLOYMENT.md](THOR_DEPLOYMENT.md) for the reproducible host procedure.

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
