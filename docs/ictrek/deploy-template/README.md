# LexAI deploy template

Copy `.env.example` to `.env`, edit secrets, then run:

```bash
./deploy.sh --platform amd
./deploy.sh --platform l4t
./deploy.sh --platform thor
```

`deploy.sh` reads the latest image tag from each component's own Feishu column and writes image variables into `.env` before running `docker compose up -d`. The LexAI app, UI, and docreader tags are resolved independently and may be different.

All services join the `lexai` Docker network. Host ports start at 30000; service-to-service traffic uses container names inside the network.

By default these compose templates enable `WEKNORA_SINGLE_USER_MODE=true`, so the web UI auto-creates the fixed default user space and enters the app without showing the login page. Set it to `false` to restore normal login.

The deployment includes a legal knowledge graph preset at `config/legal_graph_preset.json`, mounted read-only in the app container at `/app/config/legal_graph_preset.json`. LexAI is a legal-only deployment, so this preset is the default entity/relation setup for legal documents, statutes, contracts, and cases. See `../legal-knowledge-graph-config.md` for the UI configuration guide and manual override notes.

Wiki synthesis uses the same model as the knowledge base's main QA model by default. On `tc232` that main QA model is `lexai-vllm-qwen35-9b-awq-qa`; on other machines it should be whichever QA model is selected for the knowledge base.

Thinking is disabled by default for LexAI QA responses. vLLM OpenAI-compatible models use `chat_template_kwargs.enable_thinking=false`. Ollama's native chat API uses `think=false`; Ollama OpenAI-compatible models should use `extra_config.thinking_control=reasoning_effort`, which sends `reasoning_effort=none`.

For `tc232`, use the dedicated compose file. It expects the existing `qwen35-9b-awq-vllm` container to already be attached to the external `lexai` network.

```bash
cp .env.tc232.example .env.tc232
./deploy-tc232.sh
```
