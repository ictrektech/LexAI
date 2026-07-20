# 法律助手测试样例

本目录保存法律助手专项测试样例和执行脚本。每个子目录都有独立脚本，可单独跑该专项；根目录的 `run_full_legal_assistant_batch.py` 用于按顺序执行完整批次并生成总汇总。

## 一键完整批次

在已部署后端可访问的机器上执行：

```bash
python3 docs/ictrek/legal-test-samples/run_full_legal_assistant_batch.py \
  --host http://localhost:8080 \
  --auto-setup \
  --law-kb-id f07af6bb-2645-428a-8db2-829708e3a2c2 \
  --case-kb-id 4ca9a808-83f5-4222-8cc4-424ae24f6656 \
  --output-root /tmp/legal-assistant-full-$(date +%Y%m%d-%H%M%S)
```

默认会顺序执行：

| 专项 | 子脚本 |
| --- | --- |
| 法律法规问答 | `legal-qa/run_legal_qa_tests.py` |
| 裁判案例问答 | `case-qa/run_case_qa_tests.py` |
| 无依据拒答 | `no-evidence-refusal/run_no_evidence_refusal_tests.py` |
| 法律知识图谱 | `legal-knowledge-graph/run_legal_knowledge_graph_tests.py` |
| 多轮追问 | `multi-turn-followup/run_multi_turn_followup_tests.py` |
| 合同审查（快速问答） | `contract-review/run_contract_review_tests.py` |
| 合同审查（智能推理） | `contract-review/run_contract_review_tests.py` |

总控脚本默认遇到某个专项返回非 0 也会继续跑后续专项，最后统一写出：

- `summary.md`：批次汇总、非 PASS 用例和重跑命令；
- `manifest.json`：机器可读的批次记录，用于之后只重跑失败或需复核用例；
- `<专项>/summary.md` 和 `<专项>/results.json`：各专项自己的结果。

如果希望某个专项出现非 PASS 后立即停止，可以加：

```bash
--fail-fast
```

如果希望即使有失败项也让总控脚本返回退出码 0，可以加：

```bash
--allow-failures
```

## 只重跑失败项

完整批次结束后，可以根据上一轮 `manifest.json` 自动只重跑 `REVIEW`、`FAIL`、`ERROR` 和 `NON_PASS` 用例：

```bash
python3 docs/ictrek/legal-test-samples/run_full_legal_assistant_batch.py \
  --rerun-from /tmp/legal-assistant-full-20260717 \
  --host http://localhost:8080 \
  --auto-setup
```

未显式传 `--output-root` 时，重跑结果默认写到上一轮目录下的 `rerun-<时间戳>/` 子目录，例如 `/tmp/legal-assistant-full-20260717/rerun-20260720-123000/`。这样不会覆盖旧结果，也能把补跑结果和原批次放在一起。

如果只是想预览会重跑哪些用例，不实际请求后端，可以加 `--dry-run`：

```bash
python3 docs/ictrek/legal-test-samples/run_full_legal_assistant_batch.py \
  --dry-run \
  --rerun-from /tmp/legal-assistant-full-20260717 \
  --host http://localhost:8080 \
  --auto-setup
```

`--dry-run` 只打印计划执行的子命令，不创建结果目录，也不写 `manifest.json`。

对于总控脚本出现前手工跑出来的旧批次，如果目录下没有 `manifest.json`，脚本会尝试扫描各专项子目录中的 `results.json` 来推导需要重跑的用例。

也可以只重跑某几类状态：

```bash
python3 docs/ictrek/legal-test-samples/run_full_legal_assistant_batch.py \
  --rerun-from /tmp/legal-assistant-full-20260717 \
  --rerun-status FAIL \
  --rerun-status ERROR \
  --host http://localhost:8080 \
  --auto-setup
```

## 只跑指定用例

使用 `--only <专项>:<用例ID>` 可以精确重跑单条或多条用例：

```bash
python3 docs/ictrek/legal-test-samples/run_full_legal_assistant_batch.py \
  --only legal-qa:LAWQA-007 \
  --only no-evidence-refusal:REFUSAL-003 \
  --host http://localhost:8080 \
  --auto-setup
```

可用专项名：

- `legal-qa`
- `case-qa`
- `no-evidence-refusal`
- `legal-knowledge-graph`
- `multi-turn-followup`
- `contract-review-quick`
- `contract-review-reasoning`

## 只跑某个专项

```bash
python3 docs/ictrek/legal-test-samples/run_full_legal_assistant_batch.py \
  --suite legal-knowledge-graph \
  --host http://localhost:8080 \
  --auto-setup
```

`--suite` 可以重复传入。未传时默认跑完整批次。
