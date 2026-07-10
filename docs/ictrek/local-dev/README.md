# tc232 local-dev

LexAI/ictrek 在 tc232 上的本地快速开发覆盖，不改上游通用脚本。

## 快速启动

```bash
./docs/ictrek/local-dev/ictrek-dev.sh setup
make dev-start DEV_ARGS="--no-langfuse --neo4j"
./docs/ictrek/local-dev/ictrek-dev.sh start-vllm
./docs/ictrek/local-dev/ictrek-dev.sh app
make dev-frontend
```

检查环境：

```bash
./docs/ictrek/local-dev/ictrek-dev.sh check
```

## 关键配置

`setup` 会写入/更新根目录 `.env`：

- `DB_PORT=15432`
- `REDIS_PORT=6380`
- `DOCREADER_PORT=15051`
- `BUILTIN_MODELS_CONFIG=docs/ictrek/local-dev/config/builtin_models.tc232.dev.yaml`
- `ICTREK_DEV_VLLM_BASE_URL=http://localhost:38118/v1`
- `OLLAMA_BASE_URL=http://localhost:21436`
- `ENABLE_GRAPH_RAG=true`
- `NEO4J_ENABLE=true`

如果已有同名容器，脚本会复用它。要让新的启动参数生效，先删除旧容器：

```bash
docker rm -f qwen35-9b-awq-vllm
./docs/ictrek/local-dev/ictrek-dev.sh start-vllm
```

## 地址

- 前端：`http://localhost:5173`
- 后端：`http://localhost:8080`
- vLLM：`http://localhost:38118/v1`
