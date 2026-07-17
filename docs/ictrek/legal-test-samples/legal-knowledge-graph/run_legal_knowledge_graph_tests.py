#!/usr/bin/env python3
"""Run legal-knowledge-graph sample tests against a LexAI/WeKnora server."""

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
    "劳动者单方解除": ["劳动者单方解除", "劳动者解除", "劳动者可以解除"],
    "用人单位单方解除": ["用人单位单方解除", "用人单位解除", "单位解除"],
    "违法解除": ["违法解除", "违法终止", "非法解除"],
    "赔偿金": ["赔偿金", "二倍经济补偿", "赔偿责任"],
    "个人信息处理者": ["个人信息处理者", "处理者"],
    "处理行为": ["处理行为", "处理个人信息", "收集", "使用", "加工", "传输", "提供", "公开"],
    "合法性基础": ["合法性基础", "处理规则", "合法事由", "合法依据"],
    "查阅复制": ["查阅复制", "查阅", "复制"],
    "更正删除": ["更正删除", "更正", "删除"],
    "投诉举报": ["投诉举报", "投诉", "举报"],
    "七日无理由退货": ["七日无理由退货", "七日内退货", "无理由退货"],
    "收到商品": ["收到商品", "收货", "自收到商品之日起"],
    "退货例外": ["退货例外", "不适用无理由退货", "不宜退货"],
    "鲜活易腐": ["鲜活易腐", "鲜活", "易腐"],
    "价款支付": ["价款支付", "支付价款", "付款"],
    "违约责任": ["违约责任", "承担违约责任"],
    "继续履行": ["继续履行", "履行"],
    "赔偿损失": ["赔偿损失", "损失赔偿", "赔偿"],
    "违约金调整": ["违约金调整", "调整违约金", "请求增加", "请求减少"],
    "检验期限": ["检验期限", "检验期间", "合理期限"],
    "当事人": ["当事人", "原告", "被告", "上诉人", "被上诉人"],
    "争议焦点": ["争议焦点", "争议", "焦点"],
    "法院认定": ["法院认定", "本院认为", "法院认为", "认定"],
    "裁判结果": ["裁判结果", "判决", "裁定", "结果"],
    "适用依据": ["适用依据", "法律依据", "依据", "法条"],
    "案例观点": ["案例观点", "裁判观点", "法院观点", "裁判规则"],
    "事实要素": ["事实要素", "案件事实", "关键事实"],
    "法律原则": ["法律原则", "原则"],
    "风险触发": ["风险触发", "风险点", "风险事件", "风险状态", "触发条件"],
    "责任认定": ["责任认定", "责任主张", "法律后果", "违约责任", "质量异议"],
    "付款早于验收": ["付款早于验收", "先付款后验收", "到货前支付", "预付款"],
    "验收不合格": ["验收不合格", "验收未通过", "质量不合格", "不符合约定"],
    "价款追回风险": ["价款追回风险", "资金损失", "资金占用", "无法退货", "追索难度", "款项处于高风险", "资金控制权"],
    "履行不符合约定": ["履行不符合约定", "不符合质量要求", "设备不合格", "质量缺陷", "质量要求", "不符合约定"],
    "合同风险": ["合同风险", "风险"],
    "举证责任": ["举证责任", "举证", "证明责任", "证明"],
    "解除合同": ["解除合同", "合同解除", "解除"],
    "维修更换退货": ["维修更换退货", "维修", "更换", "退货"],
}

RELATION_SYNONYMS = {
    "包括": ["包括", "包含", "分为", "类型"],
    "可能产生": ["可能产生", "可能导致", "产生", "触发"],
    "导致": ["导致", "引发", "构成", "承担"],
    "实施": ["实施", "进行", "处理"],
    "需要": ["需要", "应当具备", "以", "基于"],
    "享有": ["享有", "有权", "权利"],
    "可以请求": ["可以请求", "有权请求", "可要求", "主张"],
    "承担": ["承担", "负有", "履行"],
    "通过网络购买": ["通过网络购买", "网络购买", "网购"],
    "有权": ["有权", "可以", "享有"],
    "起算于": ["起算于", "自", "从"],
    "属于": ["属于", "列为", "是"],
    "应当": ["应当", "应", "需要"],
    "负有": ["负有", "承担", "应当履行"],
    "围绕": ["围绕", "涉及", "争议"],
    "认定": ["认定", "认为", "查明"],
    "支持": ["支持", "支撑", "据此"],
    "依据": ["依据", "适用", "引用"],
    "对应": ["对应", "关联", "映射"],
    "基于": ["基于", "根据", "依托"],
    "触发": ["触发", "对应", "适用"],
    "映射到": ["映射到", "对应", "关联"],
    "解释": ["解释", "支持", "说明"],
    "增加": ["增加", "提高", "加重", "面临", "处于", "失去", "引发", "导致"],
    "可能构成": ["可能构成", "可能属于", "构成"],
    "需要证明": ["需要证明", "需证明", "必须证明", "承担举证", "举证证明", "提供检验报告", "提供证据"],
    "可以主张": ["可以主张", "可主张", "请求", "要求"],
}

STRUCTURE_SIGNALS = [
    r"```mermaid",
    r"\bgraph\s+(TD|LR|BT|RL)\b",
    r"\|[^|\n]+\|[^|\n]+\|",
    r"节点",
    r"关系",
    r"边表",
    r"edges?",
    r"nodes?",
    r"source",
    r"target",
    r"->",
    r"-->",
    r"-\s*主体",
    r"-\s*行为",
]

CITATION_SECTION_TERMS = ["依据", "来源", "法条", "案例", "引用", "裁判"]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run the legal-knowledge-graph sample suite and write JSON/Markdown results."
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
    parser.add_argument("--agent-id", default=os.getenv("WEKNORA_LEGAL_KG_AGENT_ID"), help="Optional legal KG or legal assistant agent ID.")
    parser.add_argument(
        "--endpoint",
        choices=("knowledge", "agent"),
        default=os.getenv("LEGAL_KG_ENDPOINT", "knowledge"),
        help="Use knowledge-chat or agent-chat.",
    )
    parser.add_argument("--cases", type=Path, default=DEFAULT_CASES)
    parser.add_argument("--output-dir", type=Path, default=None)
    parser.add_argument("--only", action="append", default=[], help="Run only matching test IDs. Can be passed more than once.")
    parser.add_argument("--timeout", type=int, default=int(os.getenv("LEGAL_KG_TIMEOUT", "240")))
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
        {"title": f"legal-kg-test {test_id}", "description": "Automated legal knowledge graph sample test"},
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


def build_query(test: dict[str, Any]) -> str:
    kb_labels = "、".join(KB_ROLE_LABELS.get(item, item) for item in test.get("knowledge_base_ids", []))
    return f"""你是法律知识图谱测试执行助手。请只基于已选择的知识库回答，不要编造法条、案例、案号或裁判观点。

图谱类型：{test["graph_type"]}
应使用知识库：{kb_labels}
用户问题：
{test["query"]}

输出要求：
1. 必须输出结构化图谱，不要只写普通段落解释。可以使用 Markdown 表格、Mermaid graph、缩进列表或 JSON-like 节点/边列表。
2. 至少包含“节点”“关系”“依据/来源”三部分。
3. 关系请尽量表达为“源节点 - 关系 - 目标节点”，并覆盖主体、行为、条件、责任、救济或案例要素。
4. 如果使用 Mermaid，请输出前端可直接渲染的安全语法：节点标签使用双引号，例如 N1["合同约定：到货前支付90%"]；不要在节点标签中使用 <b>、<br/>、HTML 标签或 Markdown 加粗；边标签使用 A -->|关系| B，不要在竖线内外加入多余空格。
5. 对法条和案例要分开标注来源；如果未检索到案例或法条依据，请明确写“现有知识库未检索到可用依据”。
6. 不要输出确定性个案胜诉结论。
"""


def chat_stream(args: argparse.Namespace, headers: dict[str, str], test: dict[str, Any], query: str) -> dict[str, Any]:
    session_id = create_session(args.host, headers, test["id"], args.timeout)
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
        "session_id": session_id,
        "answer": "".join(answer_parts),
        "references": references,
        "events": events,
        "done": done,
        "elapsed_seconds": round(time.monotonic() - started, 2),
    }


def keyword_variants(keyword: str, synonym_map: dict[str, list[str]]) -> list[str]:
    variants = synonym_map.get(keyword, [])
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
        if contains_any(text, keyword_variants(term, KEYWORD_SYNONYMS)):
            hits.append(term)
            seen.add(term)
    return hits


def relation_hit(text: str, edge: dict[str, str]) -> dict[str, Any]:
    source = edge.get("source", "")
    relation = edge.get("relation", "")
    target = edge.get("target", "")
    source_hit = contains_any(text, keyword_variants(source, KEYWORD_SYNONYMS))
    relation_hit_value = contains_any(text, keyword_variants(relation, RELATION_SYNONYMS))
    target_hit = contains_any(text, keyword_variants(target, KEYWORD_SYNONYMS))
    hit = source_hit and target_hit and relation_hit_value
    return {
        "edge": edge,
        "hit": hit,
        "source_hit": source_hit,
        "relation_hit": relation_hit_value,
        "target_hit": target_hit,
    }


def has_structured_graph_output(text: str) -> bool:
    signal_hits = 0
    for pattern in STRUCTURE_SIGNALS:
        if re.search(pattern, text, flags=re.IGNORECASE):
            signal_hits += 1
    relation_lines = len(re.findall(r".+(-|->|-->|→).+(-|->|-->|→).+", text))
    numbered_or_bulleted = len(re.findall(r"^\s*(?:[-*]|\d+\.)\s+", text, flags=re.MULTILINE))
    return signal_hits >= 2 or relation_lines >= 2 or (signal_hits >= 1 and numbered_or_bulleted >= 4)


def forbidden_matches(text: str, patterns: list[str]) -> list[dict[str, str]]:
    matches: list[dict[str, str]] = []
    for pattern in patterns:
        try:
            compiled = re.compile(pattern, re.IGNORECASE | re.DOTALL)
        except re.error as exc:
            matches.append({"pattern": pattern, "match": f"INVALID REGEX: {exc}"})
            continue
        for match in compiled.finditer(text):
            matches.append({"pattern": pattern, "match": match.group(0)[:160]})
    return matches


def judge(test: dict[str, Any], answer: str, references: list[dict[str, Any]], done: bool) -> dict[str, Any]:
    node_hits = term_hits(answer, test.get("expected_nodes", []))
    edge_results = [relation_hit(answer, edge) for edge in test.get("expected_edges", [])]
    edge_hits = [item for item in edge_results if item["hit"]]
    section_hits = term_hits(answer, test.get("expected_sections", []))
    citation_hits = term_hits(answer, [*CITATION_SECTION_TERMS, *test.get("citation_terms", [])])
    forbidden = forbidden_matches(answer, test.get("forbidden_patterns", []))

    node_total = len(test.get("expected_nodes", []))
    edge_total = len(test.get("expected_edges", []))
    section_total = len(test.get("expected_sections", []))
    node_ratio = len(node_hits) / node_total if node_total else 0
    edge_ratio = len(edge_hits) / edge_total if edge_total else 0
    section_ratio = len(section_hits) / section_total if section_total else 0
    structured_ok = has_structured_graph_output(answer)
    evidence_ok = len(references) >= 1 or len(citation_hits) >= 2
    forbidden_ok = len(forbidden) == 0
    answered = bool(answer.strip())

    if done and answered and structured_ok and evidence_ok and forbidden_ok and node_ratio >= 0.6 and edge_ratio >= 0.5:
        status = "PASS"
    elif not answered or not done or not forbidden_ok or (not structured_ok and edge_ratio < 0.35):
        status = "FAIL"
    elif node_ratio >= 0.35 or edge_ratio >= 0.3 or evidence_ok:
        status = "REVIEW"
    else:
        status = "FAIL"

    return {
        "status": status,
        "passed": status == "PASS",
        "done": done,
        "answered": answered,
        "structured_ok": structured_ok,
        "evidence_ok": evidence_ok,
        "forbidden_ok": forbidden_ok,
        "node_hits": node_hits,
        "node_total": node_total,
        "node_ratio": round(node_ratio, 4),
        "edge_hits": len(edge_hits),
        "edge_total": edge_total,
        "edge_ratio": round(edge_ratio, 4),
        "section_hits": section_hits,
        "section_total": section_total,
        "section_ratio": round(section_ratio, 4),
        "citation_hits": citation_hits,
        "reference_count": len(references),
        "edge_results": edge_results,
        "forbidden_matches": forbidden,
        "pass_criteria": test.get("pass_criteria"),
    }


def reference_title(reference: dict[str, Any]) -> str:
    for key in ("document_name", "knowledge_title", "title", "name"):
        if reference.get(key):
            return str(reference[key])
    return "(untitled reference)"


def write_case_response(output_dir: Path, result: dict[str, Any]) -> None:
    case_dir = output_dir / result["id"]
    case_dir.mkdir(parents=True, exist_ok=True)
    judge_result = result["judge"]
    lines = [
        f"# {result['id']} {result['title']}",
        "",
        f"- 结果：{judge_result.get('status', 'FAIL')}",
        f"- 图谱类型：{result.get('graph_type', '')}",
        f"- 耗时：{result.get('elapsed_seconds', 0)}s",
        f"- 会话 ID：{result.get('session_id', '')}",
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
        f"- 图谱结构：{'Y' if judge_result.get('structured_ok') else 'N'}",
        f"- 依据说明：{'Y' if judge_result.get('evidence_ok') else 'N'}",
        f"- 节点命中：{len(judge_result.get('node_hits', []))}/{judge_result.get('node_total', 0)}",
        f"- 关系命中：{judge_result.get('edge_hits', 0)}/{judge_result.get('edge_total', 0)}",
        f"- 章节命中：{len(judge_result.get('section_hits', []))}/{judge_result.get('section_total', 0)}",
        f"- 引用术语：{', '.join(judge_result.get('citation_hits', [])) or '未命中'}",
        f"- 禁止内容：{len(judge_result.get('forbidden_matches', []))} 处",
        f"- 通过标准：{judge_result.get('pass_criteria', '')}",
    ]
    if judge_result.get("node_hits"):
        lines.extend(["", "## 命中节点", ""])
        for item in judge_result["node_hits"]:
            lines.append(f"- {item}")
    if judge_result.get("edge_results"):
        lines.extend(["", "## 关系检查", ""])
        for item in judge_result["edge_results"]:
            edge = item["edge"]
            mark = "Y" if item["hit"] else "N"
            lines.append(f"- {mark} {edge.get('source')} - {edge.get('relation')} - {edge.get('target')}")
    if judge_result.get("forbidden_matches"):
        lines.extend(["", "## 禁止内容命中", ""])
        for item in judge_result["forbidden_matches"]:
            lines.append(f"- `{item['pattern']}` -> {item['match']}")
    if result.get("references"):
        lines.extend(["", "## 引用摘要", ""])
        for reference in result["references"][:8]:
            lines.append(f"- {reference_title(reference)}")
    (case_dir / "response.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_outputs(output_dir: Path, results: list[dict[str, Any]], args: argparse.Namespace) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    for result in results:
        write_case_response(output_dir, result)

    summary = {
        "generated_at": dt.datetime.now().isoformat(timespec="seconds"),
        "host": args.host,
        "endpoint": args.endpoint,
        "agent_id": args.agent_id,
        "law_kb_id": args.law_kb_id,
        "case_kb_id": args.case_kb_id,
        "total": len(results),
        "passed": sum(1 for r in results if r["judge"]["status"] == "PASS"),
        "review": sum(1 for r in results if r["judge"]["status"] == "REVIEW"),
        "failed": sum(1 for r in results if r["judge"]["status"] == "FAIL"),
    }
    summary["legal_knowledge_graph_rate"] = round(summary["passed"] / summary["total"], 4) if summary["total"] else 0
    with (output_dir / "results.json").open("w", encoding="utf-8") as f:
        json.dump({"summary": summary, "results": results}, f, ensure_ascii=False, indent=2)

    lines = [
        "# 法律知识图谱自动测试结果",
        "",
        f"- 生成时间：{summary['generated_at']}",
        f"- 服务地址：{summary['host']}",
        f"- 接口模式：{summary['endpoint']}",
        f"- 法律条文库：{summary['law_kb_id']}",
        f"- 法律案例库：{summary['case_kb_id']}",
        f"- 法律知识图谱可用率：{summary['passed']} / {summary['total']} = {summary['legal_knowledge_graph_rate']:.2%}",
        "",
        "| 用例 | 结果 | 图谱结构 | 节点命中 | 关系命中 | 依据 | 引用数 | 耗时 |",
        "| --- | --- | --- | ---: | ---: | --- | ---: | ---: |",
    ]
    for result in results:
        j = result["judge"]
        lines.append(
            f"| {result['id']} | {j.get('status', 'FAIL')} | {'Y' if j.get('structured_ok') else 'N'} | "
            f"{len(j.get('node_hits', []))}/{j.get('node_total', 0)} | {j.get('edge_hits', 0)}/{j.get('edge_total', 0)} | "
            f"{'Y' if j.get('evidence_ok') else 'N'} | {j.get('reference_count', 0)} | {result.get('elapsed_seconds', 0)}s |"
        )
    lines.extend([
        "",
        "## 复核提示",
        "",
        "- `PASS` 代表机器检查命中核心节点、核心关系、结构化图谱和依据要求。",
        "- `REVIEW` 代表有部分节点/关系或引用命中，但结构、依据或关系完整度不足。",
        "- `FAIL` 代表回答未完成、缺少图谱结构、核心关系不足或命中禁止内容模式。",
        "- 自动判定只作为初筛，不代表正式法律意见质量背书。",
    ])
    with (output_dir / "summary.md").open("w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def main() -> int:
    args = parse_args()
    tests = load_cases(args.cases, args.only)
    if args.dry_run:
        for test in tests:
            kb_ids = ", ".join(resolve_kb_ids(test, args))
            print(f"{test['id']}: {test['title']} -> {args.endpoint} [{kb_ids}]")
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
            stream_result = chat_stream(args, headers, test, build_query(test))
            judgment = judge(test, stream_result["answer"], stream_result["references"], stream_result["done"])
            results.append({
                "id": test["id"],
                "title": test["title"],
                "priority": test.get("priority"),
                "graph_type": test.get("graph_type"),
                "query": test["query"],
                "knowledge_base_ids": resolve_kb_ids(test, args),
                "answer": stream_result["answer"],
                "references": stream_result["references"],
                "events": stream_result["events"],
                "session_id": stream_result["session_id"],
                "elapsed_seconds": stream_result["elapsed_seconds"],
                "judge": judgment,
            })
            print(f"  -> {judgment['status']}", flush=True)
        except Exception as exc:  # noqa: BLE001 - CLI should record failures and continue.
            results.append({
                "id": test["id"],
                "title": test["title"],
                "priority": test.get("priority"),
                "graph_type": test.get("graph_type"),
                "query": test["query"],
                "knowledge_base_ids": resolve_kb_ids(test, args),
                "error": str(exc),
                "elapsed_seconds": 0,
                "judge": {
                    "status": "FAIL",
                    "passed": False,
                    "done": False,
                    "answered": False,
                    "structured_ok": False,
                    "evidence_ok": False,
                    "forbidden_ok": False,
                    "node_hits": [],
                    "node_total": len(test.get("expected_nodes", [])),
                    "node_ratio": 0,
                    "edge_hits": 0,
                    "edge_total": len(test.get("expected_edges", [])),
                    "edge_ratio": 0,
                    "section_hits": [],
                    "section_total": len(test.get("expected_sections", [])),
                    "section_ratio": 0,
                    "citation_hits": [],
                    "reference_count": 0,
                    "edge_results": [],
                    "forbidden_matches": [],
                    "pass_criteria": test.get("pass_criteria"),
                },
            })
            print(f"  -> ERROR: {exc}", flush=True)

    write_outputs(output_dir, results, args)
    print(f"wrote {output_dir / 'summary.md'}")
    print(f"wrote {output_dir / 'results.json'}")
    return 0 if all(r["judge"]["status"] == "PASS" for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
