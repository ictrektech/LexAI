#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-${ROOT_DIR}/.env.thor}"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/docker-compose.thor.yml}"
NETWORK_NAME="$(
  python3 - "$ENV_FILE" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
network = "lexai"
if path.exists():
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.startswith("LEXAI_NETWORK="):
            network = line.split("=", 1)[1].strip() or network
print(network)
PY
)"

docker network inspect "$NETWORK_NAME" >/dev/null 2>&1 || docker network create "$NETWORK_NAME" >/dev/null

ENV_FILE="$ENV_FILE" \
COMPOSE_FILE="$COMPOSE_FILE" \
"${ROOT_DIR}/deploy.sh" --platform thor "$@"
