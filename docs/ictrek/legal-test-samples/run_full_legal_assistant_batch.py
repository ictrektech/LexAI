#!/usr/bin/env python3
"""Run the full legal assistant sample batch and summarize rerunnable failures."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent
DEFAULT_LAW_KB_ID = "f07af6bb-2645-428a-8db2-829708e3a2c2"
DEFAULT_CASE_KB_ID = "4ca9a808-83f5-4222-8cc4-424ae24f6656"
DEFAULT_CONTRACT_QUICK_AGENT_ID = "90fd6ab5-ba06-4bff-8a23-878ce00837ef"
DEFAULT_CONTRACT_REASONING_AGENT_ID = "511258fd-4f3f-419e-8053-09f652ab50a5"
DEFAULT_RERUN_STATUSES = {"REVIEW", "FAIL", "ERROR", "NON_PASS"}


@dataclass(frozen=True)
class Suite:
    key: str
    title: str
    script: Path
    output_name: str
    needs_law_kb: bool = False
    needs_case_kb: bool = False
    endpoint: str | None = None
    agent_attr: str | None = None
    judge_mode: str | None = None
    non_pass_status: str = "REVIEW"


SUITES: dict[str, Suite] = {
    "legal-qa": Suite(
        key="legal-qa",
        title="法律法规问答",
        script=ROOT / "legal-qa" / "run_legal_qa_tests.py",
        output_name="legal-qa",
        needs_law_kb=True,
    ),
    "case-qa": Suite(
        key="case-qa",
        title="裁判案例问答",
        script=ROOT / "case-qa" / "run_case_qa_tests.py",
        output_name="case-qa",
        needs_law_kb=True,
        needs_case_kb=True,
    ),
    "no-evidence-refusal": Suite(
        key="no-evidence-refusal",
        title="无依据拒答",
        script=ROOT / "no-evidence-refusal" / "run_no_evidence_refusal_tests.py",
        output_name="no-evidence-refusal",
        needs_law_kb=True,
        needs_case_kb=True,
        non_pass_status="FAIL",
    ),
    "legal-knowledge-graph": Suite(
        key="legal-knowledge-graph",
        title="法律知识图谱",
        script=ROOT / "legal-knowledge-graph" / "run_legal_knowledge_graph_tests.py",
        output_name="legal-knowledge-graph",
        needs_law_kb=True,
        needs_case_kb=True,
        non_pass_status="NON_PASS",
    ),
    "multi-turn-followup": Suite(
        key="multi-turn-followup",
        title="多轮追问",
        script=ROOT / "multi-turn-followup" / "run_multi_turn_followup_tests.py",
        output_name="multi-turn-followup",
        needs_law_kb=True,
        needs_case_kb=True,
        non_pass_status="NON_PASS",
    ),
    "contract-review-quick": Suite(
        key="contract-review-quick",
        title="合同审查（快速问答）",
        script=ROOT / "contract-review" / "run_contract_review_tests.py",
        output_name="contract-review-agent-quick",
        needs_law_kb=True,
        needs_case_kb=True,
        endpoint="agent",
        agent_attr="contract_quick_agent_id",
        judge_mode="auto",
    ),
    "contract-review-reasoning": Suite(
        key="contract-review-reasoning",
        title="合同审查（智能推理）",
        script=ROOT / "contract-review" / "run_contract_review_tests.py",
        output_name="contract-review-agent-reasoning",
        needs_law_kb=True,
        needs_case_kb=True,
        endpoint="agent",
        agent_attr="contract_reasoning_agent_id",
        judge_mode="auto",
    ),
}
DEFAULT_SUITE_ORDER = [
    "legal-qa",
    "case-qa",
    "no-evidence-refusal",
    "legal-knowledge-graph",
    "multi-turn-followup",
    "contract-review-quick",
    "contract-review-reasoning",
]
OUTPUT_NAME_TO_SUITE = {suite.output_name: suite.key for suite in SUITES.values()}
OUTPUT_NAME_TO_SUITE["legal-knowledge-graph-selected"] = "legal-knowledge-graph"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run all legal assistant sample suites and write a full-batch summary."
    )
    parser.add_argument("--host", default=os.getenv("WEKNORA_HOST") or "http://localhost:8080")
    parser.add_argument("--api-key", default=os.getenv("WEKNORA_API_KEY"))
    parser.add_argument("--bearer-token", default=os.getenv("WEKNORA_BEARER_TOKEN") or os.getenv("WEKNORA_TOKEN"))
    parser.add_argument("--auto-setup", action="store_true", help="Pass --auto-setup to child suites.")
    parser.add_argument("--law-kb-id", default=os.getenv("LEGAL_LAW_KB_ID") or DEFAULT_LAW_KB_ID)
    parser.add_argument("--case-kb-id", default=os.getenv("LEGAL_CASE_KB_ID") or DEFAULT_CASE_KB_ID)
    parser.add_argument(
        "--contract-quick-agent-id",
        default=os.getenv("WEKNORA_CONTRACT_QUICK_AGENT_ID") or DEFAULT_CONTRACT_QUICK_AGENT_ID,
    )
    parser.add_argument(
        "--contract-reasoning-agent-id",
        default=os.getenv("WEKNORA_CONTRACT_REASONING_AGENT_ID") or DEFAULT_CONTRACT_REASONING_AGENT_ID,
    )
    parser.add_argument(
        "--suite",
        action="append",
        choices=DEFAULT_SUITE_ORDER,
        default=[],
        help="Suite to run. Can be repeated. Defaults to all suites.",
    )
    parser.add_argument(
        "--output-root",
        type=Path,
        default=None,
        help=(
            "Batch output directory. Defaults to legal-test-samples/results/full-<timestamp>. "
            "With --rerun-from, defaults to <rerun-from>/rerun-<timestamp>."
        ),
    )
    parser.add_argument(
        "--rerun-from",
        type=Path,
        default=None,
        help="Previous full-batch output directory. Reruns only non-PASS cases recorded in manifest.json.",
    )
    parser.add_argument(
        "--rerun-status",
        action="append",
        default=[],
        help="Status to rerun from a previous manifest. Repeatable. Defaults to REVIEW, FAIL, ERROR and NON_PASS.",
    )
    parser.add_argument(
        "--only",
        action="append",
        default=[],
        help="Run only one case in the form suite:test_id, for example legal-qa:LAWQA-007. Repeatable.",
    )
    parser.add_argument("--fail-fast", action="store_true", help="Stop after the first suite with non-PASS cases or command error.")
    parser.add_argument(
        "--allow-failures",
        action="store_true",
        help="Exit 0 even when some suites have REVIEW/FAIL/ERROR cases.",
    )
    parser.add_argument(
        "--suite-timeout",
        type=int,
        default=0,
        help="Optional timeout in seconds for each child suite. 0 means no orchestrator timeout.",
    )
    parser.add_argument("--dry-run", action="store_true", help="Print planned child commands without running them.")
    return parser.parse_args()


def default_output_root() -> Path:
    stamp = dt.datetime.now().strftime("%Y%m%d-%H%M%S")
    return ROOT / "results" / f"full-{stamp}"


def default_rerun_output_root(rerun_from: Path) -> Path:
    stamp = dt.datetime.now().strftime("%Y%m%d-%H%M%S")
    return rerun_from / f"rerun-{stamp}"


def auth_args(args: argparse.Namespace) -> list[str]:
    if args.auto_setup:
        return ["--auto-setup"]
    if args.api_key:
        return ["--api-key", args.api_key]
    if args.bearer_token:
        return ["--bearer-token", args.bearer_token]
    raise SystemExit("Missing auth. Pass --auto-setup, --api-key, or --bearer-token.")


def load_manifest(path: Path) -> dict[str, Any]:
    manifest_path = path / "manifest.json"
    if not manifest_path.exists():
        return legacy_manifest(path)
    with manifest_path.open("r", encoding="utf-8") as f:
        return json.load(f)


def legacy_manifest(path: Path) -> dict[str, Any]:
    if not path.exists():
        raise SystemExit(f"Missing batch directory: {path}")
    suites: list[dict[str, Any]] = []
    for results_path in sorted(path.glob("*/results.json")):
        output_name = results_path.parent.name
        suite_key = OUTPUT_NAME_TO_SUITE.get(output_name)
        if not suite_key:
            continue
        suite = SUITES[suite_key]
        parsed = parse_suite_output(suite, results_path.parent)
        suites.append(
            {
                "key": suite.key,
                "title": suite.title,
                "output_dir": str(results_path.parent),
                "total": parsed.get("total", 0),
                "passed": parsed.get("passed", 0),
                "non_pass_cases": parsed.get("non_pass_cases", []),
            }
        )
    if not suites:
        raise SystemExit(f"Missing manifest and no child results.json files found in: {path}")
    return {
        "generated_at": None,
        "host": None,
        "law_kb_id": None,
        "case_kb_id": None,
        "legacy_from": str(path),
        "suites": suites,
    }


def suite_selection(args: argparse.Namespace) -> list[str]:
    if args.suite:
        return args.suite
    return DEFAULT_SUITE_ORDER.copy()


def only_map(values: list[str]) -> dict[str, list[str]]:
    selected: dict[str, list[str]] = {}
    for value in values:
        if ":" not in value:
            raise SystemExit(f"--only must use suite:test_id format, got: {value}")
        suite_key, test_id = value.split(":", 1)
        if suite_key not in SUITES:
            raise SystemExit(f"Unknown suite in --only: {suite_key}")
        if not test_id:
            raise SystemExit(f"Missing test id in --only: {value}")
        selected.setdefault(suite_key, []).append(test_id)
    return selected


def rerun_map(args: argparse.Namespace) -> dict[str, list[str]]:
    if not args.rerun_from:
        return {}
    manifest = load_manifest(args.rerun_from)
    wanted = set(args.rerun_status or DEFAULT_RERUN_STATUSES)
    selected: dict[str, list[str]] = {}
    for suite in manifest.get("suites", []):
        ids = [
            case["id"]
            for case in suite.get("non_pass_cases", [])
            if case.get("status") in wanted
        ]
        if ids:
            selected[suite["key"]] = ids
    return selected


def build_command(suite: Suite, args: argparse.Namespace, output_dir: Path, only_ids: list[str]) -> list[str]:
    cmd = [sys.executable, str(suite.script), "--host", args.host, *auth_args(args), "--output-dir", str(output_dir)]
    if suite.needs_law_kb:
        cmd.extend(["--law-kb-id", args.law_kb_id])
    if suite.needs_case_kb:
        cmd.extend(["--case-kb-id", args.case_kb_id])
    if suite.endpoint:
        cmd.extend(["--endpoint", suite.endpoint])
    if suite.agent_attr:
        agent_id = getattr(args, suite.agent_attr)
        if agent_id:
            cmd.extend(["--agent-id", agent_id])
    if suite.judge_mode:
        cmd.extend(["--judge-mode", suite.judge_mode])
    for test_id in only_ids:
        cmd.extend(["--only", test_id])
    return cmd


def result_status(suite: Suite, result: dict[str, Any]) -> str:
    judge = result.get("judge") or {}
    if isinstance(judge.get("status"), str):
        return judge["status"]
    if judge.get("passed") is True:
        return "PASS"
    if judge.get("passed") is False:
        return suite.non_pass_status
    return "ERROR"


def parse_suite_output(suite: Suite, output_dir: Path) -> dict[str, Any]:
    results_path = output_dir / "results.json"
    summary_path = output_dir / "summary.md"
    parsed: dict[str, Any] = {
        "summary": {},
        "results_path": str(results_path),
        "summary_path": str(summary_path),
        "total": 0,
        "passed": 0,
        "non_pass_cases": [],
    }
    if not results_path.exists():
        parsed["parse_error"] = f"missing {results_path}"
        return parsed
    try:
        with results_path.open("r", encoding="utf-8") as f:
            payload = json.load(f)
    except (OSError, json.JSONDecodeError) as exc:
        parsed["parse_error"] = str(exc)
        return parsed

    parsed["summary"] = payload.get("summary") or {}
    results = payload.get("results") or []
    parsed["total"] = len(results)
    for item in results:
        status = result_status(suite, item)
        if status == "PASS":
            parsed["passed"] += 1
        else:
            parsed["non_pass_cases"].append(
                {
                    "id": item.get("id") or item.get("test_id") or "unknown",
                    "status": status,
                }
            )
    return parsed


def run_suite(suite: Suite, args: argparse.Namespace, batch_root: Path, only_ids: list[str]) -> dict[str, Any]:
    output_dir = batch_root / suite.output_name
    cmd = build_command(suite, args, output_dir, only_ids)
    record: dict[str, Any] = {
        "key": suite.key,
        "title": suite.title,
        "output_dir": str(output_dir),
        "only": only_ids,
        "command": cmd,
    }

    print(f"[suite] {suite.key}: {suite.title}")
    print("  " + " ".join(cmd))
    if args.dry_run:
        record.update({"exit_code": None, "dry_run": True, "total": 0, "passed": 0, "non_pass_cases": []})
        return record

    output_dir.mkdir(parents=True, exist_ok=True)
    try:
        completed = subprocess.run(
            cmd,
            cwd=ROOT.parents[2],
            text=True,
            timeout=args.suite_timeout or None,
        )
        record["exit_code"] = completed.returncode
    except subprocess.TimeoutExpired as exc:
        record["exit_code"] = 124
        record["command_error"] = f"suite timed out after {exc.timeout}s"

    parsed = parse_suite_output(suite, output_dir)
    record.update(parsed)
    if record.get("command_error") and not record["non_pass_cases"]:
        record["non_pass_cases"] = [{"id": "*", "status": "ERROR"}]
    if record.get("parse_error") and not record["non_pass_cases"]:
        record["non_pass_cases"] = [{"id": "*", "status": "ERROR"}]
    return record


def write_summary(batch_root: Path, manifest: dict[str, Any]) -> None:
    lines = [
        "# 法律助手完整批次结果",
        "",
        f"- 生成时间：{manifest['generated_at']}",
        f"- 服务地址：{manifest['host']}",
        f"- 法律条文库：{manifest['law_kb_id']}",
        f"- 法律案例库：{manifest['case_kb_id']}",
        f"- 结果根目录：`{batch_root}`",
        "",
        "## 汇总",
        "",
        "| 专项 | 结果 | 非 PASS | 退出码 | 结果目录 |",
        "| --- | ---: | ---: | ---: | --- |",
    ]
    for suite in manifest["suites"]:
        non_pass_count = len(suite.get("non_pass_cases", []))
        total = suite.get("total", 0)
        passed = suite.get("passed", 0)
        lines.append(
            f"| {suite['title']} | {passed} / {total} | {non_pass_count} | {suite.get('exit_code')} | `{suite['output_dir']}` |"
        )

    failed_suites = [s for s in manifest["suites"] if s.get("non_pass_cases")]
    lines.extend(["", "## 失败和复核项", ""])
    if not failed_suites:
        lines.append("- 无。")
    else:
        for suite in failed_suites:
            cases = ", ".join(f"{case['id']}({case['status']})" for case in suite["non_pass_cases"])
            lines.append(f"- `{suite['key']}`：{cases}")

    lines.extend(
        [
            "",
            "## 只重跑失败项",
            "",
            "默认重跑上一轮 manifest 中的 `REVIEW`、`FAIL`、`ERROR` 和 `NON_PASS`：",
            "",
            "```bash",
            f"python3 docs/ictrek/legal-test-samples/run_full_legal_assistant_batch.py --rerun-from {batch_root} --host {manifest['host']} --auto-setup",
            "```",
            "",
            "也可以指定单条用例：",
            "",
            "```bash",
            "python3 docs/ictrek/legal-test-samples/run_full_legal_assistant_batch.py --only legal-qa:LAWQA-007 --host http://localhost:8080 --auto-setup",
            "```",
        ]
    )
    with (batch_root / "summary.md").open("w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def main() -> int:
    args = parse_args()
    batch_root = args.output_root or (default_rerun_output_root(args.rerun_from) if args.rerun_from else default_output_root())

    explicit_only = only_map(args.only)
    previous_only = rerun_map(args)
    if explicit_only:
        selected_only = explicit_only
        selected_suites = list(selected_only.keys())
    elif args.rerun_from:
        selected_only = previous_only
        selected_suites = list(selected_only.keys())
    else:
        selected_only = {}
        selected_suites = suite_selection(args)
    if args.suite:
        selected_suites = [suite_key for suite_key in selected_suites if suite_key in set(args.suite)]

    manifest: dict[str, Any] = {
        "generated_at": dt.datetime.now().isoformat(timespec="seconds"),
        "host": args.host,
        "law_kb_id": args.law_kb_id,
        "case_kb_id": args.case_kb_id,
        "rerun_from": str(args.rerun_from) if args.rerun_from else None,
        "suites": [],
    }

    if args.rerun_from and not selected_suites:
        print("No cases selected from previous manifest.")

    for suite_key in selected_suites:
        suite = SUITES[suite_key]
        only_ids = selected_only.get(suite_key, []) if selected_only else []
        record = run_suite(suite, args, batch_root, only_ids)
        manifest["suites"].append(record)
        if args.fail_fast and record.get("non_pass_cases"):
            break

    if args.dry_run:
        print("dry-run only; no files written")
        return 0

    batch_root.mkdir(parents=True, exist_ok=True)
    with (batch_root / "manifest.json").open("w", encoding="utf-8") as f:
        json.dump(manifest, f, ensure_ascii=False, indent=2)
    write_summary(batch_root, manifest)

    non_pass_total = sum(len(suite.get("non_pass_cases", [])) for suite in manifest["suites"])
    print(f"wrote {batch_root / 'manifest.json'}")
    print(f"wrote {batch_root / 'summary.md'}")
    if non_pass_total:
        print(f"non-PASS cases: {non_pass_total}")
        return 0 if args.allow_failures else 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
