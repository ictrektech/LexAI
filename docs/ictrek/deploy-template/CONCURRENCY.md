# LexAI 并发和队列配置

本文说明 ictrek 部署模板里的并发配置。实际部署时，把这些值写到目标机 env 文件：`.env`、`.env.tc232` 或 `.env.thor`。

## 三层控制

并发不是一个参数控制完的，至少分三层：

| 层级 | 作用 | 主要变量 |
| --- | --- | --- |
| Asynq 后台任务池 | 控制后台任务 worker 总数，以及不同任务队列的调度权重。 | `WEKNORA_ASYNQ_CONCURRENCY`、`WEKNORA_ASYNQ_QUEUE_*` |
| 后台 LLM 限流 | 防止 Graph、Wiki、自动问题生成把主 QA 模型并发吃满。 | `WEKNORA_MAIN_QA_MODEL_CONCURRENCY`、`WEKNORA_CHAT_RESERVED_CONCURRENCY`、`WEKNORA_GRAPH_LLM_CONCURRENCY`、`WEKNORA_WIKI_INGEST_*` |
| 模型服务容量 | 控制 vLLM、Ollama 或其他 OpenAI-compatible 服务实际能同时处理多少请求。 | `VLLM_MAX_NUM_SEQS`、`BGE_VLLM_MAX_NUM_SEQS`、`CONCURRENCY_POOL_SIZE`、`BATCH_EMBED_SIZE`、`OLLAMA_NUM_PARALLEL` |

队列权重不是硬性的模型并发预留。真正给聊天保留模型槽位的是后台 LLM 限流。

## 主 QA/LLM 并发

对话、Graph 抽取、Wiki 生成、自动问题生成可能共用同一个主 QA/LLM 模型。部署时按模型服务真实容量配置：

```dotenv
WEKNORA_MAIN_QA_MODEL_CONCURRENCY=8
WEKNORA_CHAT_RESERVED_CONCURRENCY=3
WEKNORA_GRAPH_LLM_CONCURRENCY=2
WEKNORA_WIKI_INGEST_MAP_PARALLEL=2
WEKNORA_WIKI_INGEST_REDUCE_PARALLEL=2
```

`WEKNORA_MAIN_QA_MODEL_CONCURRENCY` 应该对齐主 QA 模型服务的真实在线并发。vLLM 场景下，通常和 `VLLM_MAX_NUM_SEQS` 保持一致。

`WEKNORA_CHAT_RESERVED_CONCURRENCY` 是给在线聊天保留的最低并发，不让后台 LLM 任务占用。它是 LexAI 应用侧的后台 LLM 限流，不是 vLLM 自带的硬隔离；所有文档后处理、Graph、Wiki、自动问题生成等后台 LLM 调用都必须走 `acquireBackgroundLLMSlot`，否则会绕过预留直接占满模型服务。后台 LLM 可用槽位近似为：

```text
background_llm_slots = WEKNORA_MAIN_QA_MODEL_CONCURRENCY - WEKNORA_CHAT_RESERVED_CONCURRENCY
```

如果两个值都大于 0，且 `main <= reserved`，LexAI 仍会保留 1 个后台槽位，避免 Graph/Wiki/Question 完全不执行。如果任意一个值为空或为 `0`，后台 LLM 限流不会启用。

`WEKNORA_GRAPH_LLM_CONCURRENCY` 限制单个文档 Graph 抽取中的 LLM 并发。

`WEKNORA_WIKI_INGEST_MAP_PARALLEL` 和 `WEKNORA_WIKI_INGEST_REDUCE_PARALLEL` 限制 Wiki map/reduce 阶段的 LLM 并发。知识库级别的 `wiki_config.ingest_map_parallel` 或 `wiki_config.ingest_reduce_parallel` 会覆盖 env 默认值。

## Asynq 队列权重

后台任务队列权重通过 env 读取：

```dotenv
WEKNORA_ASYNQ_CONCURRENCY=9
WEKNORA_ASYNQ_QUEUE_CRITICAL=6
WEKNORA_ASYNQ_QUEUE_DEFAULT=4
WEKNORA_ASYNQ_QUEUE_LOW=2
WEKNORA_ASYNQ_QUEUE_MULTIMODAL=2
WEKNORA_ASYNQ_QUEUE_GRAPH=2
WEKNORA_ASYNQ_QUEUE_QUESTION=2
```

`WEKNORA_ASYNQ_CONCURRENCY` 是后台 worker 总并发。`WEKNORA_ASYNQ_QUEUE_*` 是队列调度权重，权重越高越容易被调度，但不是严格的每队列并发上限，不能用它代替后台 LLM limiter。

小机器上不要把 Graph 和 Question 队列权重调太高。聊天请求本身不走这些后台队列，但后台任务仍可能竞争同一个 LLM 或 Embedding 模型服务。

## Embedding 并发

文档向量化主要看这几个参数：

```dotenv
BATCH_EMBED_SIZE=4
CONCURRENCY_POOL_SIZE=5
BGE_VLLM_MAX_NUM_SEQS=8
```

`BATCH_EMBED_SIZE` 是单次 embedding 请求里打包的 chunk 数。

`CONCURRENCY_POOL_SIZE` 是应用侧文档 embedding 请求并发上限。它如果低于文档 worker 数，后台解析可能看起来卡在 embedding 阶段。

`BGE_VLLM_MAX_NUM_SEQS` 是 bge-m3 vLLM 服务的序列并发。想给聊天检索保留余量时，应让文档 embedding 的应用侧并发低于这个值，或单独降低后台解析 worker 数。

如果使用 Ollama 作为 Embedding 备用路径，`OLLAMA_NUM_PARALLEL` 控制 Ollama 侧请求并发。Thor 默认 Embedding 是 vLLM bge-m3，Ollama bge-m3 只作为备用，不作为常驻主路径。

## 推荐值

| 机器类型 | LLM 服务并发 | 聊天保留 | Graph | Wiki map/reduce | Graph 队列 | Question 队列 | 说明 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Thor `192.168.1.81` | 8 | 3 | 2 | 2 / 2 | 2 | 2 | QA/Graph/Wiki/Question 共用 9B vLLM，后台 LLM 最多占 5 个槽位。 |
| tc232 9B vLLM | 6 | 2 | 2 | 2 / 2 | 2 | 2 | 按 tc232 当前 6 并发模型容量保留 2 个聊天槽位。 |
| 通用 4 并发主机 | 4 | 1 | 2 | 1 / 1 | 2 | 2 | 优先降低 Wiki 并发，不要先压缩聊天保留。 |

Thor 的 bge-m3 vLLM 推荐值：

```dotenv
BGE_VLLM_MAX_NUM_SEQS=8
WEKNORA_ASYNQ_CONCURRENCY=9
CONCURRENCY_POOL_SIZE=5
BATCH_EMBED_SIZE=4
```

这样 bge-m3 vLLM 最大 8 并发，文档 embedding 的应用侧并发最多 5，给聊天检索保留约 3 个服务槽位。如果聊天检索仍出现排队，应继续降低 `CONCURRENCY_POOL_SIZE` 或后台 worker，而不是提高 bge-m3 vLLM 到机器承受不了的并发。

## 调参判断

| 现象 | 优先调整 |
| --- | --- |
| 文档入库时聊天变慢 | 增大 `WEKNORA_CHAT_RESERVED_CONCURRENCY`，或降低 Graph/Wiki/Question 的并发和队列权重。 |
| Graph 或 Wiki 很慢，但聊天正常 | 只有在模型服务还有余量时，才提高 `WEKNORA_GRAPH_LLM_CONCURRENCY` 或 Wiki map/reduce。 |
| 卡在 embedding 阶段 | 先检查 bge-m3 服务是否 ready，再对比 `WEKNORA_ASYNQ_CONCURRENCY`、`CONCURRENCY_POOL_SIZE`、`BATCH_EMBED_SIZE`、`BGE_VLLM_MAX_NUM_SEQS`。 |
| GPU 显存接近打满 | 先降低模型服务侧参数，例如 `VLLM_MAX_NUM_SEQS`、上下文长度或显存占用率，再把应用侧并发同步降下来。 |

## 现场确认

在目标机上看运行中的容器，不要只看 env 文件：

```bash
docker inspect lexai-thor-app-1 --format '{{range .Config.Env}}{{println .}}{{end}}' \
  | grep -E '^(WEKNORA_MAIN_QA_MODEL_CONCURRENCY|WEKNORA_CHAT_RESERVED_CONCURRENCY|CONCURRENCY_POOL_SIZE|BATCH_EMBED_SIZE)='

docker inspect qwen35-9b-vllm --format '{{range .Config.Cmd}}{{println .}}{{end}}' \
  | grep -E 'max-num-seqs|max-model-len|max-num-batched-tokens|gpu-memory-utilization'

curl -sS http://127.0.0.1:32222/metrics \
  | grep -E 'vllm:num_requests_(running|waiting)'
curl -sS http://127.0.0.1:32223/metrics \
  | grep -E 'vllm:num_requests_(running|waiting)'
```

Thor 当前配置下，后台 LLM 正常应长期压在 5 个以内；用户聊天同时发生时，qwen running 可以短时超过 5，但不应持续出现 `waiting > 0`。bge-m3 当前是 8 槽，文档 embedding 应用侧最多 5 个请求；如果 bge `waiting > 0`，先降 `CONCURRENCY_POOL_SIZE` 或后台 worker。

修改后要同步 env、compose 里的模型服务参数和内置模型配置。只改 env 时，重新执行对应 compose `up -d` 即可让 app 读取新值。
