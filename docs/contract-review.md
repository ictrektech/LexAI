# 合同审查工作区

合同审查是位于 `/legal/contract-review` 下、按用户持久化保存的法律工作流，不使用聊天会话，也不依赖会过期的聊天附件。

## 生命周期

`draft -> uploading -> ready -> analyzing -> reviewing_clauses -> completed`

解析或模型失败时，任务会进入 `failed` 状态；调用重试接口后，可以继续解析或重新启动结构化审查。归档任务会保留文档和审查结果。删除操作会将任务从产品中移除，并删除已存储的源文件。

## API

- `GET|POST /api/v1/contract-reviews`
- `POST /api/v1/contract-reviews/bulk/archive|restore|delete`（`{ "ids": [...] }`，最多 500 个 ID，返回逐项结果）
- `GET|PATCH|DELETE /api/v1/contract-reviews/:id`
- `POST /api/v1/contract-reviews/:id/document`（multipart 字段 `file`；仅支持 PDF/DOCX）
- `GET /api/v1/contract-reviews/:id/document/preview`（通过 `ServeContent` 支持 HTTP Range）
- `POST /api/v1/contract-reviews/:id/start`
- `POST /api/v1/contract-reviews/:id/retry`
- `GET /api/v1/contract-reviews/:id/events`（SSE 快照与心跳）
- `GET /api/v1/contract-review-playbooks`

列表界面可以选择当前可见的任务，并批量归档、恢复或删除。正在运行的任务不能归档，会被该操作跳过。用户确认明确警告后可以删除运行中的任务；带作用域的更新路径会阻止正在执行的后台 worker 重新创建已软删除的审查任务。

详情工作区使用 SSE，并以 1.5 秒快照作为回退机制。任务运行期间，列表每 2 秒刷新一次，因此状态变化不需要手动刷新。

每个接口都会从经过身份验证的请求中解析租户和用户归属。系统未开放通过 API key 访问的方式。

## 模型与审查方案

结构化执行器会读取租户定制的 `builtin-contract-review` agent，复用其中的模型、温度和 token 配置，同时使用专门的精简提示词输出条款级 JSON。它不会追加面向用户的完整报告提示词，因为其中的标题、引用和报告要求会与结构化条款 schema 冲突。如果条款响应受长度限制或不是有效 JSON，worker 会使用精简的恢复提示词和 8192 token 的补全预算重试一次。如果该 agent 未配置模型，则依次使用租户默认的激活 KnowledgeQA 模型和第一个激活的 KnowledgeQA 模型。没有任何可用模型时启动审查会返回 `MODEL_NOT_CONFIGURED`。

初始版本只有 General Contract Review v1.0 一个审查方案。新增方案时，应加入类型化注册表，并引入专用的 prompt/schema 版本，同时不修改历史任务记录。

## 数据库迁移与 worker

PostgreSQL 迁移 `000079` 和 SQLite 迁移 `000002` 新增审查任务、条款和问题相关数据。文档解析在核心附件队列中运行，条款分析在 enrichment summary 队列中运行；两个处理器也都注册到了 Lite 同步执行器中。
