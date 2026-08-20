# 合同起草与 Office Engine

LexAI 的“编辑合同”功能通过独立项目 `/home/czy/p/office-engine` 提供
`office.engine.v1` gRPC 能力。LexAI 保存源 DOCX 的存储引用和 SHA256、用户指令、
模型与 EditPlan、任务状态、能力快照、操作账本及产物存储引用；发送给 Worker 的
只有文档字节、摘要和结构化请求，不包含宿主机路径、数据库信息或对象存储凭据。

## 本阶段能力

- Adeu：DOCX 读取、搜索、结构化编辑、评论、原生 Track Changes，以及 redline/clean 产物。
- OfficeCLI：DOCX 读取、编辑、OpenXML 验证、issues 和 HTML/PNG 预览；只输出 clean，不支持评论或 Track Changes。
- Hybrid：Adeu 是唯一修改器；OfficeCLI 只接收 Adeu 的 clean 结果做验证和渲染，避免重复修改。
- XLSX/PPTX 仅保留协议枚举，本阶段不允许前端上传。

## 当前实际生成流程

“编辑合同”页面提交的是“DOCX 文件 + 自然语言修改要求 + 执行模式”。Hybrid
不是一个单独的 LLM，而是文档执行编排模式；LLM 负责把自然语言要求转换成
结构化 `EditPlan`，Adeu 负责实际修改，OfficeCLI 负责验证和渲染。

当前页面没有传入 `model_id`，因此 LexAI 会选择默认的 `KnowledgeQA` 模型。
仓库中的 `tc232` 内置配置将其声明为：

```text
模型：qwen3.5-9b-awq
显示名：qwen3.5-9b-awq (vLLM QA)
调用方式：vLLM OpenAI-compatible API
```

`qwen3.5:4b (Ollama QA)` 在该配置中是备用模型。这里描述的是仓库配置而非实时
运行状态；数据库中的模型配置、显式传入的 `model_id` 或其他部署配置都可能改变
实际使用的模型，应以任务详情中的 `model_id` 为准。

## 三种模式的执行过程

三种模式的公共前置流程相同：浏览器上传 DOCX 和修改要求后，LexAI 校验文件
类型和大小、保存只读源文件并计算 `source_sha256`，然后创建 Asynq 异步任务。
任务执行过程中，LexAI 始终先读取源文件并校验 SHA256，再生成或加载结构化
`EditPlan`，校验操作类型、目标字段和 `expected_matches == 1` 等结构约束，最后
才允许 Worker 执行。目标文本在文档中的实际匹配次数由 Worker 在 Apply 阶段
校验；如果请求已经携带合法的 `edit_plan`，则跳过 LLM 规划阶段。

### Adeu 单独模式

```text
浏览器上传 DOCX 和修改要求
  → LexAI 校验文件、保存源文件并计算 source_sha256
  → 创建 Asynq 异步任务
  → Adeu Inspect 提取文档文本和结构
  → LexAI 调用 KnowledgeQA 模型生成 EditPlan JSON
  → LexAI 校验计划结构、操作类型和源文件 SHA256
  → Adeu 在 Apply 阶段校验目标文本的实际唯一匹配
  → Adeu Apply 执行唯一一次原子修改
  → Adeu 生成原生 Track Changes、redline 和 clean
  → Adeu Validate 校验 clean DOCX
  → LexAI 保存 redline、clean、validation 和诊断产物
  → 任务标记为 completed
```

Adeu 是唯一的修改器，支持 `replace_text`、`insert_before`、`insert_after`、
`delete_text` 和 `add_comment`，并支持原生 Word 修订。Adeu 不提供页面渲染，
因此该模式不会生成页面预览，前端应显示“无预览”提示。

### OfficeCLI 单独模式

```text
浏览器上传 DOCX 和修改要求
  → LexAI 校验文件、保存源文件并计算 source_sha256
  → 创建 Asynq 异步任务
  → OfficeCLI Inspect 提取文档文本和结构
  → LexAI 调用 KnowledgeQA 模型生成 EditPlan JSON
  → LexAI 校验计划结构、操作类型和源文件 SHA256
  → OfficeCLI 在 Apply 阶段校验目标文本的实际唯一匹配
  → OfficeCLI 在工作副本上逐条执行修改，全部成功后才返回 clean
  → OfficeCLI Validate 执行 OpenXML 校验并生成 issues
  → OfficeCLI Render 生成 HTML 页面预览（也支持 PNG 渲染能力）
  → LexAI 保存 clean、validation、render 和诊断产物
  → 任务标记为 completed
```

OfficeCLI 是唯一的修改器，只输出 clean DOCX，不支持 Track Changes 和评论。
因此 OfficeCLI 模式下前端不应提供 redline 或评论选项；如果 `EditPlan` 包含
`add_comment`，LexAI 会在执行前拒绝任务，不会调用 Apply。

### Hybrid 模式

```text
浏览器上传 DOCX 和修改要求
  → LexAI 校验文件、保存源文件并计算 source_sha256
  → 创建 Asynq 异步任务
  → Adeu Inspect 提取文档文本（失败时回退到 OfficeCLI Inspect）
  → LexAI 调用 KnowledgeQA 模型生成 EditPlan JSON
  → LexAI 校验计划结构、操作类型和源文件 SHA256
  → Adeu 在 Apply 阶段校验目标文本的实际唯一匹配
  → Adeu Apply 执行唯一一次修改，生成 redline 和 clean
  → OfficeCLI 只接收 clean DOCX 并执行 Validate
  → OfficeCLI 只接收 clean DOCX 并执行 Render，生成 HTML 页面预览
  → LexAI 保存 redline、clean、validation、render 和诊断产物
  → 任务标记为 completed
```

Hybrid 中 Adeu 是唯一的修改器，OfficeCLI 没有 Apply 权限，不会再次修改文档，
因此不会发生重复修改。该模式同时保留 Adeu 的 Track Changes、评论和 redline
能力，以及 OfficeCLI 的 OpenXML 校验和页面预览能力，是页面默认模式。

### 三种模式差异

| 模式 | Inspect / 规划输入 | 实际修改 | Validate | Render | 主要产物 |
| --- | --- | --- | --- | --- | --- |
| Adeu | Adeu | Adeu | Adeu | 不支持 | redline、clean、validation |
| OfficeCLI | OfficeCLI | OfficeCLI | OfficeCLI | OfficeCLI | clean、validation、render |
| Hybrid | Adeu（失败时回退 OfficeCLI） | 仅 Adeu | OfficeCLI | OfficeCLI | redline、clean、validation、render |

### KnowledgeQA 调用与 EditPlan 差异

当前三种模式不会调用不同的 KnowledgeQA 模型。只要任务没有预先提交合法的
`edit_plan`，三种模式都会使用相同的模型选择逻辑、Prompt、JSON Schema 和调用
参数：

```text
model_id：请求显式指定；未指定时选择默认 KnowledgeQA 模型
temperature：0.1
max_completion_tokens：4096
thinking：false
返回格式：EditPlan JSON
```

请求预先提交的合法 `edit_plan` 会在创建任务时持久化。由 KnowledgeQA 生成且通过
校验的计划也会在 Apply 前立即持久化；如果任务在 Apply、Validate 或 Render 阶段
失败并由 Asynq 重试，后续尝试直接复用该计划，不会重复调用模型。初次模型调用
本身失败时任务直接失败；只有模型返回的 JSON 或嵌套结构不合法时，才会追加最多
一次修复调用。

实际差异来自规划阶段使用的 Inspect 文本：

| 模式 | 传给 KnowledgeQA 的 Inspect 来源 | 影响 |
| --- | --- | --- |
| Adeu | Adeu 文本提取 | 更接近纯文本 |
| OfficeCLI | OfficeCLI `view text` | 可能包含节点路径、不同的空格和换行规范化 |
| Hybrid | 优先 Adeu，失败时回退 OfficeCLI | 通常与 Adeu 的规划输入一致 |

当前 `capabilitiesForMode` 在 EditPlan 生成之后才执行，因此模型生成时并不知道
OfficeCLI 不支持评论和 Track Changes，也没有得到模式对应的产物能力约束。虽然
`EditPlan` 的操作结构相同，但后置校验只对 OfficeCLI 特别禁止了 `add_comment`。
此外，当前 Prompt 对所有模式都声明 `output_modes: [redline, clean]`，而 OfficeCLI
实际上只能产生 clean，这是后续需要修正的协议提示问题。

OfficeCLI 会严格执行 EditPlan 的 `target.quote` 和 `payload.text`，不会理解自然
语言中的“填写字段”语义。例如，以下计划会替换字段标题：

```json
{
  "target": {"quote": "甲方（出借方）：", "expected_matches": 1},
  "payload": {"text": "上芯途异构（出借方）："}
}
```

如果需求是填写空白，计划应保留字段标题，目标和结果应类似：

```text
甲方（出借方）：                                     → 甲方（出借方）：芯途异构
```

因此，同一自然语言指令在不同模式下可能生成不同的 EditPlan，最终差异不一定是
Worker 修改能力造成的，而可能来自 Inspect 文本和规划锚点不同。后续应把模式和
Worker 能力显式注入 Planner，并增加字段标题保护、禁止 Markdown 标记、修改后
结果包含校验等规则。

三种模式都遵循相同的安全边界：原始 DOCX 不被覆盖，任务创建时记录的
`source_sha256` 必须保持不变；编辑采用原子执行和唯一匹配校验；只有验证成功
且所有产物保存成功后，任务才会发布为 `completed`。

LLM 规划阶段的输入包括用户修改要求、文档提取文本和源文件 SHA256。文档文本
会被标记为不可信数据，模型只能引用文档中能够精确匹配的原文作为编辑目标，
不能直接操作文件系统、OOXML 或 Worker。

LLM 输出的 `EditPlan` 包含 `replace_text`、`insert_before`、`insert_after`、
`delete_text` 和 `add_comment` 等操作。单次执行正常情况下调用一次 LLM；如果
返回的 JSON 或嵌套结构不合法，最多追加一次修复调用。目标文本不唯一时 Worker
会拒绝 Apply，不会发布修改产物。若请求已经携带合法的 `edit_plan`，则跳过 LLM
规划阶段。

Hybrid 中只有 Adeu 有修改权限，OfficeCLI 不会再次 Apply，因此不会重复修改。
最终通常保存 redline、clean、validation 报告和 render 预览；原始 DOCX 保持
只读，源文件 SHA256 不变。

任务详情接口可以查看本次实际使用的模型和计划：

```text
GET /api/v1/document-edits/{job_id}
```

重点字段为 `model_id`、`plan`、`capabilities`、`artifacts` 和 `status`。

## 执行诊断与模式对比

任务详情右上角的“执行诊断”进入：

```text
/legal/drafting/{job_id}/debug
```

该页面仅允许任务创建者在当前租户内访问，展示 Inspect、Plan、Apply、Validate、
Render、Publish 的尝试次数、引擎、耗时、错误和摘要；Planner messages、模型原始
响应与 Inspect/clean 文本快照只在展开时按需加载。调试 API 不返回对象存储引用、
模型凭据、HTTP Header 或隐藏推理内容。旧任务没有阶段轨迹时会明确显示“创建时未
记录”，但仍展示已有 Plan、Capabilities、Operations 和 Artifacts。

诊断接口：

```text
GET  /api/v1/document-edits/{job_id}/debug
GET  /api/v1/document-edits/{job_id}/debug/stages/{stage_id}/blobs/{kind}
POST /api/v1/document-edits/{job_id}/comparisons
GET  /api/v1/document-edits/{job_id}/comparison
```

终态任务可以创建两种对比：`replan` 固定源文件、修改要求和模型，让每种模式使用
各自 Inspect 重新规划；`locked_plan` 固定原 Plan，只比较 Worker。若 locked Plan
含有 OfficeCLI 不支持的 `add_comment`，服务会在创建任何对比任务前整体拒绝。
Hybrid 的阶段轨迹应始终显示 Adeu 执行 Apply，OfficeCLI 只执行 Validate/Render。

验证新诊断能力时，先完成一次任务，再检查：

```bash
JOB_ID='<任务 ID>'
curl -sS \
  -H "Authorization: Bearer ${LEXAI_TOKEN}" \
  -H "X-Tenant-ID: ${LEXAI_TENANT_ID}" \
  "http://127.0.0.1:8080/api/v1/document-edits/${JOB_ID}/debug"
```

响应应包含 `stages`，成功任务通常有 Inspect、Plan、Apply、Validate、Render、Publish；
`blobs` 只包含 kind、SHA256、大小等元数据，不应包含 `storage_ref`。在页面点击文本
快照后才应发起 Blob 请求。对比验证可在页面选择 `replan` 或 `locked_plan`，也可调用：

```bash
curl -sS -X POST \
  -H "Authorization: Bearer ${LEXAI_TOKEN}" \
  -H "X-Tenant-ID: ${LEXAI_TENANT_ID}" \
  -H 'Content-Type: application/json' \
  -d '{"modes":["adeu","officecli"],"strategy":"replan"}' \
  "http://127.0.0.1:8080/api/v1/document-edits/${JOB_ID}/comparisons"
```

使用另一个用户或租户访问上述 job/debug/blob/comparison 接口应返回未找到或无权访问，
且响应中不应出现 `source_ref`、`storage_ref`、API Key 或其他用户的任务数据。

## 直接进程启动

本期不构建镜像，也不修改 LexAI Compose/Helm。Adeu Worker 要求 Python 3.12+
和 `uv`；OfficeCLI Worker 要求 Go 工具链和可用的 OfficeCLI 可执行文件。启动前
先检查：

```bash
python3 --version
uv --version
go version
command -v officecli
```

`command -v officecli` 应输出有效路径。OfficeCLI Worker 默认也能通过启动进程的
`PATH` 查找 `officecli`；为使实际使用的二进制路径更明确，下面的直接启动示例会
先动态解析绝对路径，再通过 `OFFICECLI_BINARY` 传给 Worker。进程管理器使用不同
`PATH` 时，也可以直接配置固定的绝对路径。

两个 Worker 都是常驻进程，必须分别在两个终端启动。

终端一：

```bash
cd /home/czy/p/office-engine
OFFICE_ENGINE_ADDR=127.0.0.1:50052 ./scripts/run-adeu-worker.sh
```

终端二：

```bash
cd /home/czy/p/office-engine
officecli_path="$(command -v officecli)" || exit 1
OFFICE_ENGINE_ADDR=127.0.0.1:50053 \
OFFICECLI_BINARY="$officecli_path" \
./scripts/run-officecli-worker.sh
```

这样会从当前 `PATH` 中解析 OfficeCLI 的绝对路径，再通过 `OFFICECLI_BINARY` 传给
Worker；未找到 OfficeCLI 时命令直接退出。如果 OfficeCLI 不在启动进程的 `PATH`
中，再显式指定：

```bash
cd /home/czy/p/office-engine
OFFICE_ENGINE_ADDR=127.0.0.1:50053 \
OFFICECLI_BINARY=/opt/officecli/bin/officecli \
./scripts/run-officecli-worker.sh
```

任务状态通过 Asynq、SSE 和数据库账本持久化；前端可取消排队或执行中的任务。取消采用协作式安全边界，已取消任务不会发布 Worker 的临时产物。

在启动 LexAI 的同一个 Shell 或进程管理配置中设置：

```bash
export OFFICE_EDITING_ENABLED=true
export OFFICE_ENGINE_MODE=hybrid
export OFFICE_ENGINE_ADEU_ADDR=127.0.0.1:50052
export OFFICE_ENGINE_OFFICECLI_ADDR=127.0.0.1:50053
```

这些配置只在 LexAI 进程启动时读取，修改后必须重启 LexAI。默认
`config/config.yaml` 中编辑能力关闭、两个地址为空，所以只启动前端或只刷新页面
不会使 Worker 变为可用。如果 LexAI 与 Worker 不在同一台机器，Worker 应监听
可访问地址，LexAI 中应配置对应的主机名或 IP，而不是 `127.0.0.1`。

Worker 缺失时 LexAI 仍可启动，只返回不可用能力并禁用对应编辑任务。

在 `a6000` 需要退出终端后仍保持直接进程运行时，可以交给用户级 systemd；这仍然
不是容器部署：

```bash
systemctl --user status office-engine-adeu-dev.service office-engine-officecli-dev.service
systemctl --user restart office-engine-adeu-dev.service office-engine-officecli-dev.service
journalctl --user -u office-engine-adeu-dev.service -u office-engine-officecli-dev.service -f
```

当前开发机使用上述两个 transient service 名称，分别监听 50052 和 50053。

## 验证服务是否正常

建议按照“端口 → Worker gRPC → LexAI 探测 → DOCX 端到端”的顺序验证。端口可达
只表示进程已监听，不能替代 gRPC Health 和实际文档任务验证。

### 1. 检查监听端口

在 Worker 所在机器执行：

```bash
ss -ltnp | rg ':(50052|50053)\b'
```

应同时看到 `127.0.0.1:50052` 和 `127.0.0.1:50053` 处于 `LISTEN` 状态。缺少某个
端口时先检查对应终端的启动日志。

### 2. 调用 Worker Health 和 GetCapabilities

安装了 `grpcurl` 时，可直接使用仓库中的 Proto 调用服务；Worker 当前未启用 gRPC
reflection，因此必须传入 `-import-path` 和 `-proto`。

Adeu：

```bash
grpcurl -plaintext \
  -import-path /home/czy/p/office-engine/api/proto \
  -proto office/engine/v1/document_engine.proto \
  -d '{}' \
  127.0.0.1:50052 \
  office.engine.v1.OfficeEngineService/Health

grpcurl -plaintext \
  -import-path /home/czy/p/office-engine/api/proto \
  -proto office/engine/v1/document_engine.proto \
  -d '{}' \
  127.0.0.1:50052 \
  office.engine.v1.OfficeEngineService/GetCapabilities
```

OfficeCLI：

```bash
grpcurl -plaintext \
  -import-path /home/czy/p/office-engine/api/proto \
  -proto office/engine/v1/document_engine.proto \
  -d '{}' \
  127.0.0.1:50053 \
  office.engine.v1.OfficeEngineService/Health

grpcurl -plaintext \
  -import-path /home/czy/p/office-engine/api/proto \
  -proto office/engine/v1/document_engine.proto \
  -d '{}' \
  127.0.0.1:50053 \
  office.engine.v1.OfficeEngineService/GetCapabilities
```

两个 Health 响应都应包含 `status: "ok"` 和协议版本 `office.engine.v1`。Adeu 的
能力应包含 Track Changes、评论和验证，不包含渲染；OfficeCLI 应包含验证和渲染，
但 Track Changes 和评论为 `false`。OfficeCLI Health 会实际执行配置的二进制
`--help`，因此二进制不存在或不可执行时不会返回 `ok`。

### 3. 验证 LexAI 能力探测

确认 LexAI 已经带 `OFFICE_*` 配置重启后，在“编辑合同”页面点击“刷新”，Adeu 和
OfficeCLI 都应显示 `ok`。也可以携带当前登录令牌直接调用：

```bash
LEXAI_TOKEN='<登录令牌>'
LEXAI_TENANT_ID='<租户 ID>'
curl -sS \
  -H "Authorization: Bearer ${LEXAI_TOKEN}" \
  -H "X-Tenant-ID: ${LEXAI_TENANT_ID}" \
  http://127.0.0.1:8080/api/v1/document-edits/capabilities
```

响应中的 `data.health.adeu.status` 和 `data.health.officecli.status` 应均为 `ok`。
如果 Worker 的 Health 正常但这里仍为 `unavailable`，优先检查 LexAI 是否已经重启、
Worker 地址是否与部署拓扑一致，以及 LexAI 到 Worker 端口是否可达。

### 4. 验证三种模式的 DOCX 闭环

准备一个包含“付款期限为十日”和“争议提交上海仲裁委员会”两段唯一文本的简单
DOCX，并记录原文件摘要：

```bash
sha256sum contract-test.docx
```

在 `/legal/drafting` 分别提交以下任务：

| 模式 | 示例要求 | 预期产物 |
| --- | --- | --- |
| Adeu | 将“付款期限为十日”改为“付款期限为三十日”，并给仲裁条款添加说明批注 | redline、clean、validation；无 render |
| OfficeCLI | 将“付款期限为十日”改为“付款期限为三十日” | clean、validation、render；无 redline |
| Hybrid | 将“付款期限为十日”改为“付款期限为三十日”，并给仲裁条款添加说明批注 | redline、clean、validation、render |

任务应依次进入 `queued → running → completed`，详情接口中的 `model_id`、`plan`、
`capabilities` 和 `artifacts` 应有值。下载产物后确认：redline 中存在 Word 原生修订，
clean 中修改已经接受，HTML 预览可以打开；再次执行 `sha256sum contract-test.docx`
应与任务前完全一致。

再使用包含两处相同目标文本的 DOCX 提交修改，任务应以目标不唯一失败，且不发布
半成品产物。该用例用于确认唯一匹配和原子发布边界确实生效。

### 常见故障

| 现象 | 优先检查 |
| --- | --- |
| 两个 Worker 都是 `unavailable` | `OFFICE_EDITING_ENABLED=true` 是否在 LexAI 启动前设置，以及 LexAI 是否重启 |
| 只有 Adeu 不可用 | Python 是否为 3.12+、`uv` 依赖安装是否成功、50052 启动日志 |
| 只有 OfficeCLI 不可用 | `command -v officecli` 和 `officecli --help` 是否成功；若进程 `PATH` 中找不到，再设置 `OFFICECLI_BINARY` |
| Worker Health 正常但页面不可用 | LexAI 配置地址、跨主机监听地址、防火墙及 50052/50053 连通性 |
| 任务停留在 `queued` | Asynq Worker 和 Redis 是否正常、文档编辑任务队列是否被消费 |
| 任务在规划阶段失败 | 默认 KnowledgeQA 模型是否存在且可访问，查看任务 `error_code` 和 `error_message` |
| Validate 或 Render 失败 | OfficeCLI 版本、二进制日志、DOCX OpenXML 问题及外部关系警告 |

## 数据库迁移

合同起草任务基础数据使用 `document_edit_jobs`、`document_edit_artifacts` 和
`document_edit_operations`；`000096`（SQLite 为 `000009`）增加
`document_edit_stage_runs`、`document_edit_debug_blobs`、对比组字段和操作执行诊断。
已执行过旧版 `000094` 但缺少操作账本表的
部署，会通过后续 `000095_document_edit_operations_repair` 迁移补齐表和索引；
不要修改已执行的历史迁移文件。
