# 多轮追问测试样例

本目录提供法律助手「多轮追问」专项测试材料，用于验证助手能否在同一会话中保持上下文、承接前轮事实、识别新增事实变化，并在追问中继续使用知识库依据，而不是每轮孤立回答。

本测试只验证知识库问答或智能体问答的多轮上下文能力，不构成正式法律意见。

## 文件结构

| 文件 | 用途 |
| --- | --- |
| [test-cards.md](test-cards.md) | 5 组多轮追问测试卡、逐轮预期承接点和通过标准 |
| [test-cases.json](test-cases.json) | 自动测试脚本使用的结构化用例配置 |
| [run_multi_turn_followup_tests.py](run_multi_turn_followup_tests.py) | 自动执行多轮追问用例并生成结果报告 |

## 前置条件

- 已有「法律条文」知识库，且相关法律法规已完成解析、分块和向量化。
- 已有「法律案例」知识库，且裁判案例材料已完成解析、分块和向量化。
- 默认知识库 ID：
  - 法律条文库：`f07af6bb-2645-428a-8db2-829708e3a2c2`
  - 法律案例库：`4ca9a808-83f5-4222-8cc4-424ae24f6656`
- 如在其他环境执行，可通过 `--law-kb-id` 和 `--case-kb-id` 覆盖。

## 自动执行

查看将执行的用例，不访问服务：

```bash
python3 docs/ictrek/legal-test-samples/multi-turn-followup/run_multi_turn_followup_tests.py --dry-run
```

使用 API Key：

```bash
python3 docs/ictrek/legal-test-samples/multi-turn-followup/run_multi_turn_followup_tests.py \
  --host "http://localhost:8080" \
  --api-key "$WEKNORA_API_KEY"
```

本地单用户模式也可以使用自动登录：

```bash
python3 docs/ictrek/legal-test-samples/multi-turn-followup/run_multi_turn_followup_tests.py \
  --host "http://localhost:8080" \
  --auto-setup
```

如需指定法律问答或法律助手智能体：

```bash
python3 docs/ictrek/legal-test-samples/multi-turn-followup/run_multi_turn_followup_tests.py \
  --host "http://localhost:8080" \
  --auto-setup \
  --endpoint agent \
  --agent-id "<法律助手智能体ID>"
```

只执行单组用例并指定输出目录：

```bash
python3 docs/ictrek/legal-test-samples/multi-turn-followup/run_multi_turn_followup_tests.py \
  --host "http://localhost:8080" \
  --auto-setup \
  --only MTF-001 \
  --output-dir /tmp/multi-turn-followup-smoke
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

- `summary.md`：通过数、多轮追问通过率、每组用例的机器判断和逐轮命中情况；
- `results.json`：每组用例的完整回答、引用、事件、session_id 和命中明细；
- `<用例编号>/turn-01-response.md`、`turn-02-response.md` 等：逐轮问题、回答、引用摘要和机器判定；
- 每组用例只创建一个临时会话，所有轮次复用同一个 `session_id`。

脚本退出码说明：

- `0`：所有用例机器检查为 `PASS`；
- `1`：至少 1 条用例为 `REVIEW`、`FAIL` 或执行失败。

## 判定口径

单组多轮追问用例满足以下条件时，记为 `PASS`：

- 多数轮次完成回答，且回答不是空文本；
- 每轮能引用或复述前轮关键事实，例如“试用期”“网购”“延期交付”等；
- 每轮能识别本轮新增事实，并据此调整法律结论、风险等级或处理路径；
- 每轮继续给出法规、案例或标准引用依据；
- 未出现自相矛盾、遗忘上下文、把已否定事实当成事实、编造法条或案号等禁止行为；
- 同一组所有轮次的请求使用同一个 `session_id`。

判定分级：

| 判定 | 含义 |
| --- | --- |
| `PASS` | 多数轮次同时满足承接、增量判断、依据说明，且未出现明显禁用内容 |
| `REVIEW` | 能回答追问，但上下文承接、增量判断或依据说明不足，需要人工复核 |
| `FAIL` | 每轮像新问题、明显遗忘前文、无依据确定性结论，或未复用同一会话 |

自动判定只作为初筛；最终质量仍需人工复核上下文承接是否真实、引用来源是否准确、法律结论是否稳妥。

## 结果统计

本样例默认执行 5 组多轮追问用例。

```text
多轮追问通过率 = PASS 用例数 / 5
```

关键用例建议至少重复执行 2 次，观察模型在同一 session 内的上下文稳定性。
