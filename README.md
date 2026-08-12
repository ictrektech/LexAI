# LexAI

LexAI 是一个面向企业法务、采购和业务团队的法律 AI 工作台。它将法律问答、企业资料和合同审查集中在同一个工作空间中，帮助用户快速查找依据、识别风险并整理处理建议。

用户可以上传合同和业务资料，或连接企业知识库，选择合适的法律 Agent，在保留来源引用的基础上完成问答和审查。

## 核心场景

- **合同签署前审查**：销售合同、采购合同、服务协议、保密协议等文件的风险初筛与条款定位。
- **合同智能档案**：集中管理历史合同和业务文档，提取关键字段、关联主体和重要时间节点。
- **企业法律问答**：基于制度、模板、历史合同和知识库回答法律问题，并保留来源引用。
- **业务协同与风险分流**：让法务把高风险问题交给人工处理，把常见查询和重复审查交给 AI 预处理。

## 当前功能

### 法务助手

- 多轮法律问答与流式回答。
- 连接知识库和法律 Agent，支持引用、附件、上下文追问。
- 可选择 Agent、知识库和模型，适配不同的法律任务。
- 保留原文依据，便于法务复核和继续追问。

### Contract Review

Contract Review 是一个持久化的合同审查工作区，流程为：

```text
新建任务 → 上传 PDF / DOCX → 解析文档 → 配置审查标准与代表方
      → 启动审查 → 按条款逐步输出风险 → 定位原文 → 复核或重新审查
```

主要能力：

- Active / Archived 任务列表，以及重命名、归档、恢复和删除。
- `General Contract Review` 审查标准，支持 Customer / Vendor / Neutral 代表方配置。
- 审查状态实时更新：Uploading、Analyzing、Reviewing clauses、Completed、Failed。
- 结构化输出 Overview、Issues、Clauses、Suggestions。
- 每条问题包含 High / Medium / Low 风险等级、问题说明、原文引用和修改建议。
- PDF 使用 PDF.js 渲染，DOCX 使用原生文档预览；支持页码、缩放、文本选择和条款高亮。
- Review Issue 与文档双向联动：点击问题定位原文，点击文档标记打开对应问题。
- 结果自动保存，支持刷新恢复、失败重试和已完成任务的重新审查。

### Smart Archive / 合同智能档案

智能档案用于沉淀和检索历史合同及相关业务材料：

- 批量导入 PDF、DOC/DOCX、XLS/XLSX、JPG/JPEG、PNG、WEBP 文件，后台解析或 OCR 并提取合同类型、协议编号、客户、金额、签署日、到期日等字段。
- 文档、客户和关键字段统一管理，支持关键词和自然语言检索。
- 每个抽取字段保留原文引用、页码、段落、表格单元格或图片 OCR 等来源证据，便于核对和人工修正。
- 对到期、归还、付款、交付和续约等时间节点生成待确认事项；确认日期、提前天数和负责人后再创建并启用提醒。
- 提供复核队列、失败重试、文档预览、归档、恢复和软删除，保留历史资料的可追溯性。

使用示例：打开 `/legal/smart-archive` 批量导入历史资料，在文档详情中核对字段证据；通过搜索定位相关合同，再从待确认事项中确认日期、提前时间和负责人，创建后启用提醒。

## 一个完整的使用示例

1. 打开 `/legal/contract-review`，点击 **New Review**。
2. 上传一份 PDF 或 DOCX 合同。
3. 选择审查标准和当前代表方，点击 **Start Review**。
4. 在右侧查看风险概览和逐条问题，点击 **View Clause** 回到合同原文。
5. 修改合同后可重新审查，或将任务归档供后续复核。

## 产品与技术特点

- **Agent 驱动**：复用法律审查 Agent 的规则、模型和推理参数；合同工作区使用独立的结构化输出协议，不影响聊天式问答。
- **知识增强**：通过知识库检索企业制度、模板和历史资料，减少脱离业务上下文的回答。
- **持久化工作流**：从文档上传、解析、条款审查到结果定位和复核，所有状态与结果持久化保存。
- **可扩展架构**：工具导航、路由、Playbook、后端 API 和文档 Viewer 均采用模块化设计，可继续扩展 Legal Research、Contract Drafting（合同起草）、Due Diligence、Redline 和 Export。
- **安全与边界**：按用户和租户隔离任务及文件，删除任务时清理持久文件；原文引用用于复核，AI 输出不能替代执业律师的最终判断。
- **部署与扩展**：支持本地或私有化部署，使用 PostgreSQL / SQLite 迁移、异步任务和 SSE 事件流，方便接入现有系统并继续扩展。

## 路由概览

| 页面 | 用途 |
| --- | --- |
| `/legal/ai-assistant` | 新建法律问答 |
| `/legal/ai-assistant/chat/:chatid` | 法律问答会话 |
| `/legal/contract-review` | 合同审查任务列表 |
| `/legal/contract-review/:reviewId` | 合同审查工作区 |
| `/legal/smart-archive` | 合同智能档案 |
| `/platform/knowledge-bases` | 知识库管理 |
| `/platform/agents` | Agent 管理 |
| `/platform/settings` | 系统设置 |

## 快速开始

请参阅[本地开发与启动指南](./docs/ictrek/local-dev/README.md)，其中包含开发环境初始化、服务启动、模型配置、前端启动和环境检查命令。

## 开发文档

- [前端工作台与扩展方式](./frontend/README.md)
- [Contract Review API、状态机与 Playbook](./docs/contract-review.md)
- [智能档案 API、字段证据与提醒](./docs/smart-archive.md)

## 许可证与第三方声明

本项目的基础代码源自 [Tencent/WeKnora](https://github.com/Tencent/WeKnora)。请参阅仓库中的 [LICENSE](./LICENSE) 文件，保留上游版权、许可证和归属声明。第三方组件可能适用各自的许可证，具体以 LICENSE 中的说明为准。
