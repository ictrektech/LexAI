#!/usr/bin/env python3
"""Run multi-turn legal follow-up sample tests against a LexAI/WeKnora server."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import sys
import time
from pathlib import Path
from typing import Any
from urllib import error, request


ROOT = Path(__file__).resolve().parent
DEFAULT_CASES = ROOT / "test-cases.json"
DEFAULT_OUTPUT_ROOT = ROOT / "results"
DEFAULT_LAW_KB_ID = "f07af6bb-2645-428a-8db2-829708e3a2c2"
DEFAULT_CASE_KB_ID = "4ca9a808-83f5-4222-8cc4-424ae24f6656"

KB_ROLE_LABELS = {
    "law": "法律条文",
    "case": "法律案例",
}

KEYWORD_SYNONYMS = {
    "入职 5 个月": ["入职 5 个月", "入职5个月", "五个月", "5个月"],
    "书面考核": ["书面考核", "考核记录", "书面记录", "考核证据"],
    "违法解除风险": ["违法解除风险", "违法解除", "解除违法", "赔偿金风险"],
    "证据不足": ["证据不足", "缺少证据", "依据不足", "无法证明"],
    "不能直接解除": ["不能直接解除", "不能当然解除", "不得直接解除", "不能随意解除"],
    "风险下降": ["风险下降", "风险降低", "合法性增强", "解除依据更充分"],
    "不当然影响解除": ["不当然影响解除", "不必然影响解除", "不当然阻止解除", "不当然导致解除无效"],
    "二倍工资": ["二倍工资", "双倍工资", "二倍工资差额"],
    "退货退款": ["退货退款", "退货", "退款"],
    "第 3 天": ["第 3 天", "第3天", "三天", "3 天", "3天"],
    "七日无理由退货": ["七日无理由退货", "七日内退货", "无理由退货"],
    "定制": ["定制", "定作", "按尺寸"],
    "不当然适用七日无理由": ["不当然适用七日无理由", "不适用七日无理由", "不能当然七日无理由"],
    "迟延交付": ["迟延交付", "延期交付", "逾期交付", "迟延履行"],
    "赔偿损失": ["赔偿损失", "损失赔偿", "赔偿"],
    "违约金过低": ["违约金过低", "约定过低", "低于损失", "明显低于"],
    "请求增加": ["请求增加", "请求调整", "请求提高", "适当增加"],
    "验收标准不清": ["验收标准不清", "验收标准不明确", "未约定验收标准", "没有写清验收标准"],
    "合格交付": ["合格交付", "符合约定的交付", "交付是否合格", "符合验收"],
    "书面催告": ["书面催告", "催告函", "邮件催告", "催告"],
    "告知同意": ["告知同意", "告知并同意", "取得同意"],
    "必要性": ["必要性", "必要范围", "最小必要", "限于实现处理目的"],
    "同意不是唯一条件": ["同意不是唯一条件", "并非取得同意即合法", "同意不等于当然合法"],
    "合法正当必要": ["合法正当必要", "合法、正当、必要", "合法 正当 必要"],
    "超出必要范围": ["超出必要范围", "超出最小必要", "不必要", "过度收集"],
    "拒绝基本服务": ["拒绝基本服务", "拒绝提供服务", "不得拒绝提供服务"],
    "单独同意": ["单独同意", "取得个人的单独同意", "单独授权"],
    "特定目的": ["特定目的", "特定的目的", "充分必要"],
    "案件事实": ["案件事实", "基本事实", "事实"],
    "争议焦点": ["争议焦点", "争议", "焦点"],
    "裁判结果": ["裁判结果", "判决结果", "裁判结论"],
    "裁判理由": ["裁判理由", "法院认为", "本院认为", "理由"],
    "事实认定": ["事实认定", "事实查明", "认定事实"],
    "法律适用": ["法律适用", "适用法律", "依据"],
    "事实要素": ["事实要素", "关键事实", "事实差异"],
    "差异": ["差异", "不同", "区别", "比较"],
}

NEGATING_CONTEXT_TERMS = ["不能", "无法", "不应", "不可", "不要", "避免", "并非", "不是", "不等于", "未"]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run the multi-turn legal follow-up sample suite and write JSON/Markdown results."
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
    parser.add_argument("--law-kb-id", default=os.getenv("LEGAL_LAW_KB_ID") or DEFAULT_LAW_KB_ID)
    parser.add_argument("--case-kb-id", default=os.getenv("LEGAL_CASE_KB_ID") or DEFAULT_CASE_KB_ID)
    parser.add_argument("--agent-id", default=os.getenv("WEKNORA_LEGAL_FOLLOWUP_AGENT_ID"), help="Optional legal assistant agent ID.")
    parser.add_argument(
        "--endpoint",
        choices=("knowledge", "agent"),
        default=os.getenv("MULTI_TURN_FOLLOWUP_ENDPOINT", "knowledge"),
        help="Use knowledge-chat or agent-chat.",
    )
    parser.add_argument("--cases", type=Path, default=DEFAULT_CASES)
    parser.add_argument("--output-dir", type=Path, default=None)
    parser.add_argument("--only", action="append", default=[], help="Run only matching test IDs. Can be passed more than once.")
    parser.add_argument("--timeout", type=int, default=int(os.getenv("MULTI_TURN_FOLLOWUP_TIMEOUT", "240")))
    parser.add_argument("--dry-run", action="store_true", help="Print selected tests without calling the server.")
    return parser.parse_args()


def load_cases(path: Path, only: list[str]) -> list[dict[str, Any]]:
    with path.open("r", encoding="utf-8") as f:
        payload = json.load(f)
    tests = payload.get("tests", [])
    if only:
        wanted = set(only)
        tests = [t for t in tests if t["id"] in wanted or t.get("title") in wanted]
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
        {"title": f"multi-turn-followup-test {test_id}", "description": "Automated legal multi-turn follow-up sample test"},
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


def resolve_kb_ids(test: dict[str, Any], args: argparse.Namespace) -> list[str]:
    resolved: list[str] = []
    for item in test.get("knowledge_base_ids", []):
        if item == "law":
            resolved.append(args.law_kb_id)
        elif item == "case":
            resolved.append(args.case_kb_id)
        else:
            resolved.append(item)
    return resolved


def mentioned_items(test: dict[str, Any], args: argparse.Namespace) -> list[dict[str, str]]:
    items = []
    for item in test.get("knowledge_base_ids", []):
        if item == "law":
            items.append({"id": args.law_kb_id, "name": KB_ROLE_LABELS[item], "type": "kb", "kb_type": "document"})
        elif item == "case":
            items.append({"id": args.case_kb_id, "name": KB_ROLE_LABELS[item], "type": "kb", "kb_type": "document"})
        else:
            items.append({"id": item, "name": item, "type": "kb", "kb_type": "document"})
    return items


def build_query(test: dict[str, Any], turn: dict[str, Any]) -> str:
    kb_labels = "、".join(KB_ROLE_LABELS.get(item, item) for item in test.get("knowledge_base_ids", []))
    return f"""你是法律助手多轮追问测试执行助手。请只基于已选择的知识库和本会话上下文回答，不要编造法条、案例、案号或确定性个案结论。

场景：{test["scenario"]}
应使用知识库：{kb_labels}
当前轮次：第 {turn["turn"]} 轮
用户本轮问题：
{turn["query"]}

输出要求：
1. 先给本轮简明结论。
2. 明确说明你承接了哪些前文事实；如果是第 1 轮，请说明本轮已知事实。
3. 单独说明本轮新增事实如何改变法律判断、风险等级或处理路径。
4. 继续给出法规、案例或知识库引用依据；如果未检索到可用依据，请明确说明。
5. 不要把上一轮已否定、待确认或缺证据的事实当成已经成立。
"""


def chat_turn(
    args: argparse.Namespace,
    headers: dict[str, str],
    test: dict[str, Any],
    session_id: str,
    query: str,
) -> dict[str, Any]:
    kb_ids = resolve_kb_ids(test, args)
    if args.endpoint == "agent":
        path = f"/agent-chat/{session_id}"
        payload: dict[str, Any] = {
            "query": query,
            "agent_enabled": True,
            "knowledge_base_ids": kb_ids,
            "channel": "api",
            "mentioned_items": mentioned_items(test, args),
        }
        if args.agent_id:
            payload["agent_id"] = args.agent_id
    else:
        path = f"/knowledge-chat/{session_id}"
        payload = {
            "query": query,
            "knowledge_base_ids": kb_ids,
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


def term_hits(text: str, terms: list[str]) -> list[str]:
    hits = []
    seen = set()
    for term in terms:
        if term in seen:
            continue
        if contains_any(text, keyword_variants(term)):
            hits.append(term)
            seen.add(term)
    return hits


def forbidden_matches(text: str, patterns: list[str]) -> list[dict[str, str]]:
    matches: list[dict[str, str]] = []
    for pattern in patterns:
        try:
            compiled = re.compile(pattern, re.IGNORECASE | re.DOTALL)
        except re.error as exc:
            matches.append({"pattern": pattern, "match": f"INVALID REGEX: {exc}"})
            continue
        for match in compiled.finditer(text):
            context_start = max(0, match.start() - 24)
            context = text[context_start:match.start()]
            if any(term in context for term in NEGATING_CONTEXT_TERMS):
                continue
            matches.append({"pattern": pattern, "match": match.group(0)[:160]})
    return matches


def judge_turn(turn: dict[str, Any], answer: str, references: list[dict[str, Any]], done: bool) -> dict[str, Any]:
    carryover_hits = term_hits(answer, turn.get("expected_carryover_terms", []))
    new_point_hits = term_hits(answer, turn.get("expected_new_points", []))
    citation_hits = term_hits(answer, turn.get("citation_terms", []))
    forbidden = forbidden_matches(answer, turn.get("forbidden_patterns", []))

    carryover_total = len(turn.get("expected_carryover_terms", []))
    new_point_total = len(turn.get("expected_new_points", []))
    citation_total = len(turn.get("citation_terms", []))
    min_carryover = 1 if carryover_total else 0
    min_new_points = 1 if new_point_total <= 2 else 2
    answered = bool(answer.strip())
    carryover_ok = len(carryover_hits) >= min_carryover
    new_points_ok = len(new_point_hits) >= min_new_points
    evidence_ok = len(references) >= 1 or len(citation_hits) >= 1
    forbidden_ok = len(forbidden) == 0

    satisfied = sum([carryover_ok, new_points_ok, evidence_ok, forbidden_ok])
    if done and answered and carryover_ok and new_points_ok and evidence_ok and forbidden_ok:
        status = "PASS"
    elif not done or not answered or not forbidden_ok or satisfied <= 1:
        status = "FAIL"
    else:
        status = "REVIEW"

    return {
        "status": status,
        "passed": status == "PASS",
        "done": done,
        "answered": answered,
        "carryover_ok": carryover_ok,
        "new_points_ok": new_points_ok,
        "evidence_ok": evidence_ok,
        "forbidden_ok": forbidden_ok,
        "carryover_hits": carryover_hits,
        "carryover_total": carryover_total,
        "new_point_hits": new_point_hits,
        "new_point_total": new_point_total,
        "citation_hits": citation_hits,
        "citation_total": citation_total,
        "reference_count": len(references),
        "forbidden_matches": forbidden,
    }


def judge_case(test: dict[str, Any], turn_results: list[dict[str, Any]], session_id: str) -> dict[str, Any]:
    turn_judges = [item["judge"] for item in turn_results]
    pass_count = sum(1 for item in turn_judges if item["status"] == "PASS")
    review_count = sum(1 for item in turn_judges if item["status"] == "REVIEW")
    fail_count = sum(1 for item in turn_judges if item["status"] == "FAIL")
    total = len(turn_judges)
    session_reused = bool(session_id) and all(item.get("session_id") == session_id for item in turn_results)
    min_pass = max(1, (total * 2 + 2) // 3)

    if session_reused and pass_count >= min_pass and fail_count == 0:
        status = "PASS"
    elif not session_reused or fail_count >= max(1, total // 2):
        status = "FAIL"
    else:
        status = "REVIEW"

    return {
        "status": status,
        "passed": status == "PASS",
        "session_reused": session_reused,
        "turn_passed": pass_count,
        "turn_review": review_count,
        "turn_failed": fail_count,
        "turn_total": total,
        "pass_criteria": test.get("pass_criteria"),
    }


def reference_title(reference: dict[str, Any]) -> str:
    for key in ("document_name", "knowledge_title", "title", "name"):
        if reference.get(key):
            return str(reference[key])
    return "(untitled reference)"


def write_turn_response(case_dir: Path, result: dict[str, Any]) -> None:
    judge_result = result["judge"]
    lines = [
        f"# {result['id']} 第 {result['turn']} 轮",
        "",
        f"- 结果：{judge_result['status']}",
        f"- 会话 ID：{result.get('session_id', '')}",
        f"- 耗时：{result.get('elapsed_seconds', 0)}s",
        f"- 引用数：{judge_result.get('reference_count', 0)}",
        "",
        "## 用户问题",
        "",
        result.get("query", ""),
        "",
        "## 回答",
        "",
        result.get("answer") or result.get("error", ""),
        "",
        "## 机器判定",
        "",
        f"- 承接前文：{'Y' if judge_result.get('carryover_ok') else 'N'}；命中 {len(judge_result.get('carryover_hits', []))}/{judge_result.get('carryover_total', 0)}",
        f"- 新增判断：{'Y' if judge_result.get('new_points_ok') else 'N'}；命中 {len(judge_result.get('new_point_hits', []))}/{judge_result.get('new_point_total', 0)}",
        f"- 引用依据：{'Y' if judge_result.get('evidence_ok') else 'N'}；答案命中 {len(judge_result.get('citation_hits', []))}/{judge_result.get('citation_total', 0)}，标准引用 {judge_result.get('reference_count', 0)}",
        f"- 禁止内容：{len(judge_result.get('forbidden_matches', []))} 处",
    ]
    if judge_result.get("carryover_hits"):
        lines.append(f"- 承接命中词：{', '.join(judge_result['carryover_hits'])}")
    if judge_result.get("new_point_hits"):
        lines.append(f"- 新增判断命中词：{', '.join(judge_result['new_point_hits'])}")
    if judge_result.get("citation_hits"):
        lines.append(f"- 依据命中词：{', '.join(judge_result['citation_hits'])}")
    if judge_result.get("forbidden_matches"):
        lines.extend(["", "## 禁止内容命中", ""])
        for item in judge_result["forbidden_matches"]:
            lines.append(f"- `{item['pattern']}` -> {item['match']}")
    if result.get("references"):
        lines.extend(["", "## 引用摘要", ""])
        for reference in result["references"][:8]:
            lines.append(f"- {reference_title(reference)}")
    turn_path = case_dir / f"turn-{int(result['turn']):02d}-response.md"
    turn_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_case_outputs(output_dir: Path, result: dict[str, Any]) -> None:
    case_dir = output_dir / result["id"]
    case_dir.mkdir(parents=True, exist_ok=True)
    for turn_result in result.get("turns", []):
        write_turn_response(case_dir, turn_result)


def write_outputs(output_dir: Path, results: list[dict[str, Any]], args: argparse.Namespace) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    for result in results:
        write_case_outputs(output_dir, result)

    summary = {
        "generated_at": dt.datetime.now().isoformat(timespec="seconds"),
        "host": args.host,
        "endpoint": args.endpoint,
        "agent_id": args.agent_id,
        "law_kb_id": args.law_kb_id,
        "case_kb_id": args.case_kb_id,
        "total": len(results),
        "passed": sum(1 for r in results if r["judge"]["passed"]),
    }
    summary["multi_turn_followup_rate"] = round(summary["passed"] / summary["total"], 4) if summary["total"] else 0
    with (output_dir / "results.json").open("w", encoding="utf-8") as f:
        json.dump({"summary": summary, "results": results}, f, ensure_ascii=False, indent=2)

    lines = [
        "# 多轮追问自动测试结果",
        "",
        f"- 生成时间：{summary['generated_at']}",
        f"- 服务地址：{summary['host']}",
        f"- 接口模式：{summary['endpoint']}",
        f"- 法律条文库：{summary['law_kb_id']}",
        f"- 法律案例库：{summary['case_kb_id']}",
        f"- 多轮追问通过率：{summary['passed']} / {summary['total']} = {summary['multi_turn_followup_rate']:.2%}",
        "",
        "| 用例 | 结果 | 轮次 | PASS | REVIEW | FAIL | 复用 session_id | 总耗时 |",
        "| --- | --- | ---: | ---: | ---: | ---: | --- | ---: |",
    ]
    for result in results:
        j = result["judge"]
        elapsed = sum(turn.get("elapsed_seconds", 0) for turn in result.get("turns", []))
        lines.append(
            f"| {result['id']} | {j['status']} | {j['turn_total']} | {j['turn_passed']} | "
            f"{j['turn_review']} | {j['turn_failed']} | {'Y' if j.get('session_reused') else 'N'} | {elapsed:.2f}s |"
        )
    lines.extend([
        "",
        "## 逐轮命中摘要",
        "",
        "| 用例 | 轮次 | 结果 | 承接 | 新增判断 | 引用依据 | 引用数 |",
        "| --- | ---: | --- | --- | --- | --- | ---: |",
    ])
    for result in results:
        for turn in result.get("turns", []):
            j = turn["judge"]
            lines.append(
                f"| {result['id']} | {turn['turn']} | {j['status']} | "
                f"{'Y' if j.get('carryover_ok') else 'N'} | {'Y' if j.get('new_points_ok') else 'N'} | "
                f"{'Y' if j.get('evidence_ok') else 'N'} | {j.get('reference_count', 0)} |"
            )
    lines.extend([
        "",
        "## 复核提示",
        "",
        "- `PASS` 代表多数轮次同时命中承接、增量判断和依据说明，且同一用例复用同一个 `session_id`。",
        "- `REVIEW` 代表能回答追问，但上下文承接、增量判断或依据说明不足，需要人工打开逐轮回答复核。",
        "- `FAIL` 代表明显遗忘前文、未复用会话、无依据确定性结论或出现禁止内容。",
    ])
    with (output_dir / "summary.md").open("w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def run_case(args: argparse.Namespace, headers: dict[str, str], test: dict[str, Any]) -> dict[str, Any]:
    session_id = create_session(args.host, headers, test["id"], args.timeout)
    turn_results: list[dict[str, Any]] = []
    for turn in test.get("turns", []):
        print(f"  turn {turn['turn']}...", flush=True)
        stream_result = chat_turn(args, headers, test, session_id, build_query(test, turn))
        judgment = judge_turn(turn, stream_result["answer"], stream_result["references"], stream_result["done"])
        turn_results.append({
            "id": test["id"],
            "title": test["title"],
            "turn": turn["turn"],
            "query": turn["query"],
            "session_id": session_id,
            "answer": stream_result["answer"],
            "references": stream_result["references"],
            "events": stream_result["events"],
            "elapsed_seconds": stream_result["elapsed_seconds"],
            "judge": judgment,
        })
    return {
        "id": test["id"],
        "title": test["title"],
        "priority": test.get("priority"),
        "scenario": test.get("scenario"),
        "knowledge_base_ids": resolve_kb_ids(test, args),
        "expected_final_sections": test.get("expected_final_sections", []),
        "session_id": session_id,
        "turns": turn_results,
        "judge": judge_case(test, turn_results, session_id),
    }


def failed_case_result(args: argparse.Namespace, test: dict[str, Any], exc: Exception) -> dict[str, Any]:
    turn_total = len(test.get("turns", []))
    return {
        "id": test["id"],
        "title": test["title"],
        "priority": test.get("priority"),
        "scenario": test.get("scenario"),
        "knowledge_base_ids": resolve_kb_ids(test, args),
        "error": str(exc),
        "session_id": "",
        "turns": [],
        "judge": {
            "status": "FAIL",
            "passed": False,
            "session_reused": False,
            "turn_passed": 0,
            "turn_review": 0,
            "turn_failed": turn_total,
            "turn_total": turn_total,
            "pass_criteria": test.get("pass_criteria"),
        },
    }


def main() -> int:
    args = parse_args()
    tests = load_cases(args.cases, args.only)
    if args.dry_run:
        for test in tests:
            kb_ids = ", ".join(resolve_kb_ids(test, args))
            print(f"{test['id']}: {test['title']} -> {args.endpoint} [{kb_ids}] turns={len(test.get('turns', []))}")
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
            result = run_case(args, headers, test)
            results.append(result)
            print(f"  -> {result['judge']['status']}", flush=True)
        except Exception as exc:  # noqa: BLE001 - CLI should record failures and continue.
            result = failed_case_result(args, test, exc)
            results.append(result)
            print(f"  -> ERROR: {exc}", flush=True)

    write_outputs(output_dir, results, args)
    print(f"wrote {output_dir / 'summary.md'}")
    print(f"wrote {output_dir / 'results.json'}")
    return 0 if all(r["judge"]["passed"] for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
