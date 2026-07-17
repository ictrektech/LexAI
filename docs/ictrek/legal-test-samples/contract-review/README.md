# 合同审查测试样例

本目录提供法律助手合同审查专项测试材料，用于覆盖风险识别、合同原文引用、法律/案例依据引用、修改建议和长答案完整性。

所有合同均为人工构造的测试文本，不包含真实交易信息，不构成正式法律意见。

## 文件结构

| 文件 | 用途 |
| --- | --- |
| [test-cards.md](test-cards.md) | 5 份合同的测试卡、问题、期望风险点和通过标准 |
| [test-cases.json](test-cases.json) | 自动测试脚本使用的结构化用例配置 |
| [run_contract_review_tests.py](run_contract_review_tests.py) | 自动执行 10 条合同审查用例并生成结果报告 |
| [contracts/contract-001-equipment-purchase.md](contracts/contract-001-equipment-purchase.md) | 设备采购合同样例 |
| [contracts/contract-002-technical-service.md](contracts/contract-002-technical-service.md) | 技术服务合同样例 |
| [contracts/contract-003-saas-subscription.md](contracts/contract-003-saas-subscription.md) | SaaS 订阅服务合同样例 |
| [contracts/contract-004-lease-cooperation.md](contracts/contract-004-lease-cooperation.md) | 场地租赁及合作协议样例 |
| [contracts/contract-005-long-framework.md](contracts/contract-005-long-framework.md) | 长篇框架采购合同样例 |

## 前置条件

- 已有「法律条文」知识库，且相关法律法规已完成解析、分块和向量化。
- 已有「法律案例」知识库，且裁判案例材料已完成解析、分块和向量化。
- 已创建或可使用「合同审查问答」快速问答模板，或「合同审查」智能推理智能体。
- 第一轮测试不强制新建合同知识库；自动测试会把合同全文作为本轮问题文本提交。

## 执行方式

1. 打开「合同审查问答」或「合同审查」智能体。
2. 在输入框上方同时选择「法律条文」和「法律案例」两个知识库。
3. 上传本目录 `contracts/` 下的一份合同样例。
4. 按 [test-cards.md](test-cards.md) 中对应合同的 P0 和 P1 问题发起审查。
5. 记录回答、引用来源、执行时间、是否截断和人工复核结论。

## 自动执行

可以用脚本直接调用 API 跑完 10 条用例。脚本会把合同全文放入本轮问题，并同时传入「法律条文」和「法律案例」两个知识库 ID。

使用 API Key：

```bash
python3 docs/ictrek/legal-test-samples/contract-review/run_contract_review_tests.py \
  --host "http://localhost:8080" \
  --api-key "$WEKNORA_API_KEY" \
  --law-kb-id "<法律条文知识库ID>" \
  --case-kb-id "<法律案例知识库ID>"
```

本地单用户模式也可以使用自动登录：

```bash
python3 docs/ictrek/legal-test-samples/contract-review/run_contract_review_tests.py \
  --host "http://localhost:8080" \
  --auto-setup \
  --law-kb-id "<法律条文知识库ID>" \
  --case-kb-id "<法律案例知识库ID>"
```

如果要走「合同审查」智能推理智能体，传入智能体 ID：

```bash
python3 docs/ictrek/legal-test-samples/contract-review/run_contract_review_tests.py \
  --host "http://localhost:8080" \
  --api-key "$WEKNORA_API_KEY" \
  --law-kb-id "<法律条文知识库ID>" \
  --case-kb-id "<法律案例知识库ID>" \
  --endpoint agent \
  --agent-id "<合同审查智能体ID>"
```

判分模式默认使用 `--judge-mode auto`：

- `quick`：适合快速问答智能体，要求标准引用 `knowledge_references` 或答案中的明确条款引用；
- `reasoning`：适合智能推理智能体，除标准引用外，也接受 `grep_chunks`、`list_knowledge_chunks`、`knowledge_search` 等工具调用作为证据；
- `auto`：根据事件流自动识别。若检测到 `read_skill`、`grep_chunks`、`list_knowledge_chunks` 等智能推理工具轨迹，按 `reasoning` 判分，否则按 `quick` 判分。

如需强制指定：

```bash
python3 docs/ictrek/legal-test-samples/contract-review/run_contract_review_tests.py \
  --host "http://localhost:8080" \
  --api-key "$WEKNORA_API_KEY" \
  --law-kb-id "<法律条文知识库ID>" \
  --case-kb-id "<法律案例知识库ID>" \
  --endpoint agent \
  --agent-id "<合同审查智能体ID>" \
  --judge-mode reasoning
```

输出目录默认为 `results/<时间戳>/`，包含：

- `summary.md`：通过数、合同审查可用率和每条用例的机器判断；
- `results.json`：每条用例的完整回答、引用、事件和命中明细。

脚本退出码说明：

- `0`：所有用例机器检查通过；
- `1`：至少 1 条用例需要人工复核或执行失败。

## 判定口径

单条合同审查用例满足以下条件时，记为通过：

- 命中测试卡中的核心风险点，允许同义表达；
- 至少引用到合同原文中的相关条款；
- 快速问答模式能返回标准引用，或在答案中明确引用合同/法律条款；
- 智能推理模式能返回标准引用，或能通过工具调用证明使用了法律条文库/案例库证据；
- 修改建议具体可执行，不只停留在泛泛提醒；
- 对资料不足事项明确写出“无法判断”或“需补充确认”。

长合同完整报告额外检查：

- 包含审查摘要、总体风险等级、风险清单、重点条款修改、缺失条款建议、待确认事项；
- 回答末尾没有明显截断；
- 高风险和中风险至少各覆盖 1 项。

## 结果统计

每份合同执行 2 条问题，共 10 条合同审查用例。

```text
合同审查可用率 = 通过用例数 / 10
```

若同一条用例重复执行多次，以最后一次带完整引用记录的结果为准；关键问题建议至少重复 2 次，观察模型输出稳定性。
