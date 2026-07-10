#!/usr/bin/env bash
set -euo pipefail

SCRIPT_PATH="${BASH_SOURCE[0]:-$0}"
ROOT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
FEISHU_READ_CONFIG_FILE="${FEISHU_READ_CONFIG_FILE:-${HOME}/.feishu.components.json}"
FEISHU_CONFIG_FILE="${FEISHU_CONFIG_FILE:-${HOME}/.feishu.json}"
SPREADSHEET_TOKEN="${FEISHU_SPREADSHEET_TOKEN:-Htotsn3oahO1zxt73YMcaB1zn8e}"
REGISTRY="${REGISTRY:-swr.cn-southwest-2.myhuaweicloud.com/ictrek}"
ENV_FILE="${ENV_FILE:-${ROOT_DIR}/.env}"
COMPOSE_FILE="${COMPOSE_FILE:-${ROOT_DIR}/docker-compose.yml}"
PLATFORM=""
SHEET_TITLE=""
DRY_RUN=0
CHECK_ONLY=0
CONFIG_CHANGED="${LEXAI_DEPLOY_CONFIG_CHANGED:-0}"

usage() {
  cat <<'EOF'
Usage: ./deploy.sh --platform amd|l4t|thor [--sheet SHEET] [--compose-file FILE] [--check-only] [--dry-run]

Looks up the latest LexAI, model_hub, and ollama_server image tags in Feishu,
writes them to .env, pulls the images, and recreates only managed services whose
running image digest or deployment config changed.

Environment:
  FEISHU_READ_CONFIG_FILE  Defaults to ~/.feishu.components.json for read-only lookup
  FEISHU_CONFIG_FILE       Fallback read config, defaults to ~/.feishu.json
  FEISHU_SPREADSHEET_TOKEN Defaults to the ictrek release sheet token
  ENV_FILE                 Defaults to ./deploy-template/.env
  COMPOSE_FILE             Defaults to ./deploy-template/docker-compose.yml
  LEXAI_DEPLOY_CONFIG_CHANGED=1 applies changed compose/config files without touching DB services
EOF
}

log() { echo "[INFO] $*"; }
die() { echo "[ERROR] $*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing command: $1"
}

docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  else
    docker-compose "$@"
  fi
}

compose_has_service() {
  local service="$1"
  docker_compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" config --services \
    | grep -qx "$service"
}

service_cid() {
  docker_compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps -q "$1" 2>/dev/null || true
}

pull_image_if_needed() {
  local image="$1"
  log "pull ${image}"
  docker pull "$image" >/dev/null
}

remote_image_digest() {
  local image="$1"
  local manifest
  manifest="$(docker manifest inspect "$image" 2>/dev/null || true)"
  [[ -n "$manifest" ]] || return 0
  printf '%s' "$manifest" | python3 -c '
import json, sys
data = json.load(sys.stdin)
digest = data.get("Descriptor", {}).get("digest") or data.get("config", {}).get("digest") or ""
print(digest)
'
}

running_image_digest() {
  local service="$1"
  local cid image repo_digest
  cid="$(service_cid "$service")"
  [[ -n "$cid" ]] || return 0
  image="$(docker inspect "$cid" --format '{{.Config.Image}}' 2>/dev/null || true)"
  [[ -n "$image" ]] || return 0
  repo_digest="$(docker image inspect "$image" --format '{{range .RepoDigests}}{{println .}}{{end}}' 2>/dev/null | head -n 1 || true)"
  [[ -n "$repo_digest" ]] || return 0
  echo "${repo_digest##*@}"
}

running_image_ref() {
  local service="$1"
  local cid
  cid="$(service_cid "$service")"
  [[ -n "$cid" ]] || return 0
  docker inspect "$cid" --format '{{.Config.Image}}' 2>/dev/null || true
}

service_needs_remote_image_update() {
  local service="$1"
  local image="$2"
  local cid current_ref remote current
  compose_has_service "$service" || return 1
  cid="$(service_cid "$service")"
  [[ -n "$cid" ]] || return 0
  current_ref="$(running_image_ref "$service")"
  if [[ -n "$current_ref" && "$current_ref" != "$image" ]]; then
    log "service ${service} image changed: ${current_ref} -> ${image}"
    return 0
  fi
  remote="$(remote_image_digest "$image")"
  current="$(running_image_digest "$service")"
  [[ -n "$remote" && -n "$current" ]] || return 0
  [[ "$remote" != "$current" ]]
}

service_needs_image_update() {
  local service="$1"
  local image="$2"
  local cid current target
  compose_has_service "$service" || return 1
  pull_image_if_needed "$image"
  cid="$(service_cid "$service")"
  target="$(docker image inspect "$image" --format '{{.Id}}')"
  [[ -n "$cid" ]] || return 0
  current="$(docker inspect "$cid" --format '{{.Image}}')"
  [[ "$current" != "$target" ]]
}

append_service_once() {
  local service="$1"
  local existing
  for existing in "${UPDATE_SERVICES[@]:-}"; do
    [[ "$existing" == "$service" ]] && return 0
  done
  UPDATE_SERVICES+=("$service")
}

wait_service_healthy() {
  local service="$1"
  local timeout="${2:-180}"
  local cid status deadline
  cid="$(docker_compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps -q "$service")"
  [[ -n "$cid" ]] || die "service not running: $service"

  if ! docker inspect "$cid" --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' >/dev/null 2>&1; then
    return 0
  fi

  deadline=$((SECONDS + timeout))
  while (( SECONDS < deadline )); do
    status="$(docker inspect "$cid" --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' 2>/dev/null || true)"
    [[ -z "$status" || "$status" == "healthy" ]] && return 0
    sleep 3
  done
  die "service did not become healthy: $service"
}

read_feishu_field() {
  local file="$1"
  local field="$2"
  python3 - "$file" "$field" <<'PY'
import json, sys
path, field = sys.argv[1], sys.argv[2]
with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)
val = data.get(field, "")
print(val if isinstance(val, str) else str(val))
PY
}

resolve_feishu_reader() {
  local config app_id app_secret token sheet_id err_file last_config
  FEISHU_ACTIVE_CONFIG=""
  TOKEN=""
  SHEET_ID=""
  last_config=""

  for config in "$FEISHU_READ_CONFIG_FILE" "$FEISHU_CONFIG_FILE"; do
    [[ -n "$config" && -f "$config" ]] || continue
    [[ "$config" != "$last_config" ]] || continue
    last_config="$config"

    app_id="$(read_feishu_field "$config" feishu_app_id)"
    app_secret="$(read_feishu_field "$config" feishu_app_secret)"
    [[ -n "$app_id" && -n "$app_secret" ]] || continue

    err_file="$(mktemp)"
    if token="$(get_feishu_token "$app_id" "$app_secret" 2>"$err_file")" \
      && sheet_id="$(get_sheet_id_by_title "$token" "$SHEET_TITLE" 2>>"$err_file")"; then
      FEISHU_ACTIVE_CONFIG="$config"
      TOKEN="$token"
      SHEET_ID="$sheet_id"
      rm -f "$err_file"
      log "Feishu read config=${config}"
      return 0
    fi

    log "Feishu read config failed, trying fallback: ${config}"
    rm -f "$err_file"
  done

  die "No Feishu read config can read sheet ${SHEET_TITLE}; checked ${FEISHU_READ_CONFIG_FILE} and ${FEISHU_CONFIG_FILE}"
}

feishu_api_json() {
  local method="$1"
  local url="$2"
  local token="$3"
  curl --fail -sS -X "$method" "$url" -H "Authorization: Bearer ${token}"
}

get_feishu_token() {
  local app_id="$1"
  local app_secret="$2"
  local resp
  resp="$(
    curl --fail -sS -X POST "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal" \
      -H "Content-Type: application/json" \
      -d "{\"app_id\":\"${app_id}\",\"app_secret\":\"${app_secret}\"}"
  )"
  python3 - "$resp" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
if data.get("code") != 0:
    raise SystemExit(f"get_feishu_token failed: {data}")
print(data["tenant_access_token"])
PY
}

get_sheet_id_by_title() {
  local token="$1"
  local title="$2"
  local resp
  resp="$(feishu_api_json "GET" "https://open.feishu.cn/open-apis/sheets/v3/spreadsheets/${SPREADSHEET_TOKEN}/sheets/query" "$token")"
  python3 - "$title" "$resp" <<'PY'
import json, sys
title, resp = sys.argv[1], sys.argv[2]
data = json.loads(resp)
if data.get("code") != 0:
    raise SystemExit(f"query sheets failed: {data}")
for sheet in data.get("data", {}).get("sheets", []):
    if sheet.get("title") == title:
        print(sheet["sheet_id"])
        raise SystemExit(0)
raise SystemExit(f"sheet title not found: {title}")
PY
}

get_range_values() {
  local token="$1"
  local range="$2"
  feishu_api_json "GET" "https://open.feishu.cn/open-apis/sheets/v2/spreadsheets/${SPREADSHEET_TOKEN}/values/${range}" "$token"
}

column_letter() {
  python3 - "$1" <<'PY'
import sys
n = int(sys.argv[1])
s = ""
while n > 0:
    n, r = divmod(n - 1, 26)
    s = chr(ord("A") + r) + s
print(s)
PY
}

get_sheet_column_count() {
  local token="$1"
  local sheet_id="$2"
  local resp
  resp="$(feishu_api_json "GET" "https://open.feishu.cn/open-apis/sheets/v3/spreadsheets/${SPREADSHEET_TOKEN}/sheets/query" "$token")"
  python3 - "$sheet_id" "$resp" <<'PY'
import json, sys
sheet_id, resp = sys.argv[1], sys.argv[2]
data = json.loads(resp)
if data.get("code") != 0:
    raise SystemExit(f"query sheets failed: {data}")
for sheet in data.get("data", {}).get("sheets", []):
    if sheet.get("sheet_id") == sheet_id:
        print(sheet.get("grid_properties", {}).get("column_count", 1))
        raise SystemExit(0)
raise SystemExit(f"sheet id not found: {sheet_id}")
PY
}

find_component_column_letter() {
  local token="$1"
  local sheet_id="$2"
  local component="$3"
  local column_count end_col resp
  column_count="$(get_sheet_column_count "$token" "$sheet_id")"
  end_col="$(column_letter "$column_count")"
  resp="$(get_range_values "$token" "${sheet_id}!A1:${end_col}1")"
  python3 - "$component" "$resp" <<'PY'
import json, sys
target, resp = sys.argv[1], sys.argv[2]
data = json.loads(resp)
if data.get("code") != 0:
    raise SystemExit(f"read header failed: {data}")
values = data.get("data", {}).get("valueRange", {}).get("values", [])
row = values[0] if values else []

def text(value):
    if value is None:
        return ""
    if isinstance(value, str):
        return value.strip()
    if isinstance(value, dict):
        return str(value.get("text") or value.get("link") or "").strip()
    if isinstance(value, list):
        return "".join(text(v) for v in value).strip()
    return str(value).strip()

def col(num):
    out = ""
    while num > 0:
        num, rem = divmod(num - 1, 26)
        out = chr(ord("A") + rem) + out
    return out

for index, value in enumerate(row, start=1):
    if text(value) == target:
        print(col(index))
        raise SystemExit(0)
raise SystemExit(f"component column not found in row1: {target}")
PY
}

find_latest_tag() {
  local token="$1"
  local sheet_id="$2"
  local column="$3"
  local resp
  resp="$(get_range_values "$token" "${sheet_id}!${column}4:${column}2000")"
  python3 - "$resp" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
if data.get("code") != 0:
    raise SystemExit(f"read version column failed: {data}")
values = data.get("data", {}).get("valueRange", {}).get("values", [])
for row in values:
    if not row or row[0] is None:
        continue
    value = str(row[0]).strip()
    if value and value.lower() != "null":
        print(value)
        raise SystemExit(0)
raise SystemExit("latest version not found")
PY
}

latest_tag_for_component() {
  local token="$1"
  local sheet_id="$2"
  local component="$3"
  local column tag
  column="$(find_component_column_letter "$token" "$sheet_id" "$component")"
  tag="$(find_latest_tag "$token" "$sheet_id" "$column")"
  echo "$tag"
}

image_with_tag() {
  local repository="$1"
  local tag="$2"
  echo "${repository}:${tag}"
}

write_env_value() {
  local key="$1"
  local value="$2"
  local file="$3"
  python3 - "$key" "$value" "$file" <<'PY'
from pathlib import Path
import sys
key, value, path = sys.argv[1], sys.argv[2], Path(sys.argv[3])
lines = path.read_text(encoding="utf-8").splitlines() if path.exists() else []
prefix = key + "="
out = []
done = False
for line in lines:
    if line.startswith(prefix):
        out.append(f"{key}={value}")
        done = True
    else:
        out.append(line)
if not done:
    out.append(f"{key}={value}")
path.write_text("\n".join(out) + "\n", encoding="utf-8")
PY
}

platform_sheet() {
  case "$1" in
    amd) echo "AMD_with_cuda" ;;
    l4t) echo "l4t" ;;
    thor) echo "thor_spark" ;;
    *) die "unsupported platform: $1" ;;
  esac
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --platform)
      PLATFORM="$2"
      shift 2
      ;;
    --sheet)
      SHEET_TITLE="$2"
      shift 2
      ;;
    --compose-file)
      COMPOSE_FILE="$2"
      shift 2
      ;;
    --check-only)
      CHECK_ONLY=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1"
      ;;
  esac
done

[[ -n "$PLATFORM" ]] || die "--platform amd|l4t|thor is required"
SHEET_TITLE="${SHEET_TITLE:-$(platform_sheet "$PLATFORM")}"
require_cmd curl
require_cmd python3

resolve_feishu_reader

LEXAI_APP_TAG="$(latest_tag_for_component "$TOKEN" "$SHEET_ID" lexai)"
LEXAI_UI_TAG="$(latest_tag_for_component "$TOKEN" "$SHEET_ID" lexai-ui)"
LEXAI_DOCREADER_TAG="$(latest_tag_for_component "$TOKEN" "$SHEET_ID" lexai-docreader)"
MODEL_HUB_BACKEND_TAG="$(latest_tag_for_component "$TOKEN" "$SHEET_ID" model_hub_backend)"
MODEL_HUB_FRONTEND_TAG="$(latest_tag_for_component "$TOKEN" "$SHEET_ID" model_hub_frontend)"
OLLAMA_SERVER_TAG="$(latest_tag_for_component "$TOKEN" "$SHEET_ID" ollama_server)"

LEXAI_APP_IMAGE="$(image_with_tag "${REGISTRY}/lexai" "$LEXAI_APP_TAG")"
LEXAI_UI_IMAGE="$(image_with_tag "${REGISTRY}/lexai-ui" "$LEXAI_UI_TAG")"
LEXAI_DOCREADER_IMAGE="$(image_with_tag "${REGISTRY}/lexai-docreader" "$LEXAI_DOCREADER_TAG")"
MODEL_HUB_BACKEND_IMAGE="$(image_with_tag "${REGISTRY}/model-hub-backend" "$MODEL_HUB_BACKEND_TAG")"
MODEL_HUB_FRONTEND_IMAGE="$(image_with_tag "${REGISTRY}/model-hub-frontend" "$MODEL_HUB_FRONTEND_TAG")"
OLLAMA_SERVER_IMAGE="$(image_with_tag "${REGISTRY}/ollama_server" "$OLLAMA_SERVER_TAG")"

log "sheet=${SHEET_TITLE}"
log "LEXAI_APP_IMAGE=${LEXAI_APP_IMAGE} tag=${LEXAI_APP_TAG}"
log "LEXAI_UI_IMAGE=${LEXAI_UI_IMAGE} tag=${LEXAI_UI_TAG}"
log "LEXAI_DOCREADER_IMAGE=${LEXAI_DOCREADER_IMAGE} tag=${LEXAI_DOCREADER_TAG}"
log "MODEL_HUB_BACKEND_IMAGE=${MODEL_HUB_BACKEND_IMAGE} tag=${MODEL_HUB_BACKEND_TAG}"
log "MODEL_HUB_FRONTEND_IMAGE=${MODEL_HUB_FRONTEND_IMAGE} tag=${MODEL_HUB_FRONTEND_TAG}"
log "OLLAMA_SERVER_IMAGE=${OLLAMA_SERVER_IMAGE} tag=${OLLAMA_SERVER_TAG}"

if [[ "$DRY_RUN" == "1" ]]; then
  exit 0
fi

require_cmd docker

if [[ "$CHECK_ONLY" == "1" ]]; then
  tmp_env="$(mktemp)"
  trap 'rm -f "$tmp_env"' EXIT
  if [[ -f "$ENV_FILE" ]]; then
    cp "$ENV_FILE" "$tmp_env"
  else
    : > "$tmp_env"
  fi
  ENV_FILE="$tmp_env"
fi

write_env_value LEXAI_APP_IMAGE "$LEXAI_APP_IMAGE" "$ENV_FILE"
write_env_value LEXAI_UI_IMAGE "$LEXAI_UI_IMAGE" "$ENV_FILE"
write_env_value LEXAI_DOCREADER_IMAGE "$LEXAI_DOCREADER_IMAGE" "$ENV_FILE"
write_env_value MODEL_HUB_BACKEND_IMAGE "$MODEL_HUB_BACKEND_IMAGE" "$ENV_FILE"
write_env_value MODEL_HUB_FRONTEND_IMAGE "$MODEL_HUB_FRONTEND_IMAGE" "$ENV_FILE"
write_env_value OLLAMA_SERVER_IMAGE "$OLLAMA_SERVER_IMAGE" "$ENV_FILE"

cd "$ROOT_DIR"
UPDATE_SERVICES=()
if [[ "$CHECK_ONLY" == "1" ]]; then
  if service_needs_remote_image_update frontend "$LEXAI_UI_IMAGE"; then append_service_once frontend; fi
  if service_needs_remote_image_update app "$LEXAI_APP_IMAGE"; then append_service_once app; fi
  if service_needs_remote_image_update docreader "$LEXAI_DOCREADER_IMAGE"; then append_service_once docreader; fi
  if service_needs_remote_image_update model-hub-frontend "$MODEL_HUB_FRONTEND_IMAGE"; then append_service_once model-hub-frontend; fi
  if service_needs_remote_image_update model-hub-backend "$MODEL_HUB_BACKEND_IMAGE"; then append_service_once model-hub-backend; fi
  if service_needs_remote_image_update model-hub-bootstrap "$MODEL_HUB_BACKEND_IMAGE"; then append_service_once model-hub-bootstrap; fi
  if service_needs_remote_image_update model-hub-ollama "$OLLAMA_SERVER_IMAGE"; then append_service_once model-hub-ollama; fi
  if [[ "${WEKNORA_SKIP_DEPLOY_UPDATER_UPDATE:-false}" != "true" ]] && service_needs_remote_image_update deploy-updater "$LEXAI_APP_IMAGE"; then append_service_once deploy-updater; fi
else
  if service_needs_image_update frontend "$LEXAI_UI_IMAGE"; then append_service_once frontend; fi
  if service_needs_image_update app "$LEXAI_APP_IMAGE"; then append_service_once app; fi
  if service_needs_image_update docreader "$LEXAI_DOCREADER_IMAGE"; then append_service_once docreader; fi
  if service_needs_image_update model-hub-frontend "$MODEL_HUB_FRONTEND_IMAGE"; then append_service_once model-hub-frontend; fi
  if service_needs_image_update model-hub-backend "$MODEL_HUB_BACKEND_IMAGE"; then append_service_once model-hub-backend; fi
  if service_needs_image_update model-hub-bootstrap "$MODEL_HUB_BACKEND_IMAGE"; then append_service_once model-hub-bootstrap; fi
  if service_needs_image_update model-hub-ollama "$OLLAMA_SERVER_IMAGE"; then append_service_once model-hub-ollama; fi
  if [[ "${WEKNORA_SKIP_DEPLOY_UPDATER_UPDATE:-false}" != "true" ]] && service_needs_image_update deploy-updater "$LEXAI_APP_IMAGE"; then append_service_once deploy-updater; fi
fi

if [[ "$CONFIG_CHANGED" == "1" ]]; then
  for service in frontend app docreader model-hub-frontend model-hub-backend model-hub-bootstrap model-hub-ollama qwen35-9b-vllm bge-m3-vllm; do
    compose_has_service "$service" && append_service_once "$service"
  done
  if [[ "${WEKNORA_SKIP_DEPLOY_UPDATER_UPDATE:-false}" != "true" ]] && compose_has_service deploy-updater; then
    append_service_once deploy-updater
  fi
fi

if [[ "${#UPDATE_SERVICES[@]}" == "0" ]]; then
  if [[ "$CHECK_ONLY" == "1" ]]; then
    echo "UPDATE_AVAILABLE=0"
    echo "CONFIG_CHANGED=${CONFIG_CHANGED}"
    echo "UPDATE_SERVICES="
  fi
  log "no update needed: deploy files and pulled image digests match running containers"
  exit 0
fi

if [[ "$CHECK_ONLY" == "1" ]]; then
  echo "UPDATE_AVAILABLE=1"
  echo "CONFIG_CHANGED=${CONFIG_CHANGED}"
  echo "UPDATE_SERVICES=${UPDATE_SERVICES[*]}"
  exit 0
fi

log "updating services: ${UPDATE_SERVICES[*]}"
docker_compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --no-deps "${UPDATE_SERVICES[@]}"

if [[ " ${UPDATE_SERVICES[*]} " == *" docreader "* ]] && compose_has_service docreader; then
  wait_service_healthy docreader 180
fi
if [[ " ${UPDATE_SERVICES[*]} " == *" app "* ]] && compose_has_service app; then
  wait_service_healthy app 180
fi
if [[ " ${UPDATE_SERVICES[*]} " == *" model-hub-backend "* ]] && compose_has_service model-hub-backend; then
  wait_service_healthy model-hub-backend 180
fi
if [[ " ${UPDATE_SERVICES[*]} " == *" model-hub-ollama "* ]] && compose_has_service model-hub-ollama; then
  wait_service_healthy model-hub-ollama 180
fi

if [[ " ${UPDATE_SERVICES[*]} " == *" qwen35-9b-vllm "* || " ${UPDATE_SERVICES[*]} " == *" bge-m3-vllm "* ]]; then
  log "model service changed; waiting for model readiness before reparse"
fi

if [[ (" ${UPDATE_SERVICES[*]} " == *" app "* || " ${UPDATE_SERVICES[*]} " == *" docreader "* || " ${UPDATE_SERVICES[*]} " == *" qwen35-9b-vllm "* || " ${UPDATE_SERVICES[*]} " == *" bge-m3-vllm "* || "$CONFIG_CHANGED" == "1") && "${WEKNORA_TRIGGER_REPARSE_AFTER_DEPLOY:-true}" != "false" && -x "${ROOT_DIR}/trigger-reparse-incomplete.sh" ]]; then
  log "triggering full-document reparse for incomplete knowledge"
  ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" "${ROOT_DIR}/trigger-reparse-incomplete.sh"
fi
