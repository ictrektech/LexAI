#!/usr/bin/env python3
"""Run court-case QA sample tests against a LexAI/WeKnora server."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import sys
import time
from pathlib import Path
from typing import Any
from urllib import error, request


ROOT = Path(__file__).resolve().parent
DEFAULT_CASES = ROOT / "test-cases.json"
DEFAULT_OUTPUT_ROOT = ROOT / "results"
KEYWORD_SYNONYMS = {
    "市场支配地位": ["市场支配地位", "支配地位", "优势地位"],
    "真实选择权": ["真实选择权", "自主选择权", "选择权", "自由选择"],
    "过保质期": ["过保质期", "超过保质期", "过期"],
    "十倍赔偿": ["十倍赔偿", "价款十倍", "十倍惩罚性赔偿"],
    "生活消费": ["生活消费", "个人、家庭生活", "生活需要"],
    "未查验": ["未查验", "未履行查验", "怠于履行查验"],
    "进货查验义务": ["进货查验义务", "查验义务", "进货查验"],
    "强制性": ["强制性", "强制性规定", "法定"],
    "补足": ["补足", "支付", "补付"],
    "打卡APP": ["打卡APP", "打卡 App", "打卡记录"],
    "拒不提供": ["拒不提供", "未提交", "不能提供"],
    "不利后果": ["不利后果", "不利的后果", "承担不利"],
    "超时加班": ["超时加班", "违法超时加班", "延长工作时间"],
    "租金收益": ["租金收益", "房屋收益", "租金损失"],
    "后台数据": ["后台数据", "服务器数据", "数据分析"],
    "高度盖然性": ["高度盖然性", "高度可能", "盖然性"],
    "充分告知": ["充分告知", "清晰提示", "明确告知"],
    "诚实信用": ["诚实信用", "诚信原则", "诚信"],
    "辅助生殖": ["辅助生殖", "人类辅助生殖", "辅助生殖技术"],
    "伦理道德": ["伦理道德", "伦理", "社会伦理"],
    "同等法律权利": ["同等法律权利", "同等权利", "同等法律地位"],
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run the court-case QA sample suite and write JSON/Markdown results."
    )
    parser.add_argument("--host", default=os.getenv("WEKNORA_HOST") or os.getenv("WEKNORA_E2E_HOST") or "http://localhost:8080")
    parser.add_argument("--api-key", default=os.getenv("WEKNORA_API_KEY"), help="API key for X-API-Key auth.")
    parser.add_argument(
        "--bearer-token",
        default=os.getenv("WEKNORA_BEARER_TOKEN") or os.getenv("WEKNORA_TOKEN"),
        help="JWT token for Authorization: Bearer auth.",
    )
    parser.add_argument(
        "--auto-setup",
        action="store_true",
        help="Call /auth/auto-setup and use the returned JWT. Only works in lite/single-user mode.",
    )
    parser.add_argument("--case-kb-id", default=os.getenv("LEGAL_CASE_KB_ID"), required=os.getenv("LEGAL_CASE_KB_ID") is None)
    parser.add_argument("--law-kb-id", default=os.getenv("LEGAL_LAW_KB_ID"), help="Optional statute KB ID for legal-basis questions.")
    parser.add_argument("--agent-id", default=os.getenv("WEKNORA_CASE_QA_AGENT_ID"), help="Optional case-QA agent ID.")
    parser.add_argument(
        "--endpoint",
        choices=("knowledge", "agent"),
        default=os.getenv("CASE_QA_ENDPOINT", "knowledge"),
        help="Use knowledge-chat or agent-chat.",
    )
    parser.add_argument("--cases", type=Path, default=DEFAULT_CASES)
    parser.add_argument("--output-dir", type=Path, default=None)
    parser.add_argument("--only", action="append", default=[], help="Run only matching test IDs. Can be passed more than once.")
    parser.add_argument("--timeout", type=int, default=int(os.getenv("CASE_QA_TIMEOUT", "180")))
    parser.add_argument("--dry-run", action="store_true", help="Print selected tests without calling the server.")
    return parser.parse_args()


def load_cases(path: Path, only: list[str]) -> list[dict[str, Any]]:
    with path.open("r", encoding="utf-8") as f:
        payload = json.load(f)
    tests = payload.get("tests", [])
    if only:
        wanted = set(only)
        tests = [t for t in tests if t["id"] in wanted or t.get("document") in wanted]
    if not tests:
        raise SystemExit("No tests selected.")
    return tests


def api_url(host: str, path: str) -> str:
    return host.rstrip("/") + "/api/v1" + path


def json_request(url: str, headers: dict[str, str], payload: dict[str, Any], timeout: int) -> dict[str, Any]:
    body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = request.Request(url, data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    for key, value in headers.items():
        req.add_header(key, value)
    try:
        with request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
    except error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {exc.code} from {url}: {detail}") from exc
    return json.loads(raw)


def auto_setup(host: str, timeout: int) -> str:
    response = json_request(api_url(host, "/auth/auto-setup"), {}, {}, timeout)
    token = response.get("token") or response.get("data", {}).get("token")
    if not token:
        raise RuntimeError("/auth/auto-setup did not return token")
    return token


def create_session(host: str, headers: dict[str, str], test_id: str, timeout: int) -> str:
    response = json_request(
        api_url(host, "/sessions"),
        headers,
        {"title": f"case-qa-test {test_id}", "description": "Automated court-case QA sample test"},
        timeout,
    )
    session = response.get("data") or {}
    session_id = session.get("id")
    if not session_id:
        raise RuntimeError("create session did not return data.id")
    return session_id


def auth_headers(args: argparse.Namespace) -> dict[str, str]:
    if args.auto_setup:
        args.bearer_token = auto_setup(args.host, args.timeout)
    if args.api_key:
        return {"X-API-Key": args.api_key}
    if args.bearer_token:
        return {"Authorization": f"Bearer {args.bearer_token}"}
    raise SystemExit("Missing auth. Pass --api-key, --bearer-token, or --auto-setup.")


def build_query(test: dict[str, Any], has_law_kb: bool) -> str:
    law_note = "如涉及具体法律条文，可同时参考已选择的法律条文知识库。" if has_law_kb else "如案例文档未包含具体法条，请明确说明现有案例库依据不足，不要编造法条。"
    return f"""你是裁判案例问答测试执行助手。请只基于已选择的法律案例知识库回答；{law_note}

目标案例文档：{test["document"]}
测试问题：{test["question"]}

输出要求：
1. 先给出简明结论。
2. 单独列出“案件事实”。
3. 涉及法律适用时，单独列出“裁判依据”。
4. 单独列出“法院观点/裁判理由”。
5. 单独列出“来源引用”，写明案例名称；如有法律条文依据，写明法律名称和条文编号。
6. 区分案例事实与法律依据，不要把事实表述伪装成法条。
7. 如果知识库没有检索到依据，请明确写“现有知识库未检索到可用依据”，不要编造案例或法条。
"""


def selected_kb_ids(args: argparse.Namespace) -> list[str]:
    kb_ids = [args.case_kb_id]
    if args.law_kb_id:
        kb_ids.append(args.law_kb_id)
    return kb_ids


def mentioned_items(args: argparse.Namespace) -> list[dict[str, str]]:
    items = [{"id": args.case_kb_id, "name": "法律案例", "type": "kb", "kb_type": "document"}]
    if args.law_kb_id:
        items.append({"id": args.law_kb_id, "name": "法律条文", "type": "kb", "kb_type": "document"})
    return items


def chat_stream(args: argparse.Namespace, headers: dict[str, str], test_id: str, query: str) -> dict[str, Any]:
    session_id = create_session(args.host, headers, test_id, args.timeout)
    if args.endpoint == "agent":
        path = f"/agent-chat/{session_id}"
        payload: dict[str, Any] = {
            "query": query,
            "agent_enabled": True,
            "knowledge_base_ids": selected_kb_ids(args),
            "channel": "api",
            "mentioned_items": mentioned_items(args),
        }
        if args.agent_id:
            payload["agent_id"] = args.agent_id
    else:
        path = f"/knowledge-chat/{session_id}"
        payload = {
            "query": query,
            "knowledge_base_ids": selected_kb_ids(args),
            "disable_title": True,
            "channel": "api",
        }
        if args.agent_id:
            payload["agent_id"] = args.agent_id

    req = request.Request(api_url(args.host, path), data=json.dumps(payload, ensure_ascii=False).encode("utf-8"), method="POST")
    req.add_header("Content-Type", "application/json")
    req.add_header("Accept", "text/event-stream")
    for key, value in headers.items():
        req.add_header(key, value)

    answer_parts: list[str] = []
    events: list[dict[str, Any]] = []
    references: list[dict[str, Any]] = []
    done = False
    started = time.monotonic()

    try:
        with request.urlopen(req, timeout=args.timeout) as resp:
            for raw_line in resp:
                line = raw_line.decode("utf-8", errors="replace").strip()
                if not line.startswith("data:"):
                    continue
                data = line[5:].strip()
                if not data or data == "[DONE]":
                    continue
                try:
                    event = json.loads(data)
                except json.JSONDecodeError:
                    events.append({"response_type": "decode_error", "content": data})
                    continue
                events.append(event)
                if event.get("response_type") == "answer" and event.get("content"):
                    answer_parts.append(event["content"])
                if event.get("knowledge_references"):
                    references.extend(event["knowledge_references"])
                if event.get("done"):
                    done = True
    except error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {exc.code} from chat endpoint: {detail}") from exc

    return {
        "session_id": session_id,
        "answer": "".join(answer_parts),
        "references": references,
        "events": events,
        "done": done,
        "elapsed_seconds": round(time.monotonic() - started, 2),
    }


def keyword_variants(keyword: str) -> list[str]:
    variants = KEYWORD_SYNONYMS.get(keyword, [])
    return [keyword, *variants]


def contains_any(text: str, keywords: list[str]) -> bool:
    haystack = text.lower()
    return any(keyword.lower() in haystack for keyword in keywords)


def contains_all_groups(text: str, keywords: list[str]) -> tuple[bool, list[dict[str, Any]]]:
    groups = []
    for keyword in keywords:
        variants = keyword_variants(keyword)
        hit = contains_any(text, variants)
        groups.append({"keyword": keyword, "hit": hit, "variants": variants})
    return all(group["hit"] for group in groups), groups


def judge(test: dict[str, Any], answer: str, references: list[dict[str, Any]], done: bool) -> dict[str, Any]:
    reference_text = json.dumps(references, ensure_ascii=False)
    evidence_text = f"{answer}\n{reference_text}"
    point_results = []
    for point in test.get("expected_points", []):
        hit, keyword_groups = contains_all_groups(evidence_text, point.get("keywords", []))
        point_results.append({"name": point["name"], "hit": hit, "keywords": point.get("keywords", []), "keyword_groups": keyword_groups})
    section_results = [{"name": section, "hit": section.lower() in answer.lower()} for section in test.get("required_sections", [])]
    citation_hits = [term for term in test.get("citation_terms", []) if term.lower() in evidence_text.lower()]
    point_hits = sum(1 for item in point_results if item["hit"])
    section_hits = sum(1 for item in section_results if item["hit"])
    min_points = test.get("min_expected_points")
    if min_points is None:
        min_points = 3 if test["priority"] == "P0" else min(3, len(point_results))
    min_references = test.get("min_references", 1)
    min_citation_terms = test.get("min_citation_terms", 1)
    references_ok = len(references) >= min_references
    citations_ok = len(citation_hits) >= min_citation_terms
    sections_ok = section_hits >= max(1, len(section_results) - 1)
    passed = done and point_hits >= min_points and sections_ok and citations_ok and references_ok
    return {
        "passed": passed,
        "done": done,
        "point_hits": point_hits,
        "point_total": len(point_results),
        "min_expected_points": min_points,
        "section_hits": section_hits,
        "section_total": len(section_results),
        "citation_hits": citation_hits,
        "min_citation_terms": min_citation_terms,
        "reference_count": len(references),
        "min_references": min_references,
        "references_ok": references_ok,
        "citations_ok": citations_ok,
        "sections_ok": sections_ok,
        "point_results": point_results,
        "section_results": section_results,
    }


def write_outputs(output_dir: Path, results: list[dict[str, Any]], args: argparse.Namespace) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    summary = {
        "generated_at": dt.datetime.now().isoformat(timespec="seconds"),
        "host": args.host,
        "endpoint": args.endpoint,
        "agent_id": args.agent_id,
        "case_kb_id": args.case_kb_id,
        "law_kb_id": args.law_kb_id,
        "total": len(results),
        "passed": sum(1 for r in results if r["judge"]["passed"]),
    }
    summary["case_qa_usability"] = round(summary["passed"] / summary["total"], 4) if summary["total"] else 0
    with (output_dir / "results.json").open("w", encoding="utf-8") as f:
        json.dump({"summary": summary, "results": results}, f, ensure_ascii=False, indent=2)

    lines = [
        "# 裁判案例问答自动测试结果",
        "",
        f"- 生成时间：{summary['generated_at']}",
        f"- 服务地址：{summary['host']}",
        f"- 接口模式：{summary['endpoint']}",
        f"- 裁判案例问答可用率：{summary['passed']} / {summary['total']} = {summary['case_qa_usability']:.2%}",
        "",
        "| 用例 | 结果 | 要点命中 | 章节命中 | 引用词 | 引用数 | 耗时 |",
        "| --- | --- | ---: | ---: | ---: | ---: | ---: |",
    ]
    for result in results:
        j = result["judge"]
        status = "PASS" if j["passed"] else "REVIEW"
        lines.append(
            f"| {result['id']} | {status} | {j['point_hits']}/{j['point_total']} | "
            f"{j['section_hits']}/{j['section_total']} | {len(j['citation_hits'])}/{j['min_citation_terms']} | "
            f"{j['reference_count']} | {result['elapsed_seconds']}s |"
        )
    lines.extend([
        "",
        "## 复核提示",
        "",
        "- `PASS` 代表机器检查命中核心事实、依据、章节和引用要求，不代表正式法律意见质量通过。",
        "- `REVIEW` 需要人工打开 `results.json` 查看完整回答、引用和未命中的检查项。",
        "- 本脚本会同时检查答案文本和 `knowledge_references`，但仍需人工确认引用是否确实落在目标案例片段。",
    ])
    with (output_dir / "summary.md").open("w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def main() -> int:
    args = parse_args()
    tests = load_cases(args.cases, args.only)
    if args.dry_run:
        for test in tests:
            print(f"{test['id']}: {test['document']} -> {args.endpoint}")
        return 0

    headers = auth_headers(args)
    output_dir = args.output_dir
    if output_dir is None:
        stamp = dt.datetime.now().strftime("%Y%m%d-%H%M%S")
        output_dir = DEFAULT_OUTPUT_ROOT / stamp

    results: list[dict[str, Any]] = []
    for index, test in enumerate(tests, start=1):
        print(f"[{index}/{len(tests)}] running {test['id']}...", flush=True)
        try:
            stream_result = chat_stream(args, headers, test["id"], build_query(test, bool(args.law_kb_id)))
            judgment = judge(test, stream_result["answer"], stream_result["references"], stream_result["done"])
            results.append({
                "id": test["id"],
                "priority": test["priority"],
                "document": test["document"],
                "question": test["question"],
                "answer": stream_result["answer"],
                "references": stream_result["references"],
                "events": stream_result["events"],
                "session_id": stream_result["session_id"],
                "elapsed_seconds": stream_result["elapsed_seconds"],
                "judge": judgment,
            })
            print(f"  -> {'PASS' if judgment['passed'] else 'REVIEW'}", flush=True)
        except Exception as exc:  # noqa: BLE001 - CLI should record failures and continue.
            results.append({
                "id": test["id"],
                "priority": test["priority"],
                "document": test["document"],
                "question": test["question"],
                "error": str(exc),
                "elapsed_seconds": 0,
                "judge": {
                    "passed": False,
                    "done": False,
                    "point_hits": 0,
                    "point_total": 0,
                    "min_expected_points": test.get("min_expected_points", 0),
                    "section_hits": 0,
                    "section_total": 0,
                    "citation_hits": [],
                    "min_citation_terms": test.get("min_citation_terms", 1),
                    "reference_count": 0,
                    "min_references": test.get("min_references", 1),
                },
            })
            print(f"  -> ERROR: {exc}", flush=True)

    write_outputs(output_dir, results, args)
    print(f"wrote {output_dir / 'summary.md'}")
    print(f"wrote {output_dir / 'results.json'}")
    return 0 if all(r["judge"]["passed"] for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
