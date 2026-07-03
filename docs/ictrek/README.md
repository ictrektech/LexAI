# LexAI ictrek 部署 README

本文说明 ictrek 部署包需要哪些文件、如何修改配置、如何自动检测或手动填写镜像版本，以及 LexAI 法律部署中 QA 模型、Wiki 生成、知识图谱抽取之间的关系。

开发、合并上游、构建镜像和 push 流程见 [开发文档](DEVELOPMENT.md)。

## 部署包范围

部署目标机不需要整个 repo。按场景只同步必要文件：

| 场景 | 需要的文件 | 不需要的文件 | 主要改哪里 |
| --- | --- | --- | --- |
| tc232，已有 `qwen35-9b-awq-vllm` | [deploy-tc232.sh](deploy-template/deploy-tc232.sh)、[docker-compose.tc232.yml](deploy-template/docker-compose.tc232.yml)、[.env.tc232.example](deploy-template/.env.tc232.example)、[config/](deploy-template/config/)；本地同步可用 [sync-tc232.sh](deploy-template/sync-tc232.sh) | [docker-compose.yml](deploy-template/docker-compose.yml) 里的 vllm 服务不会用；`.env.example` 不是运行入口 | `.env.tc232` 的端口、密钥、数据目录、并发；如已有 vllm 容器名不同，改 [config/builtin_models.yaml](deploy-template/config/builtin_models.yaml) 里的 `base_url` |
| 全新主机，没有可用 vllm | [deploy.sh](deploy-template/deploy.sh)、[docker-compose.yml](deploy-template/docker-compose.yml)、[.env.example](deploy-template/.env.example)、[config/](deploy-template/config/) | `deploy-tc232.sh`、`docker-compose.tc232.yml`、`.env.tc232.example` | `.env` 的 `VLLM_HOST_PORT`、`VLLM_HF_MODELS_DIR`、模型目录、端口、密钥、并发 |
| 主机已有可用 vllm，且同一个模型同时支持文本和 VLM | 以 [docker-compose.tc232.yml](deploy-template/docker-compose.tc232.yml) 为模板另存一份机器专用 compose，配套一份 env，再带上 [config/](deploy-template/config/) | 通用 compose 里的 `qwen35-9b-awq-vllm` 服务不需要启动 | 把 [config/builtin_models.yaml](deploy-template/config/builtin_models.yaml) 中 QA 和 Vision 模型的 `base_url` 都指向已有 vllm 容器名；模型 ID 可以共用同一个 served model |
| 只手动固定镜像版本 | 对应场景的 compose、env、[config/](deploy-template/config/) | [deploy.sh](deploy-template/deploy.sh) 可不用 | 直接在 env 里填写 `LEXAI_APP_IMAGE`、`LEXAI_UI_IMAGE`、`LEXAI_DOCREADER_IMAGE` 等镜像变量 |

[config/builtin_models.yaml](deploy-template/config/builtin_models.yaml) 和 [config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json) 两个配置文件建议所有部署场景都带上。前者注册默认模型，后者保留法律图谱实体/关系模板。

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
WEKNORA_GRAPH_LLM_CONCURRENCY=2
ENABLE_GRAPH_RAG=true
NEO4J_ENABLE=true
NEO4J_URI=bolt://neo4j:7687
```

tc232 部署，即复用 `lexai` 网络里已有的 `qwen35-9b-awq-vllm`：

1. 本地执行 [sync-tc232.sh](deploy-template/sync-tc232.sh)，会同步到 `tc232:/data/jhu/lexai-tc232-deploy`。
2. 在 tc232 执行 `cp .env.tc232.example .env.tc232`，如果已有 `.env.tc232` 就只补缺失项。
3. tc232 不需要配置 `VLLM_*` 镜像服务，compose 会通过 `http://qwen35-9b-awq-vllm:8000/v1` 访问 `lexai` 网络里已有的 vllm。

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
./deploy.sh --platform thor
```

如果需要指定表格 sheet：

```bash
./deploy.sh --platform amd --sheet AMD_with_cuda
```

只检查将会使用哪些镜像、不实际部署：

```bash
./deploy.sh --platform amd --dry-run
```

tc232 专用部署：

```bash
cd /data/jhu/lexai-tc232-deploy
./deploy-tc232.sh
```

[deploy.sh](deploy-template/deploy.sh) 会分别查找这些组件的最新镜像，允许 LexAI 三个组件和 model_hub/ollama 使用不同版本：

```text
LEXAI_APP_IMAGE
LEXAI_UI_IMAGE
LEXAI_DOCREADER_IMAGE
MODEL_HUB_BACKEND_IMAGE
MODEL_HUB_FRONTEND_IMAGE
OLLAMA_SERVER_IMAGE
```

脚本会把查到的镜像写回 `.env` 或 `.env.tc232`，然后执行对应 compose 文件的 `up -d`。

## 手动填写镜像部署

如果目标机没有飞书凭据，或者要固定某一批镜像版本，不要运行 `deploy.sh`。直接编辑 `.env` 或 `.env.tc232`，手动填入镜像：

```bash
LEXAI_APP_IMAGE=registry.example.com/lexai:xxx
LEXAI_UI_IMAGE=registry.example.com/lexai-ui:xxx
LEXAI_DOCREADER_IMAGE=registry.example.com/lexai-docreader:xxx
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

注意：再次运行 `deploy.sh` 或 `deploy-tc232.sh` 会重新从飞书表格检测镜像并覆盖 env 中的镜像变量。需要完全手动固定版本时，只运行 `docker compose ... up -d`。

## 已部署环境更新

已部署环境改配置后，一般只需要同步部署模板文件并重新 `up -d`：

- 改 `.env` / `.env.tc232`：重新执行对应 compose `up -d`。
- 改 [config/builtin_models.yaml](deploy-template/config/builtin_models.yaml)：同步文件后重启 `lexai-app`。
- 改 [config/legal_graph_preset.json](deploy-template/config/legal_graph_preset.json)：同步文件后重启 `lexai-app`，新建或重新保存知识库图谱配置后生效。
- 改 [docker-compose.yml](deploy-template/docker-compose.yml) 或 [docker-compose.tc232.yml](deploy-template/docker-compose.tc232.yml)：重新执行对应 compose `up -d`。
- 改 [frontend/src/config/legalGraphPreset.ts](../../frontend/src/config/legalGraphPreset.ts)：这是前端默认值，必须重新构建并部署 `lexai-ui` 镜像。

部署数据不应该跟 repo 一起同步。Postgres、Redis、Neo4j、Qdrant、上传文件、ollama 模型、HF 模型都通过 compose volume 或宿主机目录保存。换新镜像只要复用同一套 `.env`、compose 和数据目录，数据会恢复。

## Wiki/Graph 模型结论

QA 模型配好以后，不代表所有已有知识库都会自动改用它。

- 对话问答：使用会话或知识库选择的 QA 模型。
- Graph 抽取：使用知识库的 `summary_model_id`，也就是创建/编辑知识库页面里的「LLM 大语言模型」。
- Wiki 生成：优先使用知识库的 `wiki_config.synthesis_model_id`；如果为空，回退到同一个知识库的 `summary_model_id`。
- 内置模型配置文件只负责把模型注册进系统模型表，不会自动修改旧知识库已经保存的模型 ID。
- 新上传或重新解析文档时，才会按当前知识库配置重新跑 Graph/Wiki 生成。

因此，创建知识库时要选好主 QA/LLM 模型；如果启用 Wiki，可以不单独选 Wiki 合成模型，让它默认跟随主 QA/LLM 模型。tc232 上主 QA 模型就是 `lexai-vllm-qwen35-9b-awq-qa`。

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

   [docs/ictrek/deploy-template/config/builtin_models.yaml](deploy-template/config/builtin_models.yaml)

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
  tc232:/data/jhu/lexai-tc232-deploy/

rsync -az docs/ictrek/deploy-template/config/ \
  tc232:/data/jhu/lexai-tc232-deploy/config/
```

然后在 tc232 上部署：

```bash
ssh tc232
cd /data/jhu/lexai-tc232-deploy
./deploy-tc232.sh
```

如果 tag 没变但镜像 digest 变了，强制重建 LexAI 三个容器：

```bash
docker compose --env-file .env.tc232 -f docker-compose.tc232.yml up -d --force-recreate app frontend docreader
```

## 已部署后改配置怎么生效

### 只改 `.env.tc232` 或 `.env`

例如并发、Neo4j、密钥、端口、模型镜像变量：

```bash
docker compose --env-file .env.tc232 -f docker-compose.tc232.yml up -d
```

只想重启 app：

```bash
docker compose --env-file .env.tc232 -f docker-compose.tc232.yml up -d --force-recreate app
```

### 只改 `builtin_models.yaml`

`builtin_models.yaml` 由 app 启动时读取。改完后重启 app：

```bash
docker compose --env-file .env.tc232 -f docker-compose.tc232.yml up -d --force-recreate app
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
