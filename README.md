<p align="center">
  <picture>
    <img alt="LexAI" src="./frontend/src/assets/img/LexAI_logo_exact.svg" width="312" style="max-width: 100%;">
  </picture>
</p>

<h3 align="center">
LexAI：企业法律 AI 工作台
</h3>

<p align="center">
  <a href="docs/ictrek/local-dev/README.md">Documentation</a> &bull;
  <a href="docs/ictrek/DEVELOPMENT.md">Development Guide</a> &bull;
  <a href="docs/contract-review.md">Contract Review</a> &bull;
  <a href="docs/smart-archive.md">Smart Archive</a> &bull;
  <a href="frontend/README.md">Frontend Guide</a>
</p>

LexAI 是面向企业法务、采购和业务团队的法律 AI 工作台，集成法律问答、合同审查和合同智能档案。系统支持私有化部署，并保留可追溯的原文依据。

## 核心功能

### 法务助手

- 基于企业知识库进行多轮法律问答。
- 支持 Agent、附件、来源引用和上下文追问。
- 可按任务选择知识库和模型。

### 合同审查

- 上传 PDF 或 DOCX，配置审查标准和代表方后开始审查。
- 按 Overview、Issues、Clauses、Suggestions 输出结构化结果。
- 风险问题包含等级、说明、原文引用和修改建议。
- 支持 PDF/DOCX 预览、条款定位、高亮及双向联动。
- 任务自动保存并实时更新，支持重新审查、批量归档、恢复和删除。

详细说明：[Contract Review](./docs/contract-review.md)

### 合同智能档案

- 批量导入 PDF、Word、Excel、JPG、PNG 和 WEBP，支持图片 OCR。
- 抽取文档类型、协议编号、关联主体、金额和关键日期等字段。
- 支持关键词、自然语言和字段检索，所有 AI 字段均保留原文证据。
- 自动识别到期、归还、付款、交付和续约事项，但由用户确认后创建并启用提醒。
- 支持复核、重新识别、批量归档/恢复、归档文档移入回收站，以及提醒候选和提醒的批量管理。
- 文档删除仅限管理员/所有者，移入回收站后会取消关联提醒并按保留期限清理。

详细说明：[Smart Archive](./docs/smart-archive.md)

## 主要路由

| 路由 | 用途 |
| --- | --- |
| `/legal/ai-assistant` | 法务助手 |
| `/legal/ai-assistant/chat/:chatid` | 法律问答会话 |
| `/legal/contract-review` | 合同审查任务 |
| `/legal/contract-review/:reviewId` | 合同审查工作区 |
| `/legal/smart-archive` | 合同智能档案 |
| `/platform/knowledge-bases` | 知识库管理 |
| `/platform/agents` | Agent 管理 |
| `/platform/settings` | 系统设置 |

## 开发文档

- [本地开发与启动](./docs/ictrek/local-dev/README.md)
- [前端工作台与扩展](./frontend/README.md)
- [合同审查 API 与状态机](./docs/contract-review.md)
- [智能档案 API、证据与提醒](./docs/smart-archive.md)

## 许可证

本项目基于 [Tencent/WeKnora](https://github.com/Tencent/WeKnora) 开发，许可证及第三方声明见 [LICENSE](./LICENSE)。
