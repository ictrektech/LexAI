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
VLLM_MAX_MODEL_LEN=20480
VLLM_MAX_NUM_SEQS=6
VLLM_MAX_NUM_BATCHED_TOKENS=4096
VLLM_MAX_JOBS=4
VLLM_ENFORCE_EAGER=true
THOR_VLLM_ENABLE_MTP=false
BGE_VLLM_MODEL_PATH=/data/model_hub/modelscope/hub/models--BAAI--bge-m3/BAAI/bge-m3
BGE_VLLM_SERVED_MODEL_NAME=bge-m3
BGE_VLLM_GPU_MEMORY_UTILIZATION=0.2
BGE_VLLM_MAX_NUM_SEQS=12
WEKNORA_ASYNQ_CONCURRENCY=9
BATCH_EMBED_SIZE=4
CONCURRENCY_POOL_SIZE=9
```

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
ollama://bge-m3:latest
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

The deployed 9B default is `Qwen3.5-9B-AWQ` with `VLLM_MAX_MODEL_LEN=20480`. `Qwen3.5-9B-NVFP4` loaded weights on thor, but did not become HTTP-ready under either compile or eager mode during validation.

Default Embedding is `lexai-thor-vllm-bge-m3-embedding`, served by `bge-m3-vllm` through `http://bge-m3-vllm:22223/v1` with `interface_type=openai`. Ollama `bge-m3:latest` stays in the config as a non-default backup. Keep `BGE_VLLM_MAX_NUM_SEQS=12`, `WEKNORA_ASYNQ_CONCURRENCY=9`, and `CONCURRENCY_POOL_SIZE=9` so document embedding can use 9 requests while interactive retrieval keeps 3 service slots.

Keep `BATCH_EMBED_SIZE=4` on thor. The app uses `CONCURRENCY_POOL_SIZE` as the document batch embedding request cap; setting it below the document worker count can make background parsing appear stuck in the `embedding` stage.

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
curl -I -s http://127.0.0.1:30080/ | sed -n '1,8p'
curl -I -s http://127.0.0.1:30175/app/com.ictrek.model-hub/static/css/main.css | sed -n '1,10p'
```

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
