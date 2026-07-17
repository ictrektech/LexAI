# CLI RAG Full Loop E2E 测试样例

本目录记录 CLI RAG Full Loop E2E 的执行方式和清理规范，用于验证基础 RAG 闭环：建库、上传、解析、检索、问答和引用返回。

该测试只验证基础链路是否可跑通，不验证法律专项回答质量。

## 文件结构

| 文件 | 用途 |
| --- | --- |
| [README.md](README.md) | 测试说明、执行命令、通过标准和清理规范 |
| [../../../../cli/acceptance/e2e/e2e_test.go](../../../../cli/acceptance/e2e/e2e_test.go) | 自动化测试实现 |

## 前置条件

- WeKnora 后端服务可访问；
- 已获取可访问该服务的 token；
- 服务中已注册 QA 模型和 Embedding 模型；
- 在 `cli/` 目录下执行测试。

## 环境变量

本地开发环境变量示例：

```bash
export WEKNORA_E2E_HOST="http://localhost:8080"
export WEKNORA_E2E_TOKEN="$(
  curl -fsS -X POST "$WEKNORA_E2E_HOST/api/v1/auth/auto-setup" \
    -H "Content-Type: application/json" \
    -d '{}' \
  | python3 -c 'import json,sys; print(json.load(sys.stdin).get("token",""))'
)"
export WEKNORA_E2E_KB_NAME_PREFIX="cli-e2e-"
export WEKNORA_E2E_CHAT_MODEL="lexai-vllm-qwen35-9b-awq-qa"
export WEKNORA_E2E_EMBEDDING_MODEL="lexai-vllm-bge-m3-embedding"
```

上述 token 获取方式适用于本地开发部署且 `WEKNORA_SINGLE_USER_MODE=true` 的环境；测试服务器或生产环境应替换为对应环境的服务地址、token 和模型 ID。

## 自动执行

```bash
cd cli
go test -count=1 -tags=acceptance_e2e -run TestRAGFullLoop -v -timeout=8m ./acceptance/e2e/...
```

`-count=1` 用于禁用 Go test 缓存，确保每次执行都会真实访问后端服务并创建、清理本次测试知识库；如果输出中出现 `(cached)`，表示复用了历史测试结果，本次并未重新执行 E2E 流程。

## 通过标准

- 测试知识库创建成功；
- 测试文档上传成功；
- 文档状态进入可检索状态；
- chunk 检索返回至少 1 条结果；
- RAG chat 返回非空回答；
- 如服务配置支持引用，回答应返回引用索引；
- Go test 最终结果为 `PASS`。

## 清理方式

- 正常执行完成且清理成功时，测试用例会通过 `t.Cleanup` 自动删除本次创建的测试知识库，常规执行后不用手动清理；
- `t.Cleanup` 只清理本次测试创建的知识库，不会批量删除历史残留的 `cli-e2e-` 知识库；
- 如果测试进程被强制中断、机器重启、后端服务不可用或清理请求失败，可能残留名称前缀为 `cli-e2e-` 的测试知识库；
- 手动清理仅作为异常残留补救手段，执行前应确认待删除知识库属于临时测试数据，不要按前缀以外的条件批量删除业务知识库。

异常残留处理命令：

```bash
weknora kb list --format json
weknora kb delete <kb-id> -y --format json
```

如果确认所有 `cli-e2e-` 前缀知识库均为临时测试数据，可以批量清理：

```bash
weknora kb list --format json \
  | jq -r '.data[] | select(.name | startswith("cli-e2e-")) | .id' \
  | xargs -r -n1 weknora kb delete -y --format json
```
