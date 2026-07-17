#!/usr/bin/env python3
"""Run legal-statute QA sample tests against a LexAI/WeKnora server."""

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
    "过低": ["过低", "低于", "不足", "偏低"],
    "过高": ["过高", "过分高于", "高于", "偏高"],
    "请求减少": ["请求减少", "请求适当减少", "予以减少", "适当减少"],
    "请求增加": ["请求增加", "请求予以增加", "予以增加", "增加"],
    "获得的利益": ["获得的利益", "可得利益", "履行后可以获得的利益"],
    "无需说明理由": ["无需说明理由", "无须说明理由", "无理由"],
    "解释说明": ["解释说明", "解释", "说明"],
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run the legal statute QA sample suite and write JSON/Markdown results."
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
    parser.add_argument("--law-kb-id", default=os.getenv("LEGAL_LAW_KB_ID"), required=os.getenv("LEGAL_LAW_KB_ID") is None)
    parser.add_argument("--agent-id", default=os.getenv("WEKNORA_LEGAL_QA_AGENT_ID"), help="Optional quick-answer agent ID.")
    parser.add_argument(
        "--endpoint",
        choices=("knowledge", "agent"),
        default=os.getenv("LEGAL_QA_ENDPOINT", "knowledge"),
        help="Use knowledge-chat or agent-chat.",
    )
    parser.add_argument("--cases", type=Path, default=DEFAULT_CASES)
    parser.add_argument("--output-dir", type=Path, default=None)
    parser.add_argument("--only", action="append", default=[], help="Run only matching test IDs. Can be passed more than once.")
    parser.add_argument("--timeout", type=int, default=int(os.getenv("LEGAL_QA_TIMEOUT", "180")))
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
        {"title": f"legal-qa-test {test_id}", "description": "Automated legal statute QA sample test"},
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


def build_query(test: dict[str, Any]) -> str:
    return f"""你是法律法规问答测试执行助手。请只基于已选择的法律条文知识库回答。

目标法规文档：{test["document"]}
测试问题：{test["question"]}

输出要求：
1. 先给出简明结论。
2. 引用具体法律名称和条文编号。
3. 必要时说明适用条件、例外或限制。
4. 如果知识库没有检索到依据，请明确写“现有知识库未检索到可用依据”，不要编造法条。
"""


def chat_stream(args: argparse.Namespace, headers: dict[str, str], test_id: str, query: str) -> dict[str, Any]:
    session_id = create_session(args.host, headers, test_id, args.timeout)
    if args.endpoint == "agent":
        path = f"/agent-chat/{session_id}"
        payload: dict[str, Any] = {
            "query": query,
            "agent_enabled": True,
            "knowledge_base_ids": [args.law_kb_id],
            "channel": "api",
            "mentioned_items": [
                {"id": args.law_kb_id, "name": "法律条文", "type": "kb", "kb_type": "document"},
            ],
        }
        if args.agent_id:
            payload["agent_id"] = args.agent_id
    else:
        path = f"/knowledge-chat/{session_id}"
        payload = {
            "query": query,
            "knowledge_base_ids": [args.law_kb_id],
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
    point_results = []
    for point in test.get("expected_points", []):
        hit, keyword_groups = contains_all_groups(answer, point.get("keywords", []))
        point_results.append({"name": point["name"], "hit": hit, "keywords": point.get("keywords", []), "keyword_groups": keyword_groups})
    section_results = [{"name": section, "hit": section.lower() in answer.lower()} for section in test.get("required_sections", [])]
    citation_hits = [term for term in test.get("citation_terms", []) if term.lower() in answer.lower()]
    point_hits = sum(1 for item in point_results if item["hit"])
    section_hits = sum(1 for item in section_results if item["hit"])
    min_points = 2 if test["priority"] == "P0" else min(3, len(point_results))
    evidence_ok = len(citation_hits) >= 1 or len(references) >= 1
    passed = done and point_hits >= min_points and section_hits >= max(1, len(section_results) - 1) and evidence_ok
    return {
        "passed": passed,
        "done": done,
        "point_hits": point_hits,
        "point_total": len(point_results),
        "section_hits": section_hits,
        "section_total": len(section_results),
        "citation_hits": citation_hits,
        "reference_count": len(references),
        "evidence_ok": evidence_ok,
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
        "law_kb_id": args.law_kb_id,
        "total": len(results),
        "passed": sum(1 for r in results if r["judge"]["passed"]),
    }
    summary["legal_qa_usability"] = round(summary["passed"] / summary["total"], 4) if summary["total"] else 0
    with (output_dir / "results.json").open("w", encoding="utf-8") as f:
        json.dump({"summary": summary, "results": results}, f, ensure_ascii=False, indent=2)

    lines = [
        "# 法律法规问答自动测试结果",
        "",
        f"- 生成时间：{summary['generated_at']}",
        f"- 服务地址：{summary['host']}",
        f"- 接口模式：{summary['endpoint']}",
        f"- 法律法规问答可用率：{summary['passed']} / {summary['total']} = {summary['legal_qa_usability']:.2%}",
        "",
        "| 用例 | 结果 | 要点命中 | 章节命中 | 引用数 | 耗时 |",
        "| --- | --- | ---: | ---: | ---: | ---: |",
    ]
    for result in results:
        j = result["judge"]
        status = "PASS" if j["passed"] else "REVIEW"
        lines.append(
            f"| {result['id']} | {status} | {j['point_hits']}/{j['point_total']} | "
            f"{j['section_hits']}/{j['section_total']} | {j['reference_count']} | {result['elapsed_seconds']}s |"
        )
    lines.extend([
        "",
        "## 复核提示",
        "",
        "- `PASS` 代表机器检查命中核心要点、章节和证据要求，不代表正式法律意见质量通过。",
        "- `REVIEW` 需要人工打开 `results.json` 查看完整回答、引用和未命中的检查项。",
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
            stream_result = chat_stream(args, headers, test["id"], build_query(test))
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
                "judge": {"passed": False, "done": False, "point_hits": 0, "point_total": 0, "section_hits": 0, "section_total": 0, "citation_hits": [], "reference_count": 0},
            })
            print(f"  -> ERROR: {exc}", flush=True)

    write_outputs(output_dir, results, args)
    print(f"wrote {output_dir / 'summary.md'}")
    print(f"wrote {output_dir / 'results.json'}")
    return 0 if all(r["judge"]["passed"] for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
