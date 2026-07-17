# 裁判案例问答测试样例

本目录提供法律助手裁判案例问答专项测试材料，用于覆盖案件事实问答、裁判依据问答、争议焦点问答、法院观点/裁判理由问答、引用来源准确性，以及案例事实与法律依据区分。

本测试只验证知识库问答是否能基于已入库裁判案例作答，不构成正式法律意见。

## 文件结构

| 文件 | 用途 |
| --- | --- |
| [test-cards.md](test-cards.md) | 案例问答测试卡、问题、期望要点和通过标准 |
| [test-cases.json](test-cases.json) | 自动测试脚本使用的结构化用例配置 |
| [run_case_qa_tests.py](run_case_qa_tests.py) | 自动执行案例问答用例并生成结果报告 |

## 前置条件

- 已有「法律案例」知识库，且案例文档已完成解析、分块和向量化。
- 第一轮样例默认使用当前法律案例库中的文档，不新建知识库。
- 可选传入「法律条文」知识库，用于辅助回答裁判依据、法律条文或司法解释问题。

当前样例按以下已验证案例文档设计：

- `吴小秦诉陕西广电网络传媒（集团）股份有限公司捆绑交易纠纷案.md`
- `孙银山诉南京欧尚超市有限公司江宁店买卖合同纠纷案.md`
- `吴某与某电子商务有限公司买卖合同纠纷案.md`
- `用人单位与劳动者约定实行包薪制，是否需要依法支付加班费.md`
- `处理加班费争议，如何分配举证责任.md`
- `劳动者拒绝违法超时加班安排，用人单位能否解除劳动合同.md`
- `宁某某诉甘肃省定西市安定区住房和城乡建设局房屋征收补偿安置协议案.md`
- `游戏“外挂脚本”被封号 玩家起诉游戏公司被驳回.md`
- `石某某诉A医院医疗服务合同案.md`

## 自动执行

使用 API Key：

```bash
python3 docs/ictrek/legal-test-samples/case-qa/run_case_qa_tests.py \
  --host "http://localhost:8080" \
  --api-key "$WEKNORA_API_KEY" \
  --case-kb-id "<法律案例知识库ID>"
```

本地单用户模式也可以使用自动登录：

```bash
python3 docs/ictrek/legal-test-samples/case-qa/run_case_qa_tests.py \
  --host "http://localhost:8080" \
  --auto-setup \
  --case-kb-id "<法律案例知识库ID>"
```

如需同时使用法律条文库辅助回答裁判依据：

```bash
python3 docs/ictrek/legal-test-samples/case-qa/run_case_qa_tests.py \
  --host "http://localhost:8080" \
  --auto-setup \
  --case-kb-id "<法律案例知识库ID>" \
  --law-kb-id "<法律条文知识库ID>"
```

如需指定案例问答智能体：

```bash
python3 docs/ictrek/legal-test-samples/case-qa/run_case_qa_tests.py \
  --host "http://localhost:8080" \
  --auto-setup \
  --case-kb-id "<法律案例知识库ID>" \
  --law-kb-id "<法律条文知识库ID>" \
  --endpoint agent \
  --agent-id "<案例问答智能体ID>"
```

输出目录默认为 `results/<时间戳>/`，包含：

- `summary.md`：通过数、案例问答可用率和每条用例的机器判断；
- `results.json`：每条用例的完整回答、引用、事件和命中明细。

脚本退出码说明：

- `0`：所有用例机器检查通过；
- `1`：至少 1 条用例需要人工复核或执行失败。

## 判定口径

单条案例问答用例满足以下条件时，记为通过：

- 命中测试卡中的核心答案要点，允许同义表达；
- 回答中能出现目标案例名称、当事人、关键事实、裁判依据或裁判理由；
- 返回标准引用 `knowledge_references`，且引用或答案命中配置的 `citation_terms`；
- 对事实问题能说明案件事实，对依据问题能区分案件事实、裁判依据和法院观点；
- 回答没有明显偏离目标案例或混入其他案例事实。

`PASS` 只代表机器检查命中核心事实、依据和引用要求，不代表法律质量通过。`REVIEW` 需要人工打开 `results.json` 复核完整回答、引用位置和未命中的检查项。

## 结果统计

本样例默认执行 10 条案例问答用例。

```text
裁判案例问答可用率 = 通过用例数 / 10
```

自动判定只作为初筛；最终质量仍需人工复核案例事实、法律依据和引用来源准确性。
