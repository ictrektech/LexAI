#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ENV_FILE="${ENV_FILE:-${ROOT_DIR}/.env.tc232}" \
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/docker-compose.tc232.yml}" \
"${ROOT_DIR}/deploy.sh" --platform "${PLATFORM:-amd}" "$@"
