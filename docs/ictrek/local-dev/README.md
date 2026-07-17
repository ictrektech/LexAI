# tc232 local-dev

LexAI/ictrek 在 tc232 上的本地快速开发覆盖，不改上游通用脚本。

## 快速启动

独立 dev volume：

```bash
./docs/ictrek/local-dev/ictrek-dev.sh setup
make dev-start DEV_ARGS="--no-langfuse --neo4j"
./docs/ictrek/local-dev/ictrek-dev.sh start-vllm
./docs/ictrek/local-dev/ictrek-dev.sh app
make dev-frontend
```

贴近 tc232 部署目录，复用 `/data/jhu/lexai-tc232-deploy/data`：

首次在当前机器准备部署目录：

```bash
cd ~/p/LexAI/docs/ictrek/deploy-template

sudo mkdir -p /data/jhu/lexai-tc232-deploy/config
sudo mkdir -p /data/jhu/lexai-tc232-deploy/data/files
sudo chown -R "$(id -u):$(id -g)" /data/jhu/lexai-tc232-deploy/data/files

sudo rsync -az \
  deploy.sh \
  deploy-tc232.sh \
  docker-compose.yml \
  docker-compose.tc232.yml \
  .env.example \
  .env.tc232.example \
  README.md \
  /data/jhu/lexai-tc232-deploy/

sudo rsync -az config/ /data/jhu/lexai-tc232-deploy/config/

cd /data/jhu/lexai-tc232-deploy
sudo cp .env.tc232.example .env.tc232

for key in DB_PASSWORD REDIS_PASSWORD JWT_SECRET TENANT_AES_KEY SYSTEM_AES_KEY CRYPTO_MASTER_KEY CRYPTO_SALT NEO4J_PASSWORD; do val=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32); sudo sed -i "s|^${key}=.*|${key}=${val}|" .env.tc232; done
```

如果是在别的机器上通过 SSH 同步，也可以从 `docs/ictrek/deploy-template` 执行 `./sync-tc232.sh`；当前机器就是部署机、或 `tc232` 主机名不可解析时，用上面的本机 `rsync` 更直接。

然后启动贴近部署的快速开发：

```bash
cd ~/p/LexAI
./docs/ictrek/local-dev/ictrek-dev.sh deploy-setup
./docs/ictrek/local-dev/ictrek-dev.sh deploy-start
./docs/ictrek/local-dev/ictrek-dev.sh deploy-app
make dev-frontend
```

回到正式部署：

```bash
./docs/ictrek/local-dev/ictrek-dev.sh deploy-stop
cd /data/jhu/lexai-tc232-deploy
./deploy-tc232.sh
```

检查环境：

```bash
./docs/ictrek/local-dev/ictrek-dev.sh check
./docs/ictrek/local-dev/ictrek-dev.sh deploy-check
```

## 关键配置

`setup` 会写入/更新根目录 `.env`：

- `DB_PORT=15432`
- `REDIS_PORT=6380`
- `DOCREADER_PORT=15051`
- `WEKNORA_SINGLE_USER_MODE=true`
- `WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL=admin@lexai.local`
- `BUILTIN_MODELS_CONFIG=docs/ictrek/local-dev/config/builtin_models.tc232.dev.yaml`
- `ICTREK_DEV_VLLM_BASE_URL=http://localhost:38118/v1`
- `ICTREK_DEV_VLLM_MAX_MODEL_LEN=65536`
- `ICTREK_DEV_VLLM_MAX_NUM_SEQS=20`
- `WEKNORA_CHAT_MODEL_CONTEXT_TOKENS=65536`
- `WEKNORA_MAIN_QA_MODEL_CONCURRENCY=20`
- `WEKNORA_MODEL_MAX_CONCURRENCY=14`
- `WEKNORA_CHAT_RESERVED_CONCURRENCY=6`
- `OLLAMA_BASE_URL=http://localhost:21436`
- `ENABLE_GRAPH_RAG=true`
- `NEO4J_ENABLE=true`

`deploy-setup` 会先把 `/data/jhu/lexai-tc232-deploy/.env.tc232` 整文件复制到根目录 `.env`，如果原来已有 `.env` 会备份成 `.env.deploy-setup.bak`。复制后会在 `.env` 末尾追加 `local-dev overrides` 注释块，只覆盖 local-dev 必须不同的少量值：

- `APP_PORT=8080`，并设置 `VITE_DEV_PROXY_TARGET=http://localhost:8080`
- `DB_PORT=15432`
- `REDIS_PORT=6380`
- `DOCREADER_PORT=15051`
- `BUILTIN_MODELS_CONFIG=docs/ictrek/local-dev/config/builtin_models.tc232.deploy-dev.yaml`
- `ICTREK_DEV_VLLM_BASE_URL=http://localhost:38118/v1`
- `ICTREK_DEV_VLLM_MAX_MODEL_LEN` 默认取 `WEKNORA_CHAT_MODEL_CONTEXT_TOKENS`，缺省为 `65536`
- `ICTREK_DEV_VLLM_MAX_NUM_SEQS` 默认取 `WEKNORA_MAIN_QA_MODEL_CONCURRENCY`，缺省为 `20`
- `WEKNORA_CHAT_MODEL_CONTEXT_TOKENS` 会和 `ICTREK_DEV_VLLM_MAX_MODEL_LEN` 保持一致
- 主模型后台并发默认按 Thor 对齐：`WEKNORA_MAIN_QA_MODEL_CONCURRENCY=20`、`WEKNORA_MODEL_MAX_CONCURRENCY=14`、`WEKNORA_CHAT_RESERVED_CONCURRENCY=6`
- worker / Wiki / embedding 应用侧并发默认按 Thor 对齐：`core=4`、`postprocess=2`、`enrichment=2`、`maintenance=1`、`shared=0`、`wiki=4`、`CONCURRENCY_POOL_SIZE=8`、`BATCH_EMBED_SIZE=8`
- `ICTREK_DEV_BGE_VLLM_BASE_URL=http://localhost:32223/v1`
- `ICTREK_DEV_OLLAMA_BASE_URL` / `OLLAMA_BASE_URL` 指向 `.env.tc232` 的 `OLLAMA_API_HOST_PORT`
- `NEO4J_URI` 改成宿主机可访问的 `bolt://localhost:${NEO4J_BOLT_PORT}`
- `LOCAL_STORAGE_BASE_DIR=/data/jhu/lexai-tc232-deploy/data/files`

这样 `.env` 默认继承部署配置，只有容器内地址、模型地址和开发端口被改成源码进程可访问的 localhost 地址。deploy-parity 的默认 Embedding 是 `lexai-vllm-bge-m3-embedding`，模型名为 `bge-m3`，由 `deploy-start` 启动的 `bge-m3-vllm` 提供；Ollama 版 `bge-m3` 只作为备用模型保留。

系统管理入口要求当前用户返回 `is_system_admin=true`。如果首次启动时日志提示 `WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL=admin@lexai.local: user lookup failed`，说明后端启动提权时默认账号还没被自动创建；先让前端自动登录一次创建 `admin@lexai.local`，再重启 `deploy-app` / 后端，并刷新或重新登录即可。

`deploy-start` 不启动部署版 `app` / `frontend`，只用部署目录的 compose 和数据目录启动开发所需的基础设施：

- PostgreSQL：`localhost:15432`，数据来自 `/data/jhu/lexai-tc232-deploy/data/postgres`
- Redis：`localhost:6380`，数据来自 `/data/jhu/lexai-tc232-deploy/data/redis`
- DocReader：`localhost:15051`，数据来自 `/data/jhu/lexai-tc232-deploy/data/docreader`
- Neo4j：沿用 `.env.tc232` 的 `NEO4J_BOLT_PORT`，默认 `localhost:30087`
- bge-m3 vLLM：默认 `localhost:32223/v1`，模型目录会优先使用含有 `hub/models--BAAI--bge-m3/snapshots/*/config.json` 的 HF 根目录；tc232 上常见路径是 `/data/jhu/cache/huggingface`
- Ollama/model-hub：沿用 `.env.tc232` 的 `OLLAMA_API_HOST_PORT`，默认 `localhost:31434`，仅作为备用模型服务

执行 `deploy-start` 时会先停掉部署版 `app`、`frontend` 和 `deploy-updater`，避免源码后端和容器后端同时消费同一套 Redis 队列。`deploy-start` 使用 local-dev 的轻量 compose 启动 Postgres、Redis、DocReader、Neo4j 和 bge-m3 vLLM，不需要 `.env.tc232` 中已有 app/frontend/model-hub/ollama 镜像变量。

bge-m3 vLLM 默认在容器内读取 `/data/huggingface/hub/models--BAAI--bge-m3`，served model name 默认为 `bge-m3`，脚本会把宿主机上可用的 HF 根目录挂到 `/data/huggingface`。如果这台机器还没有该模型缓存，先通过既有 model_hub 下载 `hf://BAAI/bge-m3`，或设置 `ICTREK_DEV_BGE_VLLM_HF_MODELS_DIR` / `BGE_VLLM_MODEL_PATH` 指向正确目录。

如果 `/data/jhu/lexai-tc232-deploy/.env.tc232` 权限不可读，先调整文件权限；不要用 `sudo` 运行 `deploy-app`，否则根目录 `.env` 和本地缓存可能变成 root 拥有。

## vLLM 参数

`start-vllm` 会把 `ICTREK_DEV_VLLM_MODEL_DIR` 指向的目录挂载到容器内 `/model`。如果该目录是 HF cache 的 `models--...` 目录，脚本会自动选择最新的 `snapshots/*/config.json` 所在目录，避免把 snapshot 父目录直接交给 vLLM。

默认主模型参数：

- `ICTREK_DEV_VLLM_MAX_MODEL_LEN=65536`
- `ICTREK_DEV_VLLM_MAX_NUM_SEQS=20`
- `ICTREK_DEV_VLLM_GPU_MEMORY_UTILIZATION=0.65`
- `ICTREK_DEV_VLLM_MAX_NUM_BATCHED_TOKENS=4096`
- `ICTREK_DEV_VLLM_ENFORCE_EAGER=true`
- `ICTREK_DEV_VLLM_ENABLE_PREFIX_CACHING=true`
- `ICTREK_DEV_VLLM_ENABLE_CHUNKED_PREFILL=true`
- `ICTREK_DEV_VLLM_ASYNC_SCHEDULING=false`

local-dev 默认只启用官方 `vllm/vllm-openai` 镜像通用参数。Thor 专用优化参数不要默认打开；如果你确认当前镜像支持，再显式设置 `ICTREK_DEV_VLLM_KV_CACHE_DTYPE`、`ICTREK_DEV_VLLM_ATTENTION_BACKEND`、`ICTREK_DEV_VLLM_MOE_BACKEND` 或 `ICTREK_DEV_VLLM_LOAD_FORMAT`。

需要临时降档压测时，可以在启动前覆盖：

```bash
ICTREK_DEV_VLLM_MAX_NUM_SEQS=8 ./docs/ictrek/local-dev/ictrek-dev.sh start-vllm
```

如果要传额外 vLLM 参数，用空格分隔写到 `ICTREK_DEV_VLLM_EXTRA_ARGS`。已有同名容器会保留创建时的旧参数；调整这些变量后需要先删容器再启动。

## 首次生成本地密钥

首次创建 `.env` 后，可以用下面命令把本地开发用的密码和密钥替换成 32 位随机字母数字：

```bash
for key in DB_PASSWORD REDIS_PASSWORD TENANT_AES_KEY SYSTEM_AES_KEY JWT_SECRET; do val=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32); if grep -q "^${key}=" .env; then sed -i "s|^${key}=.*|${key}=${val}|" .env; else printf '\n%s=%s\n' "$key" "$val" >> .env; fi; done
```

检查长度：

```bash
awk -F= '/^(DB_PASSWORD|REDIS_PASSWORD|TENANT_AES_KEY|SYSTEM_AES_KEY|JWT_SECRET)=/ { print $1, length($2) }' .env
```

注意：如果 Postgres / Redis 的 dev volume 已经初始化过，修改 `DB_PASSWORD` 或 `REDIS_PASSWORD` 后需要清理对应 volume，或者继续使用旧密码，否则后端可能认证失败。`SYSTEM_AES_KEY` 会影响已加密配置的解密，已有数据环境不要随意轮换。

如果已有同名容器，脚本会复用它。要让新的启动参数生效，先删除旧容器：

```bash
docker rm -f qwen35-9b-awq-vllm
./docs/ictrek/local-dev/ictrek-dev.sh start-vllm
```

## 地址

- 前端：`http://localhost:5173`
- 后端：`http://localhost:8080`
- vLLM：`http://localhost:38118/v1`
- bge-m3 vLLM：`http://localhost:32223/v1`
