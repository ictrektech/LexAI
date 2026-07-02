# LexAI deploy template

Copy `.env.example` to `.env`, edit secrets, then run:

```bash
./deploy.sh --platform amd
./deploy.sh --platform l4t
./deploy.sh --platform thor
```

`deploy.sh` reads the latest image tags from Feishu and writes image variables into `.env` before running `docker compose up -d`.

All services join the `lexai` Docker network. Host ports start at 30000; service-to-service traffic uses container names inside the network.

For `tc232`, use the dedicated compose file. It expects the existing `qwen35-9b-awq-vllm` container to already be attached to the external `lexai` network.

```bash
cp .env.tc232.example .env.tc232
./deploy-tc232.sh
```
