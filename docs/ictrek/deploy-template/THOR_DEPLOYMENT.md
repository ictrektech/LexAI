# Thor LexAI deployment

This is the reproducible procedure used for the thor host `ictrek@192.168.1.81`.

## Files

Work from the repo directory `docs/ictrek/deploy-template`, then copy the directory to thor:

```bash
scp -r docs/ictrek/deploy-template ictrek@192.168.1.81:/home/ictrek/lexai-thor-deploy
```

On thor:

```bash
cd /home/ictrek/lexai-thor-deploy
cp .env.thor.example .env.thor
```

Edit secrets in `.env.thor`. Keep the thor model defaults unless the source doc changes:

```dotenv
VLLM_MODEL_PATH=/data/models/huggingface/hub/models--QuantTrio--Qwen3.5-9B-AWQ/snapshots/938f8e3ef86c9d1e9bec3705e149694c172592f1
VLLM_SERVED_MODEL_NAME=Qwen3.5-9B-AWQ
VLLM_GPU_MEMORY_UTILIZATION=0.45
VLLM_MAX_MODEL_LEN=18000
VLLM_MAX_NUM_SEQS=7
VLLM_MAX_NUM_BATCHED_TOKENS=4096
VLLM_MAX_JOBS=4
VLLM_ENFORCE_EAGER=true
THOR_VLLM_ENABLE_MTP=false
BGE_VLLM_MODEL_PATH=/data/model_hub/modelscope/hub/models--BAAI--bge-m3/BAAI/bge-m3
BGE_VLLM_SERVED_MODEL_NAME=bge-m3
BGE_VLLM_GPU_MEMORY_UTILIZATION=0.2
BGE_VLLM_MAX_NUM_SEQS=8
WEKNORA_ASYNQ_CONCURRENCY=4
WEKNORA_ASYNQ_QUEUE_PARSE=5
WEKNORA_ASYNQ_QUEUE_MULTIMODAL=3
WEKNORA_ASYNQ_QUEUE_GRAPH=1
WEKNORA_ASYNQ_QUEUE_QUESTION=2
WEKNORA_MAIN_QA_MODEL_CONCURRENCY=7
WEKNORA_CHAT_RESERVED_CONCURRENCY=3
WEKNORA_GRAPH_LLM_CONCURRENCY=2
WEKNORA_WIKI_INGEST_MAP_PARALLEL=2
WEKNORA_WIKI_INGEST_REDUCE_PARALLEL=2
BATCH_EMBED_SIZE=4
CONCURRENCY_POOL_SIZE=4
MAX_FILE_SIZE_MB=500
```

See [CONCURRENCY.md](CONCURRENCY.md) for the detailed machine sizing, queue, background LLM limiter, model-server capacity, and embedding concurrency rules. Thor currently uses `VLLM_MAX_MODEL_LEN=18000`, `VLLM_MAX_NUM_SEQS=7`, 3 chat slots reserved, 4 background workers, and the 8-slot bge-m3 embedding profile with 4 document embedding slots.

`VLLM_MODEL_PATH` must be a path that exists inside the vLLM container. Avoid host absolute symlinks such as `/data/ssd/ictrek/...` because the 9B container mounts `/data/ssd/ictrek/models` as `/data/models`, and the bge-m3 container mounts `/data/ssd/ictrek/model_hub` as `/data/model_hub`.

## Model preparation

The persistent model/data root is `/data/ssd/ictrek`.

If the models are missing, start only model_hub first and pull:

```bash
docker compose --env-file .env.thor -f docker-compose.thor.yml up -d model-hub-ollama model-hub-backend model-hub-frontend
```

Use model_hub or its API to pull:

```text
ollama://qwen3.5:4b
ms://BAAI/bge-m3
hf://QuantTrio/Qwen3.5-9B-AWQ
```

If Hugging Face is slow, keep:

```dotenv
HF_ENDPOINT=https://hf-mirror.com
```

Confirm the snapshot exists:

```bash
test -f /data/ssd/ictrek/models/huggingface/hub/models--QuantTrio--Qwen3.5-9B-AWQ/snapshots/938f8e3ef86c9d1e9bec3705e149694c172592f1/config.json
test -f /data/ssd/ictrek/model_hub/modelscope/hub/models--BAAI--bge-m3/BAAI/bge-m3/config.json
```

## Cleanup before deployment

Stop the services that conflict with this deployment:

```bash
docker compose --env-file .env.thor -f docker-compose.thor.yml down --remove-orphans
docker rm -f lexai-thor-emb-server-test 2>/dev/null || true
docker stop vllm ollama-gateway ollama-redis qwen3.5-0.8b asr-streaming ollama-test ollama-131 2>/dev/null || true
docker rm ollama-gateway ollama-redis qwen3.5-0.8b asr-streaming ollama-test ollama-131 2>/dev/null || true
```

Do not remove the original `vllm` container unless explicitly requested.

Check that deployment ports are free:

```bash
for port in 30080 30081 30175 30005 31434 31535 32222 32223 30074 30087; do
  ss -ltnp | grep ":${port} " || true
done
```

## Deploy

```bash
./deploy-thor.sh
```

`deploy-thor.sh` creates the `lexai` Docker network if needed, reads the latest thor component tags from Feishu, writes the image variables into `.env.thor`, and runs compose. Frontend and backend model_hub tags are resolved separately because their latest versions can differ.

`.env.thor` keeps `WEKNORA_REPARSE_INCOMPLETE_ON_START=true` and `WEKNORA_REPARSE_WAIT_URLS=http://qwen35-9b-vllm:22222/v1/models,http://bge-m3-vllm:22223/v1/models`. After every app container recreate, LexAI waits for those model endpoints and then automatically resubmits `failed`, `pending`, and `processing` knowledge rows to the batch reparse queue; `finalizing` rows are resubmitted only when `processed_at is null`. Documents whose text parsing and vector indexing already completed are not full-reparsed just because VLM/Graph/Wiki background work is still pending. This is part of the redeploy flow: model/vLLM restarts or interrupted work should recover without manual per-document retry.

Before resubmitting incomplete knowledge, app startup also compares each knowledge base's current feature switches against queued work. If a knowledge base has disabled multimodal recognition, pending/running VLM/OCR tasks for that KB are cancelled and their open spans are marked failed; if it has disabled knowledge graph extraction, pending/running Graph extraction tasks are cancelled the same way. Completed multimodal/graph outputs are kept. If multimodal recognition is enabled again, startup scans text-completed documents whose latest attempt has `multimodal` marked `skipped`, `cancelled`, or `failed`, reads the stored image links from text chunks, and re-enqueues only `image:multimodal` tasks for those images. This recovers scanned PDFs without repeating docreader, chunking, or embedding.

The deploy script recreates only managed services whose image digest or deployment config changed. Database services are not restarted. When `docreader` changes, the script waits for it to become healthy; when `app` changes, it waits for app health; when vLLM services change, it waits for `WEKNORA_REPARSE_WAIT_URLS` before running `trigger-reparse-incomplete.sh` to submit current incomplete knowledge through `POST /knowledge/batch-reparse`. This script uses the same guard as app startup: it full-reparses `failed`/`pending`/`processing`, and only reparses `finalizing` when `processed_at is null`. Use `WEKNORA_TRIGGER_REPARSE_AFTER_DEPLOY=false` only for a deliberate maintenance skip.

The startup scan is intentionally submitted to the `critical` queue. The per-document retry still lands in `parse`, after stale queued/retry tasks for the same knowledge are removed. After every redeploy, verify the app startup hook and the deploy-script reparse trigger from logs:

```bash
docker logs --since 5m lexai-thor-app-1 2>&1 \
  | grep -E 'startup-reparse|startup-feature-recovery|multimodal recovery|Start re-parsing knowledge|Enqueued reparse task|Batch knowledge reparse'
```

Old trace attempts that still show `running` are historical rows after a newer attempt superseded them. Do not wait for old attempts; judge current progress by the latest attempt plus Asynq queue state. The current built-in stage recovery covers enabled multimodal only. Graph recovery still requires full reparse or a future graph-specific recovery entrypoint.

The deployed and verified 81 plan uses these model roles:

| Role | Model ID | Service |
| --- | --- | --- |
| QA / Graph / Wiki / generated questions | `lexai-thor-vllm-qwen35-9b-qa` | `http://qwen35-9b-vllm:22222/v1` |
| VLM | `lexai-thor-vllm-qwen35-9b-vlm` | `http://qwen35-9b-vllm:22222/v1` |
| Default Embedding | `lexai-thor-vllm-bge-m3-embedding` | `http://bge-m3-vllm:22223/v1` |
| Backup Embedding | `lexai-thor-ollama-bge-m3-embedding` | `http://model-hub-ollama:11434` |

The deployed 9B default is `Qwen3.5-9B-AWQ` with `VLLM_MAX_MODEL_LEN=18000`. `Qwen3.5-9B-NVFP4` loaded weights on thor, but did not become HTTP-ready under either compile or eager mode during validation.

Default Embedding is `lexai-thor-vllm-bge-m3-embedding`, served by `bge-m3-vllm` through `http://bge-m3-vllm:22223/v1` with `interface_type=openai`. Ollama `bge-m3:latest` stays in the config as a non-default backup and is not preloaded. Keep `BGE_VLLM_MAX_NUM_SEQS=8`, `WEKNORA_ASYNQ_CONCURRENCY=4`, and `CONCURRENCY_POOL_SIZE=4` so document embedding can use 4 requests while interactive retrieval keeps about 3 service slots. The generic tuning rules are in [CONCURRENCY.md](CONCURRENCY.md).

Keep `BATCH_EMBED_SIZE=4` on thor. The app uses `CONCURRENCY_POOL_SIZE` as the document batch embedding request cap; raising it above 4 can consume the bge-m3 slots reserved for chat retrieval.

VLM/OCR multimodal parsing, Graph, Wiki, document summary, table summary, and generated-question postprocessing share the same 9B QA model. On thor, keep `WEKNORA_MAIN_QA_MODEL_CONCURRENCY=7`, `WEKNORA_CHAT_RESERVED_CONCURRENCY=3`, and `WEKNORA_ASYNQ_CONCURRENCY=4`; chat is the highest-priority path, and background LLM calls share only the remaining 4 slots. Keep `WEKNORA_ASYNQ_QUEUE_PARSE=5`, `WEKNORA_ASYNQ_QUEUE_MULTIMODAL=3`, `WEKNORA_GRAPH_LLM_CONCURRENCY=2`, `WEKNORA_ASYNQ_QUEUE_GRAPH=1`, `WEKNORA_ASYNQ_QUEUE_QUESTION=2`, `WEKNORA_WIKI_INGEST_MAP_PARALLEL=2`, and `WEKNORA_WIKI_INGEST_REDUCE_PARALLEL=2`, so document text parsing is scheduled before VLM, and VLM before Graph/Wiki enrichment while background tasks still do not crowd out generated questions or chat. If another thor-class machine changes model capacity, update both `.env.thor` and the model-service `VLLM_MAX_NUM_SEQS` according to [CONCURRENCY.md](CONCURRENCY.md).

Wiki generation uses the same 9B QA model. Keep Wiki source text capped at the application default of 12000 characters and set `wiki_config.extraction_granularity=focused`. A KB-level `wiki_config.ingest_map_parallel` or `wiki_config.ingest_reduce_parallel` overrides the Thor env defaults; keep those at `1-2` unless the 9B model has spare capacity. Larger prompts on the 9B model can return truncated JSON and leave pages ungenerated until the `wiki:ingest` task is retried.

If this is an existing database, confirm the default Embedding row after restarting `app`:

```bash
docker exec lexai-thor-postgres-1 psql -U lexai -d lexai -P pager=off -c \
  "select id,name,type,is_default,parameters->>'base_url' as base_url,parameters->>'interface_type' as interface_type from models where type='Embedding' order by is_default desc,id;"
```

Expected default row:

```text
lexai-thor-vllm-bge-m3-embedding | bge-m3 | Embedding | t | http://bge-m3-vllm:22223/v1 | openai
```

Keep automatic restart only for this LexAI deployment. To disable restart for unrelated containers without stopping them:

```bash
for c in $(docker ps -a --format '{{.Names}}'); do
  case "$c" in
    lexai-thor-*|qwen35-9b-vllm|bge-m3-vllm) continue ;;
  esac
  docker update --restart=no "$c"
done
```

## Verify

```bash
docker compose --env-file .env.thor -f docker-compose.thor.yml ps
curl -fsS http://127.0.0.1:30081/health
curl -fsS http://127.0.0.1:30005/health
curl -fsS http://127.0.0.1:32223/v1/models
curl -fsS -X POST http://127.0.0.1:32223/v1/embeddings -H 'Content-Type: application/json' -d '{"model":"bge-m3","input":["ping"]}' | python3 -c 'import json,sys; d=json.load(sys.stdin); print(len(d["data"][0]["embedding"]))'
curl -fsS http://127.0.0.1:32222/v1/models
docker inspect lexai-thor-app-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -E '^(WEKNORA_MAIN_QA_MODEL_CONCURRENCY|WEKNORA_CHAT_RESERVED_CONCURRENCY|CONCURRENCY_POOL_SIZE|BATCH_EMBED_SIZE)='
docker inspect qwen35-9b-vllm --format '{{range .Config.Cmd}}{{println .}}{{end}}' | grep -E 'max-num-seqs|max-model-len|max-num-batched-tokens|gpu-memory-utilization'
curl -sS http://127.0.0.1:32222/metrics | grep -E 'vllm:num_requests_(running|waiting)'
curl -sS http://127.0.0.1:32223/metrics | grep -E 'vllm:num_requests_(running|waiting)'
curl -I -s http://127.0.0.1:30080/ | sed -n '1,8p'
curl -I -s http://127.0.0.1:30175/app/com.ictrek.model-hub/static/css/main.css | sed -n '1,10p'
```

During ingestion, qwen running should usually stay at 4 background requests or below. It may rise above 4 when a user chat is active, but `waiting` should not stay above 0. bge-m3 should also avoid sustained `waiting > 0`; lower `CONCURRENCY_POOL_SIZE` first if it queues.

Expected externally reachable URLs:

```text
LexAI: http://192.168.1.81:30080
LexAI API: http://192.168.1.81:30081
model_hub: http://192.168.1.81:30175/app/com.ictrek.model-hub/
model_hub API: http://192.168.1.81:30005
Ollama API: http://192.168.1.81:31434
Qwen 9B vLLM OpenAI API: http://192.168.1.81:32222/v1
bge-m3 vLLM OpenAI API: http://192.168.1.81:32223/v1
```
