# LexAI ictrek 部署 README

本文说明 ictrek 部署包需要哪些文件、如何修改配置、如何自动检测或手动填写镜像版本，以及 LexAI 法律部署中 QA 模型、Wiki 生成、知识图谱抽取之间的关系。

开发、合并上游、构建镜像和 push 流程见 [开发文档](DEVELOPMENT.md)。

## 机器资源评估入口

任意机器部署前，先看 [deploy-template/CONCURRENCY.md](deploy-template/CONCURRENCY.md)。它是模型大小、上下文长度、vLLM 并发、聊天预留、后台 worker 池、后台模型并发和 Embedding 并发的统一参考。Thor 部署尤其先看其中「管理界面参数和 env 对照」「Thor 当前参数：每个数字限制什么」和「机器资源评估流程」。当前 tc97 Thor 配置里，`12` 是 QA vLLM 接收上限，`6` 是聊天目标预留，另一个 `6` 是后台进入主 QA 模型的总并发上限；这些值不限制在线聊天或 embedding。

资源参数不要只改一个数字。按下面顺序设置，确保部署文档、env 和 compose 里没有旧值残留：

1. 先定模型和上下文：`VLLM_MAX_MODEL_LEN` 必须等于 `WEKNORA_CHAT_MODEL_CONTEXT_TOKENS`，tc97 当前是精确 `18432`，不是近似 `18000`。
2. 再定主模型入口：`VLLM_MAX_NUM_SEQS` 必须等于 `WEKNORA_MAIN_QA_MODEL_CONCURRENCY`，tc97 当前是 `12`。
3. 再定聊天预留：`WEKNORA_CHAT_RESERVED_CONCURRENCY` 是在线聊天目标预留，tc97 当前是 `6`；更小机器至少保留 `2-3`。
4. 再定后台主模型闸门：`WEKNORA_MODEL_MAX_CONCURRENCY` 按 `min(VLLM_MAX_NUM_SEQS, vLLM 满长有效并发) - WEKNORA_CHAT_RESERVED_CONCURRENCY` 计算，tc97 当前是 `6`。Graph、Wiki、摘要、自动问题和 VLM 后台请求都必须受它限制。
5. 最后定 worker 池：core 负责文字解析和向量化，postprocess 负责派发后处理，enrichment 负责 VLM/Graph/Question/Summary，maintenance 负责批处理和清理，wiki 单独跑 Wiki。tc97 当前是 `4/2/2/1/0/4`，shared 为 `0`，避免后台增强绕过固定池。
6. Embedding 独立设置：`BGE_VLLM_MAX_NUM_SEQS` 是 bge 服务上限，`CONCURRENCY_POOL_SIZE` 是文档 embedding 应用侧并发，`BATCH_EMBED_SIZE` 是单次请求打包 chunk 数；tc97 当前是 `16/8/8`，给在线检索留余量。

## 部署包范围

部署目标机不需要整个 repo。按场景只同步必要文件：

| 场景 | 需要的文件 | 不需要的文件 | 主要改哪里 |
| --- | --- | --- | --- |
| Thor `jhu@192.168.1.97`，完整启动 LexAI + model_hub + qwen35-9b-vLLM + bge-m3-vLLM | [deploy-thor.sh](deploy-template/deploy-thor.sh)、[docker-compose.thor.yml](deploy-template/docker-compose.thor.yml)、[.env.thor.example](deploy-template/.env.thor.example)、[config/builtin_models.thor.yaml](deploy-template/config/builtin_models.thor.yaml)、[config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json)、[THOR_DEPLOYMENT.md](deploy-template/THOR_DEPLOYMENT.md)、[CONCURRENCY.md](deploy-template/CONCURRENCY.md) | 通用 [docker-compose.yml](deploy-template/docker-compose.yml) 和 `.env.example` 不是 Thor 运行入口；tc232 文件也不用 | `.env.thor` 的密钥、端口、`/data/jhu/dev/workspace/lexai` 数据目录、vLLM 模型路径、bge-m3 路径、并发/队列参数；具体按 [THOR_DEPLOYMENT.md](deploy-template/THOR_DEPLOYMENT.md) 和 [CONCURRENCY.md](deploy-template/CONCURRENCY.md) |
| tc232，已有 `qwen35-9b-awq-vllm` | [deploy-tc232.sh](deploy-template/deploy-tc232.sh)、[docker-compose.tc232.yml](deploy-template/docker-compose.tc232.yml)、[.env.tc232.example](deploy-template/.env.tc232.example)、[config/](deploy-template/config/)、[CONCURRENCY.md](deploy-template/CONCURRENCY.md)；本地同步可用 [sync-tc232.sh](deploy-template/sync-tc232.sh) | [docker-compose.yml](deploy-template/docker-compose.yml) 里的 vllm 服务不会用；`.env.example` 不是运行入口 | `.env.tc232` 的端口、密钥、数据目录、并发；如已有 vllm 容器名不同，改 [config/builtin_models.yaml](deploy-template/config/builtin_models.yaml) 里的 `base_url` |
| 全新主机，没有可用 vllm | [deploy.sh](deploy-template/deploy.sh)、[docker-compose.yml](deploy-template/docker-compose.yml)、[.env.example](deploy-template/.env.example)、[config/](deploy-template/config/)、[CONCURRENCY.md](deploy-template/CONCURRENCY.md) | `deploy-tc232.sh`、`docker-compose.tc232.yml`、`.env.tc232.example` | `.env` 的 `VLLM_HOST_PORT`、`VLLM_HF_MODELS_DIR`、模型目录、端口、密钥、并发 |
| 主机已有可用 vllm，且同一个模型同时支持文本和 VLM | 以 [docker-compose.tc232.yml](deploy-template/docker-compose.tc232.yml) 为模板另存一份机器专用 compose，配套一份 env，再带上 [config/](deploy-template/config/) 和 [CONCURRENCY.md](deploy-template/CONCURRENCY.md) | 通用 compose 里的 `qwen35-9b-awq-vllm` 服务不需要启动 | 把 [config/builtin_models.yaml](deploy-template/config/builtin_models.yaml) 中 QA 和 Vision 模型的 `base_url` 都指向已有 vllm 容器名；模型 ID 可以共用同一个 served model |
| 只手动固定镜像版本 | 对应场景的 compose、env、[config/](deploy-template/config/)、[CONCURRENCY.md](deploy-template/CONCURRENCY.md) | [deploy.sh](deploy-template/deploy.sh) 可不用 | 直接在 env 里填写 `LEXAI_APP_IMAGE`、`LEXAI_UI_IMAGE`、`LEXAI_DOCREADER_IMAGE` 等镜像变量 |

[config/builtin_models.yaml](deploy-template/config/builtin_models.yaml) 和 [config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json) 两个配置文件建议通用/tc232 部署场景都带上。Thor 使用 [config/builtin_models.thor.yaml](deploy-template/config/builtin_models.thor.yaml) 替代通用模型文件。模型文件注册默认模型，图谱文件保留法律图谱实体/关系模板。

Agent 的「技能 Skills」依赖后端 sandbox。通用、tc232、Thor 三套 compose 模板都默认设置 `WEKNORA_SANDBOX_MODE=docker`，并把宿主机 `/var/run/docker.sock` 挂到 app 容器；[deploy-template/deploy.sh](deploy-template/deploy.sh) 会读取飞书 `lexai-sandbox` 列并预拉 `WEKNORA_SANDBOX_DOCKER_IMAGE`，避免部署后创建智能体时 Skills 菜单消失。除非明确不允许执行 Skill，否则不要把 `WEKNORA_SANDBOX_MODE` 改成 `disabled`。

只有在重新构建镜像、修改前端默认模板或修改后端源码时，才需要完整 repo、Dockerfile 和源码目录。单纯部署、改端口、改模型、改图谱模板，不需要把整个 repo 放到目标机。

## 初次部署需要改哪里

全新主机部署，即主机没有可用 vllm：

1. 把 [deploy-template](deploy-template/) 目录同步到目标机，例如 `/data/jhu/lexai-deploy`。
2. 在目标机执行 `cp .env.example .env`。
3. 编辑 `.env`，至少检查密钥、端口、模型目录和并发：

```bash
DB_PASSWORD=...
REDIS_PASSWORD=...
JWT_SECRET=...
TENANT_AES_KEY=0123456789abcdef0123456789abcdef
SYSTEM_AES_KEY=0123456789abcdef0123456789abcdef
FRONTEND_PORT=30080
MODEL_HUB_FRONTEND_PORT=30175
OLLAMA_MODELS_DIR=/data/ictrek_models/ollama/models
MODEL_HUB_HF_MODELS_DIR=/data/jhu/models/huggingface
VLLM_HF_MODELS_DIR=/data/jhu/models/huggingface
WEKNORA_MAIN_QA_MODEL_CONCURRENCY=4
WEKNORA_CHAT_RESERVED_CONCURRENCY=1
WEKNORA_GRAPH_LLM_CONCURRENCY=2
WEKNORA_WIKI_INGEST_MAP_PARALLEL=1
WEKNORA_WIKI_INGEST_REDUCE_PARALLEL=1
MAX_FILE_SIZE_MB=500
WEKNORA_TENANT_DEFAULT_STORAGE_QUOTA_GB=20
ENABLE_GRAPH_RAG=true
NEO4J_ENABLE=true
NEO4J_URI=bolt://neo4j:7687
```

tc232 部署，即复用 `lexai` 网络里已有的 `qwen35-9b-awq-vllm`：

1. 本地执行 [sync-tc232.sh](deploy-template/sync-tc232.sh)，会同步到 `tc232:/data/jhu/lexai-tc232-deploy`。
2. 在 tc232 执行 `cp .env.tc232.example .env.tc232`，如果已有 `.env.tc232` 就只补缺失项。
3. tc232 不需要配置 `VLLM_*` 镜像服务，compose 会通过 `http://qwen35-9b-awq-vllm:8000/v1` 访问 `lexai` 网络里已有的 vllm。

Thor 部署，即 97 上完整启动 LexAI、model_hub、qwen35-9b-vLLM、bge-m3-vLLM：

1. 把 [deploy-template](deploy-template/) 目录同步到 `jhu@192.168.1.97:/data/jhu/dev/workspace/lexai/deploy`。
2. 在 97 执行 `cp .env.thor.example .env.thor`，如果已有 `.env.thor` 就只补缺失项和新版并发变量。
3. 按 [THOR_DEPLOYMENT.md](deploy-template/THOR_DEPLOYMENT.md) 检查 `/data/jhu/dev/workspace/lexai`、模型路径、vLLM 参数、默认 Embedding、队列/并发参数后执行 `./deploy-thor.sh`。

其他已有 vllm 的主机：

1. 复制 [docker-compose.tc232.yml](deploy-template/docker-compose.tc232.yml) 为该机器专用 compose。
2. 删除文件名里的 tc232 语义后，保留“没有 vllm 服务、通过容器名访问外部 vllm”的结构。
3. 在 [config/builtin_models.yaml](deploy-template/config/builtin_models.yaml) 中把 QA 和 Vision 模型的 `base_url` 改成已有 vllm 的 OpenAI API 地址，例如 `http://your-vllm-container:8000/v1`。
4. 如果已有 vllm 模型像 qwen3.5 一样同时支持文本和 VLM，QA 和 Vision 两条模型配置可以指向同一个 `base_url` 和同一个 served model，只保持不同模型 ID，方便前端分别选择。

## 自动检测最新镜像部署

自动模式适合目标机可以访问飞书表格配置的情况。运行脚本前，执行部署脚本的机器需要有：

- `docker` / `docker compose`
- `curl`
- `python3`
- `~/.feishu.components.json` 或 `~/.feishu.json`，包含 `feishu_app_id` 和 `feishu_app_secret`

[deploy.sh](deploy-template/deploy.sh) 只读飞书表格时会优先使用 `FEISHU_READ_CONFIG_FILE`，默认是 `~/.feishu.components.json`；如果该配置不存在或没有读表权限，会回退到 `FEISHU_CONFIG_FILE`，默认是 `~/.feishu.json`。构建并更新飞书表格的写操作不走这个脚本，仍按 [开发文档](DEVELOPMENT.md) 使用 `build_image.sh` 和 `~/.feishu.json`。

通用部署命令：

```bash
cd /data/jhu/lexai-deploy
./deploy.sh --platform amd
./deploy.sh --platform l4t
```

如果需要指定表格 sheet：

```bash
./deploy.sh --platform amd --sheet AMD_with_cuda
```

只检查将会使用哪些镜像、不实际部署：

```bash
./deploy.sh --platform amd --dry-run
```

已有部署目录的一键更新：

```bash
./update-and-deploy.sh --platform thor --check-only
./update-and-deploy.sh --platform thor
./update-and-deploy.sh --platform amd
./update-and-deploy.sh --platform l4t
```

[update-and-deploy.sh](deploy-template/update-and-deploy.sh) 的 `--check-only` 只读取飞书表格里的 `lexai`、`lexai-ui`、`lexai-docreader`、`lexai-sandbox` 四个 LexAI 运行镜像版本，并在版本相同时比较远端镜像 digest 是否变化；检测阶段不拉取 Git 仓库、不同步部署文件、不写入 `.env`、不重建容器。`lexai-sandbox` 不是独立服务，检测到它更新时会提示替换 app，因为 app 需要重新创建才能使用新的 `WEKNORA_SANDBOX_DOCKER_IMAGE`。输出 `UPDATE_AVAILABLE`、`UPDATE_SERVICES`、`UPDATE_DETAILS`，用于界面提示当前版本、最新版本和同版本镜像内容变化。

用户确认更新后，脚本才从 `LEXAI_DEPLOY_REPO` / `LEXAI_DEPLOY_REF` 拉取最新 `docs/ictrek`，同步 `deploy-template` 到当前部署目录并保留本机 `.env`、`.env.tc232`、`.env.thor`，随后 `docker pull` 飞书表格解析出的镜像并精确替换需要更新的 LexAI app、frontend、docreader 和 deploy-updater 服务。sandbox 镜像只会通过 app 环境变量生效，不会创建 `lexai-sandbox` 容器。不会重启 Postgres、Redis、Neo4j 等数据库服务；如果同次更新 app 和 frontend，会先等待 app 健康，再重建 frontend，避免 frontend 反代到旧 app 容器 IP。

界面右上角的“检测更新”按钮通过 `WEKNORA_DEPLOY_UPDATER_CONTAINER` 固定调用 `deploy-updater` sidecar，先执行 `--check-only` 并向用户列出镜像版本变化和将替换的服务；用户确认后才启动后台更新。不接收前端传入的脚本路径、compose 文件或服务名，避免匹配错容器。sidecar 使用当前部署目录的 `/lexai-deploy/update-and-deploy.sh`，日志写入部署目录的 `update-and-deploy.log`，前端会轮询显示拉取镜像和替换容器进度。更新期间页面可能短暂不可用，用户应等待一段时间后手动刷新页面。确保 `.env*` 里的 `DEPLOY_UPDATER_CONTAINER` 唯一，`FEISHU_CONFIG_HOST_FILE` 指向宿主机可读的飞书凭据文件。

`deploy-updater` 没有单独的 `lexai-deploy-updater` 镜像，它复用 `lexai` app 镜像，并额外挂载宿主机 Docker socket、部署目录和飞书凭据。这样更新按钮调用的后端逻辑和 app 版本一致，也避免维护额外的 updater 镜像。若 app 镜像更新，`deploy.sh` 会在本次更新完成后延迟刷新 `deploy-updater` sidecar；否则 sidecar 会继续运行旧 app 镜像，后续检测更新可能仍执行旧逻辑。

`app` 和 `deploy-updater` 会在容器内调用宿主机 Docker daemon，因此 app 镜像内置的 Docker CLI 不能低于宿主 daemon 的最低 API 要求。不是要求容器内 Docker 版本必须比宿主机新，而是必须满足 `Client.APIVersion >= Server.MinAPIVersion`。默认 app 镜像内置 Docker CLI `29.1.3`；构建时可用 `DOCKER_CLI_VERSION` 覆盖。部署后用下面命令检查，不能出现 `client version ... is too old`：

```bash
docker exec lexai-thor-app-1 docker version \
  --format 'client={{.Client.Version}} api={{.Client.APIVersion}} server_min={{.Server.MinAPIVersion}}'
docker exec lexai-thor-deploy-updater docker version \
  --format 'client={{.Client.Version}} api={{.Client.APIVersion}} server_min={{.Server.MinAPIVersion}}'
```

tc232 专用部署：

```bash
cd /data/jhu/lexai-tc232-deploy
./deploy-tc232.sh
```

Thor 专用部署：

```bash
cd /data/jhu/dev/workspace/lexai/deploy
./deploy-thor.sh
```

[deploy.sh](deploy-template/deploy.sh) 会分别查找这些组件的最新镜像，允许 LexAI 组件、sandbox runtime 和 model_hub/ollama 使用不同版本：

```text
LEXAI_APP_IMAGE
LEXAI_UI_IMAGE
LEXAI_DOCREADER_IMAGE
WEKNORA_SANDBOX_DOCKER_IMAGE
MODEL_HUB_BACKEND_IMAGE
MODEL_HUB_FRONTEND_IMAGE
OLLAMA_SERVER_IMAGE
```

脚本会把查到的镜像写回 `.env`、`.env.tc232` 或 `.env.thor`，然后执行对应 compose 文件的 `up -d`。

`--platform l4t` 默认读取飞书表格里的 `l4t` sheet。全 Ollama 模板中的 `model-hub-ollama` 已显式配置 `runtime: nvidia`、`NVIDIA_VISIBLE_DEVICES=all` 和 `NVIDIA_DRIVER_CAPABILITIES=compute,utility`，避免 Orin NX / Jetson 主机按 Docker 默认 `runc` 启动后模型落到 CPU。部署后可用下面命令确认：

```bash
docker inspect model-hub-ollama --format 'runtime={{.HostConfig.Runtime}}'
docker exec model-hub-ollama sh -lc 'ls /dev/nvhost-gpu /dev/nvmap /dev/nvhost-ctrl-gpu'
```

## 手动填写镜像部署

如果目标机没有飞书凭据，或者要固定某一批镜像版本，不要运行 `deploy.sh` / `deploy-tc232.sh` / `deploy-thor.sh`。直接编辑对应 env，手动填入镜像：

```bash
LEXAI_APP_IMAGE=registry.example.com/lexai:xxx
LEXAI_UI_IMAGE=registry.example.com/lexai-ui:xxx
LEXAI_DOCREADER_IMAGE=registry.example.com/lexai-docreader:xxx
WEKNORA_SANDBOX_MODE=docker
WEKNORA_SANDBOX_DOCKER_IMAGE=registry.example.com/lexai-sandbox:xxx
MODEL_HUB_BACKEND_IMAGE=registry.example.com/model_hub_backend:xxx
MODEL_HUB_FRONTEND_IMAGE=registry.example.com/model_hub_frontend:xxx
OLLAMA_SERVER_IMAGE=registry.example.com/ollama_server:xxx
```

然后直接用 compose 部署：

```bash
docker compose --env-file .env -f docker-compose.yml up -d
```

tc232：

```bash
docker compose --env-file .env.tc232 -f docker-compose.tc232.yml up -d
```

Thor：

```bash
docker compose --env-file .env.thor -f docker-compose.thor.yml up -d
```

注意：再次运行 `deploy.sh`、`deploy-tc232.sh` 或 `deploy-thor.sh` 会重新从飞书表格检测镜像并覆盖 env 中的镜像变量。需要完全手动固定版本时，只运行 `docker compose ... up -d`。

## 已部署环境更新

已部署环境改配置后，一般只需要同步部署模板文件并重新 `up -d`：

- 改 `.env` / `.env.tc232` / `.env.thor`：重新执行对应 compose `up -d`。
- 改 `WEKNORA_TENANT_DEFAULT_STORAGE_QUOTA_GB`：只影响之后新建的空间；已有空间不会自动变化。要同步到已有空间，需要系统管理员调用“批量应用默认存储配额”，或直接更新 `tenants.storage_quota`。
- 改 [config/builtin_models.yaml](deploy-template/config/builtin_models.yaml) 或 Thor 的 [config/builtin_models.thor.yaml](deploy-template/config/builtin_models.thor.yaml)：同步文件后重启 `app` 服务。
- 改 [config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json)：同步文件后重启 `app` 服务，新建或重新保存知识库图谱配置后生效。
- 改 [docker-compose.yml](deploy-template/docker-compose.yml)、[docker-compose.tc232.yml](deploy-template/docker-compose.tc232.yml) 或 [docker-compose.thor.yml](deploy-template/docker-compose.thor.yml)：重新执行对应 compose `up -d`。
- 改 [frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts)：这是前端默认值，必须重新构建并部署 `lexai-ui` 镜像。
- 改 `docreader/` 里的解析逻辑，例如 PDF 文本层乱码检测、扫描页渲染策略、文档格式解析器：必须重新构建并部署 `lexai-docreader` 镜像，再重建 `docreader` 容器；只重启旧镜像不会生效。

部署数据不应该跟 repo 一起同步。Postgres、Redis、Neo4j、Qdrant、上传文件、ollama 模型、HF 模型都通过 compose volume 或宿主机目录保存。换新镜像只要复用同一套 `.env`、compose 和数据目录，数据会恢复。

部署模板默认开启 `WEKNORA_REPARSE_INCOMPLETE_ON_START=true`。每次 app 容器重建或重启后，服务会自动扫描有可解析来源的 `failed`、`pending`、`processing` 知识条目；`finalizing` 只有在 `processed_at is null` 时才会整篇重新解析。可解析来源指上传文件有 `file_path`，`file_url` / `url` 有 `source`，手工知识有非空 `metadata.content`。已经完成文字解析和向量入库、只是停在 VLM/Graph/Wiki 后台增强的文档不会重复跑 docreader、分块和 embedding；没有实际来源的空记录不会被自动重跑。每条知识重新解析前会清理该知识残留的 queued/retry 任务，再提交新的 `document:process` 到 core worker 池。这个行为适用于通用、tc232 和 Thor 模板。已完成、已取消、删除中的知识不会被自动重跑。

部署脚本只重建镜像 digest 或部署配置发生变化的受管服务，不重启 Postgres、Redis、Neo4j 等数据库服务。`docreader` 只有在 docreader 镜像或部署配置变化时才重建；`app` 变化时等待 app health；vLLM 变化时等待 `WEKNORA_REPARSE_WAIT_URLS` 里的模型服务 ready。随后脚本运行 [deploy-template/trigger-reparse-incomplete.sh](deploy-template/trigger-reparse-incomplete.sh)，把当前失败/未完成文档通过批量 reparse API 重新提交。详细验证命令见 [deploy-template/README.md](deploy-template/README.md) 和 [deploy-template/THOR_DEPLOYMENT.md](deploy-template/THOR_DEPLOYMENT.md)。

后台 housekeeping 每 5 分钟还会清理已经没有待完成工作的残留状态：`finalizing + pending_subtasks_count=0` 只有在最新 attempt 没有 `pending/running` span、并且 Asynq 队列里也没有该知识的 queued/active 任务时，才会推进为 `completed`，避免文档文字已入库但页面长期显示「优化中」。同理，`completed + pending_subtasks_count=0 + summary_status in (pending, processing)` 也只有在没有 open span 和 queued/active 任务时，才会把摘要状态标记为 `failed`，避免没有摘要任务可跑时页面长期显示「生成摘要中」。仍在排队或运行的多模态、Graph、Wiki、摘要任务不会被 housekeeping 清掉。

app 启动时会先对 Asynq 队列做一次对账：删除已经被新 attempt 替代的任务、没有当前 attempt 的旧格式任务和完全相同的重复任务，再清理已关闭功能的任务，最后才恢复确实缺失的多模态任务和未完成文字解析。多模态恢复检测到同一文档已有多模态任务时不会重复入队；Wiki 触发器按知识库和延迟窗口去重。可通过 `docker logs <app容器> | grep startup-task-reconcile` 查看本次删除和取消数量。Housekeeping 只处理文档状态，不负责制造、重跑或批量删除队列任务。

## Wiki/Graph 模型结论

QA 模型配好以后，不代表所有已有知识库都会自动改用它。

- 对话问答：使用会话或知识库选择的 QA 模型。
- Graph 抽取：使用知识库的 `summary_model_id`，也就是创建/编辑知识库页面里的「LLM 大语言模型」。
- Wiki 生成：优先使用知识库的 `wiki_config.synthesis_model_id`；如果为空，回退到同一个知识库的 `summary_model_id`。
- 内置模型配置文件只负责把模型注册进系统模型表，不会自动修改旧知识库已经保存的模型 ID。
- 新上传或重新解析文档时，才会按当前知识库配置重新跑 Graph/Wiki 生成。

因此，创建知识库时要选好主 QA/LLM 模型；如果启用 Wiki，可以不单独选 Wiki 合成模型，让它默认跟随主 QA/LLM 模型。tc232 上主 QA 模型就是 `lexai-vllm-qwen35-9b-awq-qa`。

## 后台任务并发和聊天保留

详细说明见 [deploy-template/CONCURRENCY.md](deploy-template/CONCURRENCY.md)。这份文档是 ictrek 部署包里的机器资源评估、并发/队列配置入口，说明如何根据显存、模型大小、上下文长度、vLLM 实测满长并发、聊天预留、Asynq 队列权重、Embedding 并发来确定一台机器的部署参数。

Graph、Wiki、文档摘要、表格摘要、自动问题生成都走主 QA/LLM 模型，不要让它们把模型并发吃满。tc97 Thor 的 QA vLLM 按 `18432` 上下文、12 并发、聊天保留 6 部署：

```dotenv
VLLM_MAX_MODEL_LEN=18432
VLLM_MAX_NUM_SEQS=12
WEKNORA_MAIN_QA_MODEL_CONCURRENCY=12
WEKNORA_CHAT_RESERVED_CONCURRENCY=6
WEKNORA_MODEL_MAX_CONCURRENCY=6
WEKNORA_GRAPH_LLM_CONCURRENCY=2
WEKNORA_WIKI_INGEST_MAP_PARALLEL=4
WEKNORA_WIKI_INGEST_REDUCE_PARALLEL=4
```

`WEKNORA_MAIN_QA_MODEL_CONCURRENCY` 对齐 vLLM/Ollama 的请求数上限；`WEKNORA_CHAT_RESERVED_CONCURRENCY` 是给在线聊天保留的下限；在线 QA 仍是最高优先级。新上传文档的主文字解析和批量重新解析走 core worker 池，先完成可检索的文字解析、分块和向量化；VLM/OCR、Graph、摘要和自动问题生成走 enrichment worker 池；Wiki 使用独立 worker 池；`WEKNORA_MODEL_MAX_CONCURRENCY` 统一限制后台主模型调用。tc97 的 qwen vLLM 在 `VLLM_GPU_MEMORY_UTILIZATION=0.65`、`18432` 上下文、`VLLM_MAX_NUM_SEQS=12` 下启动日志显示满长 KV 容量高于请求上限，因此按 12 个请求槽分配，保留 6 个给聊天，后台 Graph/Wiki/VLM/摘要/问题生成最多共用 6 个。tc232 仍按该机器自己的 vLLM 容量配置。新增机器或调整队列权重时，先按 [CONCURRENCY.md](deploy-template/CONCURRENCY.md) 的推荐值和故障现象表处理。

Embedding 模型也要按角色分清：默认 Embedding 应指向吞吐稳定的 OpenAI-compatible Embedding 服务；Ollama bge-m3 可以保留为备用，但不要同时作为默认和后台常驻主路径。Thor 的默认是 `lexai-thor-vllm-bge-m3-embedding`，入口 `http://bge-m3-vllm:22223/v1`，tc97 使用 `BGE_VLLM_MAX_NUM_SEQS=16`、`BATCH_EMBED_SIZE=8`、`CONCURRENCY_POOL_SIZE=8`。

## 配置文件位置

### 内置模型

部署模板中的内置模型文件：

[docs/ictrek/deploy-template/config/builtin_models.yaml](deploy-template/config/builtin_models.yaml)

容器内挂载路径：

```text
/app/config/builtin_models.yaml
```

当前模板里主要有：

| 模型 ID | 用途 |
| --- | --- |
| `lexai-vllm-qwen35-9b-awq-qa` | tc232 vLLM 主 QA/LLM 模型，KnowledgeQA |
| `lexai-vllm-qwen35-9b-awq-vision` | tc232 vLLM Vision 模型 |
| `lexai-ollama-qwen35-4b-qa` | Ollama QA 备用模型 |
| `lexai-ollama-qwen35-4b-vision` | Ollama Vision 模型 |
| `lexai-ollama-bge-m3-embedding` | Ollama bge-m3 Embedding 模型 |

Thor 使用单独的 [builtin_models.thor.yaml](deploy-template/config/builtin_models.thor.yaml)：QA/Graph/Wiki/Question 指向 `lexai-thor-vllm-qwen35-9b-qa`，VLM 指向 `lexai-thor-vllm-qwen35-9b-vlm`，默认 Embedding 指向 `lexai-thor-vllm-bge-m3-embedding`；Ollama bge-m3 只保留为备用。

应用启动时会读取这个 YAML，并按 `id` upsert 到 `models` 表。删除 YAML 中的条目会软删除对应的 YAML 托管模型；手工在页面/API 创建的模型不受影响。

### 默认法律图谱提取模板

前端默认模板：

[frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts)

部署模板中的同源 JSON：

[docs/ictrek/deploy-template/config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json)

容器内挂载路径：

```text
/app/config/legal_graph_preset.json
```

新建知识库和上传确认弹窗里默认出现的法律实体、关系、示例文本来自前端 `legalGraphPreset.ts`。部署目录里的 `legal_graph_preset.json` 是运维可见的同源预设，方便部署包和文档引用；如果要改变前端默认值，需要改 [frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts) 并重新构建 `lexai-ui`。

## 通用部署如何修改后生效

1. 修改模型配置：

   [docs/ictrek/deploy-template/config/builtin_models.yaml](deploy-template/config/builtin_models.yaml)，Thor 则改 [docs/ictrek/deploy-template/config/builtin_models.thor.yaml](deploy-template/config/builtin_models.thor.yaml)

2. 如需修改默认图谱模板，同时修改：

   - [frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts)
   - [docs/ictrek/deploy-template/config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json)

   两份要保持语义一致。只改 JSON 不会改变前端新建知识库时的默认表单。

3. 如果改动需要重建镜像，先按 [开发文档](DEVELOPMENT.md) 构建、推送并更新飞书表格。

4. 部署：

   ```bash
   cd docs/ictrek/deploy-template
   cp .env.example .env
   ./deploy.sh --platform amd
   ```

   `deploy.sh` 会从飞书查当前平台最新的各组件 tag，写入 `.env`，再执行 `docker compose up -d`。

## tc232 部署如何修改后生效

tc232 使用专用 compose：

- [docs/ictrek/deploy-template/docker-compose.tc232.yml](deploy-template/docker-compose.tc232.yml)
- [docs/ictrek/deploy-template/deploy-tc232.sh](deploy-template/deploy-tc232.sh)
- [docs/ictrek/deploy-template/CONCURRENCY.md](deploy-template/CONCURRENCY.md)

部署目录：

```text
/data/jhu/lexai-tc232-deploy
```

更新部署模板到 tc232 时，不要用 `--delete` 同步整个目录，否则会误碰远端 `data/` 数据卷。只同步脚本、compose 和 config：

```bash
rsync -az docs/ictrek/deploy-template/deploy.sh \
  docs/ictrek/deploy-template/deploy-tc232.sh \
  docs/ictrek/deploy-template/docker-compose.tc232.yml \
  docs/ictrek/deploy-template/docker-compose.yml \
  docs/ictrek/deploy-template/CONCURRENCY.md \
  tc232:/data/jhu/lexai-tc232-deploy/

rsync -az docs/ictrek/deploy-template/config/ \
  tc232:/data/jhu/lexai-tc232-deploy/config/
```

然后在 tc232 上部署：

```bash
ssh tc232
cd /data/jhu/lexai-tc232-deploy
./update-and-deploy.sh --platform amd
```

如果 tag 没变但镜像 digest 变了，强制重建 LexAI 容器；若只更新 sandbox 镜像，重建 app 即可：

```bash
docker compose --env-file .env.tc232 -f docker-compose.tc232.yml up -d --force-recreate app frontend docreader
```

## Thor 部署如何修改后生效

Thor 使用专用 compose：

- [docs/ictrek/deploy-template/docker-compose.thor.yml](deploy-template/docker-compose.thor.yml)
- [docs/ictrek/deploy-template/deploy-thor.sh](deploy-template/deploy-thor.sh)
- [docs/ictrek/deploy-template/THOR_DEPLOYMENT.md](deploy-template/THOR_DEPLOYMENT.md)
- [docs/ictrek/deploy-template/CONCURRENCY.md](deploy-template/CONCURRENCY.md)

部署目录：

```text
/data/jhu/dev/workspace/lexai/deploy
```

更新部署模板到 97 时，同步脚本、compose 和 config，不同步数据目录：

```bash
rsync -az docs/ictrek/deploy-template/deploy.sh \
  docs/ictrek/deploy-template/deploy-thor.sh \
  docs/ictrek/deploy-template/docker-compose.thor.yml \
  docs/ictrek/deploy-template/.env.thor.example \
  docs/ictrek/deploy-template/THOR_DEPLOYMENT.md \
  docs/ictrek/deploy-template/CONCURRENCY.md \
  jhu@192.168.1.97:/data/jhu/dev/workspace/lexai/deploy/

rsync -az docs/ictrek/deploy-template/config/ \
  jhu@192.168.1.97:/data/jhu/dev/workspace/lexai/deploy/config/
```

然后在 97 上部署：

```bash
ssh jhu@192.168.1.97
cd /data/jhu/dev/workspace/lexai/deploy
./update-and-deploy.sh --platform thor
```

如果只更新 LexAI 前端或后端镜像，按需强制重建对应服务：

```bash
docker compose --env-file .env.thor -f docker-compose.thor.yml up -d --force-recreate frontend
docker compose --env-file .env.thor -f docker-compose.thor.yml up -d --force-recreate app
```

## 已部署后改配置怎么生效

### 只改 env

例如并发、Neo4j、密钥、端口、模型镜像变量：

通用：

```bash
docker compose --env-file .env -f docker-compose.yml up -d
```

tc232：

```bash
docker compose --env-file .env.tc232 -f docker-compose.tc232.yml up -d
```

Thor：

```bash
docker compose --env-file .env.thor -f docker-compose.thor.yml up -d
```

只想重启 app：

通用：

```bash
docker compose --env-file .env -f docker-compose.yml up -d --force-recreate app
```

tc232：

```bash
docker compose --env-file .env.tc232 -f docker-compose.tc232.yml up -d --force-recreate app
```

Thor：

```bash
docker compose --env-file .env.thor -f docker-compose.thor.yml up -d --force-recreate app
```

### 只改内置模型 YAML

`builtin_models.yaml` 和 Thor 的 `builtin_models.thor.yaml` 都由 app 启动时读取。改完后重启 app：

通用：

```bash
docker compose --env-file .env -f docker-compose.yml up -d --force-recreate app
```

tc232：

```bash
docker compose --env-file .env.tc232 -f docker-compose.tc232.yml up -d --force-recreate app
```

Thor：

```bash
docker compose --env-file .env.thor -f docker-compose.thor.yml up -d --force-recreate app
```

这只会更新系统模型表，不会自动修改旧知识库的 `summary_model_id` 或 `wiki_config.synthesis_model_id`。

### 改了前端默认图谱模板

改的是：

[frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts)

必须重新构建并部署 `lexai-ui`。否则浏览器里的新建知识库默认值不会变化。

### 改了知识库配置

在前端打开：

```text
知识库 -> 设置
```

需要检查：

- 「模型配置」里的 LLM 大语言模型：Graph 抽取使用它。
- 「Wiki 合成模型」：启用 Wiki 后可选；为空就使用 LLM 大语言模型。
- 「知识图谱」里的实体、关系、示例文本：Graph 抽取使用它。
- 索引策略里的 Wiki/Graph 开关：Wiki 必须启用 `wiki_enabled`；Graph 必须启用 `graph_enabled` 且 `extract_config.enabled=true`。

法律 Graph 预设只提供实体、关系和示例文本默认值，不会强制每个知识库生成 Graph。每个知识库可以单独关闭 Wiki/Graph，只保留向量/关键词检索。

保存知识库配置后，只影响后续上传和后续重新解析。已经解析过的文档不会自动重跑。

## 如果 Wiki/Graph 没有自动生成

先确认系统层启用：

- `ENABLE_GRAPH_RAG=true`
- `NEO4J_ENABLE=true`
- app 能访问 `bolt://neo4j:7687`
- 知识库不是 FAQ 类型

再确认知识库层启用：

- `summary_model_id` 不为空
- Wiki：`indexing_strategy.wiki_enabled=true`
- Graph：`indexing_strategy.graph_enabled=true`
- Graph：`extract_config.enabled=true`
- Graph：`extract_config.text/tags/nodes/relations` 都不为空

如果只是部署了 Neo4j 或改了全局环境变量，但知识库自己的开关还是 false，就不会生成。

## 已有文档如何重新生成 Graph

改了实体、关系、示例文本或模型后，已有文档要重新解析。
如果处理流水线里单个 `postprocess.graph.chunk[*]` 失败，例如 vLLM 重启或尚未 ready 导致 `connection refused`，也等模型服务 ready 后重新解析该文档；只保存知识库设置不会自动重试已经失败的文档。

前端操作：

1. 打开知识库。
2. 进入「文档」列表。
3. 选择单个或多个文档。
4. 点击「重新解析」。
5. 在上传/重新解析确认弹窗里确认图谱配置已启用，实体和关系是当前想要的版本。
6. 提交后等待解析完成。
7. 打开「图谱」页查看结果。

API 操作：

```bash
curl -X POST "$BASE_URL/api/v1/knowledge/$KNOWLEDGE_ID/reparse" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

批量重新解析：

文档页工具栏的「重新解析失败文档」只会扫描当前知识库中 `parse_status=failed` 的文档，并按知识库当前默认解析配置批量重新提交；`pending`、`processing`、`finalizing` 不会被这个按钮重跑。

```bash
curl -X POST "$BASE_URL/api/v1/knowledge/batch-reparse" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ids":["knowledge-id-1","knowledge-id-2"]}'
```

如需临时覆盖本次解析配置，可以传 `process_config`。未传时使用知识库当前配置。

## 已有文档如何重新生成 Wiki

Wiki 不是只靠模型表自动生成；它依赖知识库启用 Wiki 索引，并在文档解析后触发 Wiki ingest。

前端操作：

1. 打开知识库设置。
2. 启用 Wiki。
3. 在模型配置里选择 LLM 大语言模型；Wiki 合成模型可以留空，留空会跟随 LLM 大语言模型。
4. 保存知识库。
5. 对已有文档执行「重新解析」。
6. 等待解析和 Wiki ingest 完成。
7. 打开「Wiki」页查看页面。

`/wiki/rebuild-links` 只重建已有 Wiki 页面的链接关系，不会把原始文档重新合成为 Wiki 页面。要重新合成内容，仍然要重新解析文档。

## 法律图谱实体和关系怎么定义

默认模板适合混合法律知识库：法条、司法解释、案例、合同、证据材料都可能出现。完整配置见：

- [docs/ictrek/legal-knowledge-graph-config.md](legal-knowledge-graph-config.md)
- [docs/ictrek/deploy-template/config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json)
- [frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts)

编写原则：

1. 实体类型要稳定，不要把具体对象写成实体类型。
   - 正确：`案件`、`法条`、`法院`、`当事人`
   - 不建议：`张三案`、`民法典第577条`、`北京一中院`

2. 关系要表达法律语义，不要写成泛泛的“相关”。
   - 正确：`裁判依据`、`证明`、`构成要件`、`法律后果`
   - 不建议：`有关`、`包含信息`、`描述`

3. 属性只放检索和判断常用字段。
   - `案件`：案号、案由、审理法院、审级、裁判日期、裁判结果
   - `法条`：条号、款项、原文、适用条件、法律后果、所属法规
   - `证据`：名称、类型、证明对象、来源

4. 示例文本要覆盖你希望模型学会抽取的结构。
   - 法条原文
   - 事实经过
   - 证据
   - 争议焦点
   - 裁判理由和结果

5. 先删后加。
   默认模板已经偏通用。单一业务库不要继续堆实体，先删除无关项：
   - 合同库：保留合同、条款、当事人、权利义务、违约、赔偿、法条。
   - 案例库：保留案件、法院、当事人、争议焦点、证据、裁判观点、法条。
   - 法规库：保留法律法规、法条、司法解释、权利义务、法律责任、处罚措施。

## 修改默认图谱模板的流程

如果只是某一个知识库要改：

1. 在前端知识库设置里改「知识图谱」配置。
2. 保存。
3. 对已有文档重新解析。

如果要改所有新建知识库默认值：

1. 修改 [frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts)。
2. 同步修改 [docs/ictrek/deploy-template/config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json)。
3. 重新构建 `lexai-ui`。
4. 部署新 frontend。
5. 新建知识库检查默认实体和关系。

如果只改部署目录里的 `legal_graph_preset.json`，不会影响前端默认表单；除非后续代码改成由后端读取并下发该 JSON。

## 推荐最小检查清单

部署后：

```bash
docker ps --filter name=lexai
curl -fsS http://127.0.0.1:30081/health
curl -fsSI http://127.0.0.1:30080/
```

知识库创建后：

1. 模型配置里 LLM 大语言模型是预期 QA 模型。
2. Wiki 启用时，Wiki 合成模型为空或等于预期 QA 模型。
3. 知识图谱设置里没有「知识图谱数据库未启用」提示。
4. 实体、关系、示例文本不是空。
5. 上传或重新解析文档后，文档状态完成。
6. Wiki 页有内容，Graph 页有节点和关系。
