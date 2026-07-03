#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOST="${HOST:-tc232}"
DEST="${DEST:-/data/jhu/lexai-tc232-deploy}"

rsync -az \
  "${ROOT_DIR}/deploy.sh" \
  "${ROOT_DIR}/deploy-tc232.sh" \
  "${ROOT_DIR}/docker-compose.yml" \
  "${ROOT_DIR}/docker-compose.tc232.yml" \
  "${ROOT_DIR}/README.md" \
  "${HOST}:${DEST}/"

rsync -az "${ROOT_DIR}/config/" "${HOST}:${DEST}/config/"

echo "[INFO] synced deploy template to ${HOST}:${DEST}"
