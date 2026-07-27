#!/usr/bin/env python3
"""Run contract-review sample tests against a LexAI/WeKnora server."""

from __future__ import annotations

import argparse
import concurrent.futures
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
TOOL_EVIDENCE_NAMES = {"knowledge_search", "grep_chunks", "list_knowledge_chunks", "get_document_info"}
REASONING_TOOL_NAMES = {"read_skill", "grep_chunks", "list_knowledge_chunks", "get_document_info"}
KEYWORD_SYNONYMS = {
    "1 年": ["1 年", "1年", "一年", "合同终止后 1 年", "合同终止后1年"],
    "60 日": ["60 日", "60日", "六十日", "60 个工作日", "60个工作日"],
    "7 日": ["7 日", "7日", "7 个工作日", "7个工作日"],
    "不清": ["不清", "不明确", "不具体", "模糊", "笼统", "过低", "缺少明确"],
    "不一致": ["不一致", "冲突", "矛盾"],
    "不对等": ["不对等", "失衡", "显失公平", "严重失衡", "比例失衡"],
    "不承担赔偿": ["不承担赔偿", "不承担", "免责", "免除", "排除"],
    "不足": ["不足", "不充分", "不完善", "缺少", "缺乏"],
    "交付清单": ["交付清单", "交付物清单", "交付物明细", "交付物列表", "交付成果清单"],
    "为限": ["为限", "上限", "最高不超过", "不超过", "最高限额"],
    "价格": ["价格", "价款", "收费", "收费标准", "服务费"],
    "宽泛": ["宽泛", "模糊", "笼统", "不明确", "范围蔓延"],
    "导出": ["导出", "提供完整的数据", "数据迁移"],
    "文件优先级": ["文件优先级", "文件顺位", "适用顺序", "优先适用", "优先级"],
    "时限": ["时限", "时间", "期限", "24 小时", "24小时", "48 小时", "48小时"],
    "过低": ["过低", "偏低", "较低", "不足", "不充分", "无法充分"],
    "过短": ["过短", "偏短", "较短", "不足", "不合理"],
    "过早": ["过早", "前置", "提前", "早于", "比例过高", "资金风险"],
    "间接损失": ["间接损失", "预期利益", "预期利润", "商誉损失", "可得利益", "客户索赔", "第三方索赔"],
    "经营资质": ["经营资质", "经营所需资质", "商业管理资质", "合法经营资质", "合法经营", "经营条件", "经营所需", "资质"],
    "内部使用": ["内部使用", "公司内部", "非独占", "不可转让", "使用权"],
    "最近 3 个月": ["最近 3 个月", "最近3个月", "最近三个月", "3 个月", "3个月", "三个月"],
    "视为合格": ["视为合格", "视为验收合格", "视为该批产品验收合格"],
    "仅退还": ["仅退还", "仅退", "只退还", "只退"],
    "空白": ["空白", "缺失", "未明确", "未约定", "管辖不明"],
    "单方解除": ["单方解除", "单方解约", "单方任意解除", "提前解除", "任意解约"],
    "封顶": ["封顶", "上限", "最高不超过", "最高不超", "最高限额"],
    "补偿": ["补偿", "赔偿", "救济"],
    "缺少": ["缺少", "缺乏", "未约定", "没有", "不足", "不明确"],
    "许可": ["许可", "资质", "证照", "授权", "经营授权"],
    "替换": ["替换", "替代", "更换"],
    "第三方索赔": ["第三方索赔", "客户索赔", "第三方主张", "第三方损失"],
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run the contract-review sample test suite and write JSON/Markdown results."
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
    parser.add_argument("--case-kb-id", default=os.getenv("LEGAL_CASE_KB_ID"), required=os.getenv("LEGAL_CASE_KB_ID") is None)
    parser.add_argument("--agent-id", default=os.getenv("WEKNORA_CONTRACT_REVIEW_AGENT_ID"), help="Optional agent/template ID.")
    parser.add_argument(
        "--endpoint",
        choices=("knowledge", "agent"),
        default=os.getenv("CONTRACT_REVIEW_ENDPOINT", "knowledge"),
        help="Use knowledge-chat or agent-chat. Use agent when --agent-id points to a contract-review agent.",
    )
    parser.add_argument("--cases", type=Path, default=DEFAULT_CASES)
    parser.add_argument(
        "--judge-mode",
        choices=("auto", "quick", "reasoning"),
        default=os.getenv("CONTRACT_REVIEW_JUDGE_MODE", "auto"),
        help="Scoring mode. auto detects smart-reasoning tool traces; quick requires standard references; reasoning accepts tool evidence.",
    )
    parser.add_argument("--output-dir", type=Path, default=None)
    parser.add_argument("--only", action="append", default=[], help="Run only matching test IDs. Can be passed more than once.")
    parser.add_argument("--timeout", type=int, default=int(os.getenv("CONTRACT_REVIEW_TIMEOUT", "240")))
    parser.add_argument(
        "--workers",
        type=int,
        default=int(os.getenv("CONTRACT_REVIEW_WORKERS", "1")),
        help="Number of contract cases to run concurrently. Defaults to 1 for stable load.",
    )
    parser.add_argument("--dry-run", action="store_true", help="Print selected tests without calling the server.")
    return parser.parse_args()


def load_cases(path: Path, only: list[str]) -> list[dict[str, Any]]:
    with path.open("r", encoding="utf-8") as f:
        payload = json.load(f)
    tests = payload.get("tests", [])
    if only:
        wanted = set(only)
        tests = [t for t in tests if t["id"] in wanted or t["contract_id"] in wanted]
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
        {"title": f"contract-review-test {test_id}", "description": "Automated contract review sample test"},
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


def build_query(test: dict[str, Any], contract_text: str) -> str:
    contract_type = test.get("contract_type", "")
    type_focus = []
    if "技术服务" in contract_type:
        type_focus.append("技术服务合同必须检查并点名：服务范围是否宽泛、是否缺少交付清单/交付物清单、付款是否早于成果交付、验收标准、知识产权归属、保密期限（如合同终止后 1 年是否过短）、违约责任。")
    if "SaaS" in contract_type:
        type_focus.append("SaaS 合同必须检查并点名：SLA/可用性补偿、单方调整功能和价格、合同终止后数据删除/导出安排的具体期限（如原文为 7 日，风险点中必须写出“7 日”）、安全事件通知时限、责任上限和间接损失/第三方索赔排除。")
    if "长" in contract_type or "框架" in contract_type:
        type_focus.append("长篇/框架采购合同必须检查并点名：文件优先级、不一致时协商机制、单方调价、验收期和视为合格、供应链中断和替换供货、责任上限的具体口径（如最近 3 个月已支付订单金额为限）、间接损失和第三方索赔排除。")
    if "租赁" in contract_type or "合作" in contract_type:
        type_focus.append("租赁/合作协议必须检查并点名：产权证明、消防验收资料、授权文件、押金退还期限的具体天数（如原文为 60 日，风险点中必须写出“60 日”）和抵扣条件、收益分成口径、经营资质/许可/备案/审批责任是否过度由甲方承担。")
    type_focus_text = ""
    if type_focus:
        type_focus_text = "\n合同类型重点检查：\n" + "\n".join(f"- {item}" for item in type_focus) + "\n"

    full_report = any(section in test.get("required_sections", []) for section in ("审查摘要", "总体风险", "风险清单"))
    report_requirements = ""
    if full_report:
        multi_clause_markers = ("长", "框架", "主采购", "SaaS", "技术服务", "租赁", "合作")
        min_risks = 6 if any(marker in contract_type for marker in multi_clause_markers) else 4
        report_requirements = f"""

完整报告格式要求：
1. 必须使用以下 Markdown 二级标题，不能合并、改名或省略：
   - ## 审查摘要
   - ## 总体风险等级
   - ## 风险清单
   - ## 重点条款修改
   - ## 缺失条款建议
   - ## 待确认事项
   - ## 免责声明
2. “风险清单”必须使用表格，列出风险等级、涉及条款、风险点、合同原文依据、法律/案例依据和修改建议。
3. 写报告前，请逐项扫描以下常见合同风险维度，并把命中的具体合同措辞、数字、比例、期限或空白字段写进风险点：
   - 付款节点：预付款、较高首付款比例、付款是否早于交付/验收/成果交付；
   - 交付验收：验收标准、验收期限、短验收期、视为合格、返工/拒收机制；
   - 争议解决：争议解决地是否空白、管辖/仲裁是否明确或不利；
   - 服务/合作范围：服务范围是否宽泛，是否缺少交付清单、里程碑、服务等级、分成口径、退款和税费口径；
   - IP/保密/数据：知识产权归属、内部使用限制、保密期限、SLA/可用性补偿、安全事件通知及通知时限、数据导出/删除；
   - 单方变更：功能、范围、价格、调价、解除、搬迁等是否可由一方单方决定，沉默是否视为接受；
   - 资质与文件：产权证明、消防、授权文件、经营资质、许可、签章生效资料是否缺失或责任分配失衡；
   - 框架采购：文件优先级、不一致时协商机制、供应链中断、替换供货、责任限制、间接损失排除。
4. 本用例至少输出 {min_risks} 项不同风险，最多输出 8 项风险；如某项资料不足，请放入“待确认事项”，不要直接省略。
5. 控制篇幅，确保 7 个标题全部输出完成：审查摘要不超过 4 行；风险清单用短表格单元格；重点条款修改只写 3 条核心条款；缺失条款建议和待确认事项各不超过 5 条简短项目符号；最后必须输出“## 免责声明”和一句话。
6. 在最终回答前自行检查上述 7 个标题、风险数量，以及每个风险是否点名关键触发词，不要输出自检过程。
"""
    return f"""你是合同审查测试执行助手。请只基于本轮合同文本，以及已选择的法律条文和法律案例知识库进行审查。

审查立场：{test["stance"]}
合同类型：{test["contract_type"]}
测试问题：{test["question"]}

输出要求：
1. 按风险等级列出风险点。
2. 每个风险点说明触发原因、合同原文依据、法律或案例依据、修改建议。
3. 如知识库没有检索到可用法律或案例依据，请明确写“现有知识库未检索到可用依据”。
4. 如资料不足无法判断，请明确列为待确认事项。
{type_focus_text}
{report_requirements}

待审合同全文：

{contract_text}
"""


def chat_stream(
    args: argparse.Namespace,
    headers: dict[str, str],
    test_id: str,
    query: str,
) -> dict[str, Any]:
    session_id = create_session(args.host, headers, test_id, args.timeout)
    if args.endpoint == "agent":
        path = f"/agent-chat/{session_id}"
        payload: dict[str, Any] = {
            "query": query,
            "agent_enabled": True,
            "knowledge_base_ids": [args.law_kb_id, args.case_kb_id],
            "channel": "api",
            "mentioned_items": [
                {"id": args.law_kb_id, "name": "法律条文", "type": "kb", "kb_type": "document"},
                {"id": args.case_kb_id, "name": "法律案例", "type": "kb", "kb_type": "document"},
            ],
        }
        if args.agent_id:
            payload["agent_id"] = args.agent_id
    else:
        path = f"/knowledge-chat/{session_id}"
        payload = {
            "query": query,
            "knowledge_base_ids": [args.law_kb_id, args.case_kb_id],
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
                response_type = event.get("response_type")
                if response_type == "answer" and event.get("content"):
                    answer_parts.append(event["content"])
                if event.get("knowledge_references"):
                    references.extend(event["knowledge_references"])
                if event.get("done"):
                    done = True
    except error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {exc.code} from chat endpoint: {detail}") from exc

    elapsed = round(time.monotonic() - started, 2)
    return {
        "session_id": session_id,
        "answer": "".join(answer_parts),
        "references": references,
        "events": events,
        "done": done,
        "elapsed_seconds": elapsed,
    }


def keyword_variants(keyword: str) -> list[str]:
    variants = KEYWORD_SYNONYMS.get(keyword, [])
    return [keyword, *variants]


def compact_text(text: str) -> str:
    return "".join(text.lower().split())


def contains_any(text: str, keywords: list[str]) -> bool:
    haystack = text.lower()
    compact_haystack = compact_text(text)
    return any(keyword.lower() in haystack or compact_text(keyword) in compact_haystack for keyword in keywords)


def contains_all_groups(text: str, keywords: list[str]) -> tuple[bool, list[dict[str, Any]]]:
    groups = []
    for keyword in keywords:
        variants = keyword_variants(keyword)
        hit = contains_any(text, variants)
        groups.append({"keyword": keyword, "hit": hit, "variants": variants})
    return all(group["hit"] for group in groups), groups


def summarize_tool_evidence(events: list[dict[str, Any]]) -> dict[str, Any]:
    tool_calls = []
    tool_results = []
    evidence_results = []
    for event in events:
        response_type = event.get("response_type")
        data = event.get("data") or {}
        tool_name = data.get("tool_name")
        if response_type == "tool_call" and tool_name:
            tool_calls.append({"tool_name": tool_name, "arguments": data.get("arguments")})
        if response_type == "tool_result" and tool_name:
            result = {
                "tool_name": tool_name,
                "success": data.get("success"),
                "display_type": data.get("display_type"),
                "knowledge_base_ids": data.get("knowledge_base_ids") or [],
                "knowledge_title": data.get("knowledge_title"),
                "document_count": data.get("document_count"),
                "fetched_chunks": data.get("fetched_chunks"),
                "count": data.get("count"),
            }
            tool_results.append(result)
            if data.get("success") and tool_name in TOOL_EVIDENCE_NAMES:
                evidence_results.append(result)
    tool_names = sorted({item["tool_name"] for item in [*tool_calls, *tool_results] if item.get("tool_name")})
    return {
        "tool_call_count": len(tool_calls),
        "tool_result_count": len(tool_results),
        "tool_evidence_count": len(evidence_results),
        "tool_names": tool_names,
        "reasoning_trace": any(name in REASONING_TOOL_NAMES for name in tool_names),
        "evidence_results": evidence_results,
    }


def resolve_judge_mode(requested: str, endpoint: str, tool_evidence: dict[str, Any]) -> str:
    if requested != "auto":
        return requested
    if endpoint == "agent" and tool_evidence["reasoning_trace"]:
        return "reasoning"
    return "quick"


def judge(
    test: dict[str, Any],
    answer: str,
    references: list[dict[str, Any]],
    events: list[dict[str, Any]],
    done: bool,
    endpoint: str,
    requested_mode: str,
) -> dict[str, Any]:
    tool_evidence = summarize_tool_evidence(events)
    judge_mode = resolve_judge_mode(requested_mode, endpoint, tool_evidence)
    risk_results = []
    for risk in test.get("expected_risks", []):
        hit, keyword_groups = contains_all_groups(answer, risk.get("keywords", []))
        risk_results.append({"name": risk["name"], "hit": hit, "keywords": risk.get("keywords", []), "keyword_groups": keyword_groups})
    section_results = []
    for section in test.get("required_sections", []):
        section_results.append({"name": section, "hit": section.lower() in answer.lower()})
    citation_hits = [term for term in test.get("citation_terms", []) if contains_any(answer, [term])]
    risk_hits = sum(1 for item in risk_results if item["hit"])
    section_hits = sum(1 for item in section_results if item["hit"])
    min_risks = 2 if test["priority"] == "P0" else min(4, len(risk_results))
    standard_evidence_ok = len(citation_hits) >= 1 or len(references) >= 1
    tool_evidence_ok = tool_evidence["tool_evidence_count"] >= 1 and len(citation_hits) >= 1
    evidence_ok = standard_evidence_ok if judge_mode == "quick" else (standard_evidence_ok or tool_evidence_ok)
    passed = (
        done
        and risk_hits >= min_risks
        and section_hits >= max(1, len(section_results) - 2)
        and evidence_ok
    )
    return {
        "passed": passed,
        "judge_mode": judge_mode,
        "done": done,
        "risk_hits": risk_hits,
        "risk_total": len(risk_results),
        "section_hits": section_hits,
        "section_total": len(section_results),
        "citation_hits": citation_hits,
        "reference_count": len(references),
        "evidence_ok": evidence_ok,
        "standard_evidence_ok": standard_evidence_ok,
        "tool_evidence_ok": tool_evidence_ok,
        "tool_evidence": tool_evidence,
        "risk_results": risk_results,
        "section_results": section_results,
    }


def write_outputs(output_dir: Path, results: list[dict[str, Any]], args: argparse.Namespace) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    summary = {
        "generated_at": dt.datetime.now().isoformat(timespec="seconds"),
        "host": args.host,
        "endpoint": args.endpoint,
        "agent_id": args.agent_id,
        "judge_mode": args.judge_mode,
        "law_kb_id": args.law_kb_id,
        "case_kb_id": args.case_kb_id,
        "total": len(results),
        "passed": sum(1 for r in results if r["judge"]["passed"]),
    }
    summary["contract_review_usability"] = round(summary["passed"] / summary["total"], 4) if summary["total"] else 0
    with (output_dir / "results.json").open("w", encoding="utf-8") as f:
        json.dump({"summary": summary, "results": results}, f, ensure_ascii=False, indent=2)

    lines = [
        "# 合同审查自动测试结果",
        "",
        f"- 生成时间：{summary['generated_at']}",
        f"- 服务地址：{summary['host']}",
        f"- 接口模式：{summary['endpoint']}",
        f"- 判分模式：{summary['judge_mode']}",
        f"- 合同审查可用率：{summary['passed']} / {summary['total']} = {summary['contract_review_usability']:.2%}",
        "",
        "| 用例 | 结果 | 判分 | 风险命中 | 章节命中 | 标准引用 | 工具证据 | 耗时 |",
        "| --- | --- | --- | ---: | ---: | ---: | ---: | ---: |",
    ]
    for result in results:
        j = result["judge"]
        status = "PASS" if j["passed"] else "REVIEW"
        lines.append(
            f"| {result['id']} | {status} | {j.get('judge_mode', 'quick')} | {j['risk_hits']}/{j['risk_total']} | "
            f"{j['section_hits']}/{j['section_total']} | {j['reference_count']} | "
            f"{j.get('tool_evidence', {}).get('tool_evidence_count', 0)} | {result['elapsed_seconds']}s |"
        )
    lines.extend([
        "",
        "## 复核提示",
        "",
        "- `PASS` 代表机器检查命中核心风险、章节和证据要求，不代表正式法律意见质量通过。",
        "- `quick` 判分要求标准引用或答案内引用；`reasoning` 判分允许智能推理工具调用证据补足标准引用。",
        "- `REVIEW` 需要人工打开 `results.json` 查看完整回答、引用和未命中的检查项。",
    ])
    with (output_dir / "summary.md").open("w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def error_result(test: dict[str, Any], exc: Exception, args: argparse.Namespace) -> dict[str, Any]:
    return {
        "id": test["id"],
        "contract_id": test["contract_id"],
        "priority": test["priority"],
        "contract_file": test["contract_file"],
        "question": test["question"],
        "error": str(exc),
        "elapsed_seconds": 0,
        "judge": {
            "passed": False,
            "judge_mode": args.judge_mode,
            "done": False,
            "risk_hits": 0,
            "risk_total": 0,
            "section_hits": 0,
            "section_total": 0,
            "citation_hits": [],
            "reference_count": 0,
            "tool_evidence": {"tool_evidence_count": 0},
        },
    }


def run_test_case(args: argparse.Namespace, headers: dict[str, str], test: dict[str, Any]) -> dict[str, Any]:
    contract_path = ROOT / test["contract_file"]
    contract_text = contract_path.read_text(encoding="utf-8")
    stream_result = chat_stream(args, headers, test["id"], build_query(test, contract_text))
    judgment = judge(
        test,
        stream_result["answer"],
        stream_result["references"],
        stream_result["events"],
        stream_result["done"],
        args.endpoint,
        args.judge_mode,
    )
    return {
        "id": test["id"],
        "contract_id": test["contract_id"],
        "priority": test["priority"],
        "contract_file": test["contract_file"],
        "question": test["question"],
        "answer": stream_result["answer"],
        "references": stream_result["references"],
        "events": stream_result["events"],
        "session_id": stream_result["session_id"],
        "elapsed_seconds": stream_result["elapsed_seconds"],
        "judge": judgment,
    }


def main() -> int:
    args = parse_args()
    if args.workers < 1:
        raise SystemExit("--workers must be >= 1")
    tests = load_cases(args.cases, args.only)
    if args.dry_run:
        for test in tests:
            print(f"{test['id']}: {test['contract_file']} -> {test['endpoint'] if 'endpoint' in test else args.endpoint}")
        return 0

    headers = auth_headers(args)
    output_dir = args.output_dir
    if output_dir is None:
        stamp = dt.datetime.now().strftime("%Y%m%d-%H%M%S")
        output_dir = DEFAULT_OUTPUT_ROOT / stamp

    indexed_tests = list(enumerate(tests, start=1))
    results_by_index: dict[int, dict[str, Any]] = {}
    workers = min(args.workers, len(tests))

    if workers == 1:
        for index, test in indexed_tests:
            print(f"[{index}/{len(tests)}] running {test['id']}...", flush=True)
            try:
                result = run_test_case(args, headers, test)
                results_by_index[index] = result
                print(f"  -> {'PASS' if result['judge']['passed'] else 'REVIEW'}", flush=True)
            except Exception as exc:  # noqa: BLE001 - CLI should record failures and continue.
                results_by_index[index] = error_result(test, exc, args)
                print(f"  -> ERROR: {exc}", flush=True)
    else:
        print(f"running {len(tests)} cases with {workers} workers", flush=True)
        with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as executor:
            future_map = {}
            for index, test in indexed_tests:
                print(f"[{index}/{len(tests)}] queued {test['id']}...", flush=True)
                future = executor.submit(run_test_case, args, headers, test)
                future_map[future] = (index, test)
            for future in concurrent.futures.as_completed(future_map):
                index, test = future_map[future]
                try:
                    result = future.result()
                    results_by_index[index] = result
                    status = "PASS" if result["judge"]["passed"] else "REVIEW"
                    print(f"[{index}/{len(tests)}] {test['id']} -> {status}", flush=True)
                except Exception as exc:  # noqa: BLE001 - CLI should record failures and continue.
                    results_by_index[index] = error_result(test, exc, args)
                    print(f"[{index}/{len(tests)}] {test['id']} -> ERROR: {exc}", flush=True)

    results = [results_by_index[index] for index in range(1, len(tests) + 1)]

    write_outputs(output_dir, results, args)
    print(f"wrote {output_dir / 'summary.md'}")
    print(f"wrote {output_dir / 'results.json'}")
    return 0 if all(r["judge"]["passed"] for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
