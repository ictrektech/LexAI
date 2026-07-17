# 法律知识图谱测试样例

本目录提供法律助手法律知识图谱专项测试材料，用于验证助手能否围绕法律概念、法条、案例、主体、行为、责任和救济路径输出结构化关系，而不是只给普通问答式解释。

本测试只验证知识库问答或智能体是否能基于已入库法律条文和案例材料组织图谱化答案，不构成正式法律意见。

## 文件结构

| 文件 | 用途 |
| --- | --- |
| [test-cards.md](test-cards.md) | 7 条知识图谱测试卡、问题、预期节点/关系和通过标准 |
| [test-cases.json](test-cases.json) | 自动测试脚本使用的结构化用例配置 |
| [run_legal_knowledge_graph_tests.py](run_legal_knowledge_graph_tests.py) | 自动执行知识图谱用例并生成结果报告 |

## 前置条件

- 已有「法律条文」知识库，且相关法律法规已完成解析、分块和向量化。
- 已有「法律案例」知识库，且裁判案例材料已完成解析、分块和向量化。
- 第一轮样例默认不新建知识库，不上传新文档，仅创建临时会话并调用知识库问答或智能体问答。

当前默认知识库：

- 法律条文库：`f07af6bb-2645-428a-8db2-829708e3a2c2`
- 法律案例库：`4ca9a808-83f5-4222-8cc4-424ae24f6656`

## 自动执行

使用 API Key：

```bash
python3 docs/ictrek/legal-test-samples/legal-knowledge-graph/run_legal_knowledge_graph_tests.py \
  --host "http://localhost:8080" \
  --api-key "$WEKNORA_API_KEY" \
  --law-kb-id "f07af6bb-2645-428a-8db2-829708e3a2c2" \
  --case-kb-id "4ca9a808-83f5-4222-8cc4-424ae24f6656"
```

本地单用户模式也可以使用自动登录：

```bash
python3 docs/ictrek/legal-test-samples/legal-knowledge-graph/run_legal_knowledge_graph_tests.py \
  --host "http://localhost:8080" \
  --auto-setup \
  --law-kb-id "f07af6bb-2645-428a-8db2-829708e3a2c2" \
  --case-kb-id "4ca9a808-83f5-4222-8cc4-424ae24f6656"
```

如需指定知识图谱或法律问答智能体：

```bash
python3 docs/ictrek/legal-test-samples/legal-knowledge-graph/run_legal_knowledge_graph_tests.py \
  --host "http://localhost:8080" \
  --auto-setup \
  --law-kb-id "f07af6bb-2645-428a-8db2-829708e3a2c2" \
  --case-kb-id "4ca9a808-83f5-4222-8cc4-424ae24f6656" \
  --endpoint agent \
  --agent-id "<智能体ID>"
```

常用参数：

| 参数 | 说明 |
| --- | --- |
| `--endpoint knowledge|agent` | 选择 `knowledge-chat` 或 `agent-chat` |
| `--only <用例编号>` | 只执行指定用例，可重复传入 |
| `--output-dir <目录>` | 指定结果输出目录 |
| `--dry-run` | 只打印将执行的用例，不调用服务 |
| `--timeout <秒>` | 单次 HTTP/SSE 请求超时时间 |

输出目录默认为 `results/<时间戳>/`，包含：

- `summary.md`：通过数、图谱可用率和每条用例的机器判断；
- `results.json`：每条用例的完整回答、引用、事件和命中明细；
- `<用例编号>/response.md`：每条用例的用户问题、模型回答、节点/关系/引用命中和引用摘要。

脚本退出码说明：

- `0`：所有用例机器检查为 `PASS`；
- `1`：至少 1 条用例为 `REVIEW`、`FAIL` 或执行失败。

## 判定口径

单条知识图谱用例满足以下条件时，记为 `PASS`：

- 回答完成并非空；
- 输出具有图谱结构，允许 Markdown 表格、缩进列表、Mermaid graph、JSON-like 节点/边列表或显式“节点/关系”清单；
- 命中足够比例的预期节点；
- 命中足够比例的预期关系；
- 包含“依据 / 来源 / 法条 / 案例”等引用说明，或返回标准 `knowledge_references`；
- 未命中禁止内容模式，例如无依据编造案号、法条或确定性裁判结论。

判定分级：

| 判定 | 含义 |
| --- | --- |
| `PASS` | 核心节点、核心关系、结构化图谱和依据说明均满足 |
| `REVIEW` | 命中部分节点/关系，但结构化程度或依据说明不足，需要人工复核 |
| `FAIL` | 普通问答式回答、缺少图谱结构、缺少核心关系，或明显编造依据 |

## 结果统计

本样例默认执行 7 条知识图谱用例。

```text
法律知识图谱可用率 = PASS 用例数 / 7
```

自动判定只作为初筛；最终质量仍需人工复核图谱关系、引用来源和法条/案例映射准确性。
