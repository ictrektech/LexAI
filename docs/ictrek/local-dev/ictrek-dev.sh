#!/bin/bash
# tc232 fast-development helper kept outside upstream-owned scripts.

set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/../../.." && pwd )"
ENV_FILE="$PROJECT_ROOT/.env"
DEV_MODEL_CONFIG="docs/ictrek/local-dev/config/builtin_models.tc232.dev.yaml"

VLLM_CONTAINER="${ICTREK_DEV_VLLM_CONTAINER:-qwen35-9b-awq-vllm}"
VLLM_IMAGE="${ICTREK_DEV_VLLM_IMAGE:-vllm/vllm-openai:v0.18.1-cu130}"
VLLM_MODEL_DIR="${ICTREK_DEV_VLLM_MODEL_DIR:-/data/jhu/models/hf/QuantTrio--Qwen3.5-9B-AWQ}"
VLLM_PORT="${ICTREK_DEV_VLLM_PORT:-38118}"
VLLM_SERVED_MODEL="${ICTREK_DEV_VLLM_SERVED_MODEL:-qwen3.5-9b-awq}"
VLLM_BASE_URL="${ICTREK_DEV_VLLM_BASE_URL:-http://localhost:${VLLM_PORT}/v1}"
VLLM_HF_HOME="${ICTREK_DEV_VLLM_HF_HOME:-/tmp/hf-home}"
OLLAMA_PORT="${ICTREK_DEV_OLLAMA_PORT:-21436}"
OLLAMA_BASE_URL_VALUE="${ICTREK_DEV_OLLAMA_BASE_URL:-http://localhost:${OLLAMA_PORT}}"

log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[ERROR]${NC} $1"
}

show_help() {
    cat <<EOF
WeKnora ictrek tc232 fast-development helper

Usage:
  $0 setup       Create/update .env values for tc232 fast dev
  $0 start-vllm  Start or reuse qwen35-9b-awq vLLM on localhost:${VLLM_PORT}
  $0 app         Start the Go backend with tc232 dev port overrides
  $0 check       Check tc232 fast-development dependencies
  $0 help        Show this help

Common flow:
  ./docs/ictrek/local-dev/ictrek-dev.sh setup
  make dev-start DEV_ARGS="--no-langfuse --neo4j"
  ./docs/ictrek/local-dev/ictrek-dev.sh start-vllm
  ./docs/ictrek/local-dev/ictrek-dev.sh app
  make dev-frontend
EOF
}

load_env_if_exists() {
    if [ -f "$ENV_FILE" ]; then
        set -a
        # shellcheck source=/dev/null
        source "$ENV_FILE"
        set +a
    fi
}

refresh_runtime_config() {
    VLLM_CONTAINER="${ICTREK_DEV_VLLM_CONTAINER:-qwen35-9b-awq-vllm}"
    VLLM_IMAGE="${ICTREK_DEV_VLLM_IMAGE:-vllm/vllm-openai:v0.18.1-cu130}"
    VLLM_MODEL_DIR="${ICTREK_DEV_VLLM_MODEL_DIR:-/data/jhu/models/hf/QuantTrio--Qwen3.5-9B-AWQ}"
    VLLM_PORT="${ICTREK_DEV_VLLM_PORT:-38118}"
    VLLM_SERVED_MODEL="${ICTREK_DEV_VLLM_SERVED_MODEL:-qwen3.5-9b-awq}"
    VLLM_BASE_URL="${ICTREK_DEV_VLLM_BASE_URL:-http://localhost:${VLLM_PORT}/v1}"
    VLLM_HF_HOME="${ICTREK_DEV_VLLM_HF_HOME:-/tmp/hf-home}"
    OLLAMA_PORT="${ICTREK_DEV_OLLAMA_PORT:-21436}"
    OLLAMA_BASE_URL_VALUE="${ICTREK_DEV_OLLAMA_BASE_URL:-http://localhost:${OLLAMA_PORT}}"
}

ensure_env_file() {
    cd "$PROJECT_ROOT"
    if [ -f "$ENV_FILE" ]; then
        return 0
    fi
    if [ ! -f "$PROJECT_ROOT/.env.example" ]; then
        log_error ".env and .env.example are both missing"
        return 1
    fi
    cp "$PROJECT_ROOT/.env.example" "$ENV_FILE"
    log_success "Created .env from .env.example"
}

set_env_value() {
    local key="$1"
    local value="$2"
    local tmp

    tmp="$(mktemp)"
    if grep -Eq "^[[:space:]]*#?[[:space:]]*${key}=" "$ENV_FILE"; then
        awk -v key="$key" -v value="$value" '
            BEGIN { done = 0 }
            $0 ~ "^[[:space:]]*#?[[:space:]]*" key "=" {
                if (!done) {
                    print key "=" value
                    done = 1
                }
                next
            }
            { print }
            END {
                if (!done) {
                    print key "=" value
                }
            }
        ' "$ENV_FILE" > "$tmp"
        mv "$tmp" "$ENV_FILE"
    else
        rm -f "$tmp"
        printf "\n%s=%s\n" "$key" "$value" >> "$ENV_FILE"
    fi
}

get_env_value() {
    local key="$1"
    grep -E "^[[:space:]]*${key}=" "$ENV_FILE" 2>/dev/null | tail -n 1 | sed 's/^[^=]*=//'
}

ensure_csv_env_values() {
    local key="$1"
    shift
    local current
    local item

    current="$(get_env_value "$key")"
    for item in "$@"; do
        if [ -z "$current" ]; then
            current="$item"
            continue
        fi
        case ",$current," in
            *",$item,"*) ;;
            *) current="${current},${item}" ;;
        esac
    done
    set_env_value "$key" "$current"
}

setup_env() {
    cd "$PROJECT_ROOT"
    ensure_env_file
    load_env_if_exists
    refresh_runtime_config

    if [ ! -f "$PROJECT_ROOT/$DEV_MODEL_CONFIG" ]; then
        log_error "Missing model config: $DEV_MODEL_CONFIG"
        return 1
    fi

    set_env_value "DB_PORT" "${DB_PORT:-15432}"
    set_env_value "REDIS_PORT" "${REDIS_PORT:-6380}"
    set_env_value "DOCREADER_PORT" "${DOCREADER_PORT:-15051}"
    set_env_value "WEKNORA_SINGLE_USER_MODE" "true"
    set_env_value "BUILTIN_MODELS_CONFIG" "$DEV_MODEL_CONFIG"
    set_env_value "ICTREK_DEV_VLLM_BASE_URL" "$VLLM_BASE_URL"
    set_env_value "ICTREK_DEV_OLLAMA_BASE_URL" "$OLLAMA_BASE_URL_VALUE"
    set_env_value "OLLAMA_BASE_URL" "$OLLAMA_BASE_URL_VALUE"
    ensure_csv_env_values "SSRF_WHITELIST" "localhost" "127.0.0.1"

    set_env_value "ENABLE_GRAPH_RAG" "true"
    set_env_value "NEO4J_ENABLE" "true"
    set_env_value "NEO4J_URI" "bolt://localhost:7687"
    set_env_value "NEO4J_USERNAME" "${NEO4J_USERNAME:-neo4j}"
    set_env_value "NEO4J_PASSWORD" "${NEO4J_PASSWORD:-password}"

    set_env_value "LANGFUSE_WEB_PORT" "${LANGFUSE_WEB_PORT:-13000}"
    set_env_value "LANGFUSE_MINIO_S3_PORT" "${LANGFUSE_MINIO_S3_PORT:-19100}"
    set_env_value "LANGFUSE_MINIO_CONSOLE_PORT" "${LANGFUSE_MINIO_CONSOLE_PORT:-19101}"

    log_success "tc232 fast-development .env values are in place"
    echo ""
    log_info "Next:"
    echo "  make dev-start DEV_ARGS=\"--no-langfuse --neo4j\""
    echo "  ./docs/ictrek/local-dev/ictrek-dev.sh start-vllm"
    echo "  ./docs/ictrek/local-dev/ictrek-dev.sh app"
    echo "  make dev-frontend"
}

check_docker() {
    if ! command -v docker >/dev/null 2>&1; then
        log_error "Docker is not installed"
        return 1
    fi
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker daemon is not running"
        return 1
    fi
}

ensure_docker_network() {
    local network="$1"
    if docker network inspect "$network" >/dev/null 2>&1; then
        return 0
    fi
    docker network create "$network" >/dev/null
    log_success "Created docker network: $network"
}

connect_network_if_exists() {
    local network="$1"
    if docker network inspect "$network" >/dev/null 2>&1; then
        docker network connect "$network" "$VLLM_CONTAINER" >/dev/null 2>&1 || true
    fi
}

wait_for_vllm() {
    local models_url="${VLLM_BASE_URL%/}/models"
    local max_wait="${ICTREK_DEV_VLLM_WAIT_SEC:-300}"
    local waited=0
    local interval=5

    if ! command -v curl >/dev/null 2>&1; then
        log_warning "curl is not installed; skip vLLM readiness check"
        return 0
    fi

    log_info "Waiting for vLLM: $models_url"
    while [ "$waited" -lt "$max_wait" ]; do
        if curl -fsS "$models_url" >/dev/null 2>&1; then
            log_success "vLLM is ready: $models_url"
            return 0
        fi
        sleep "$interval"
        waited=$((waited + interval))
    done

    log_warning "vLLM did not become ready in ${max_wait}s; inspect with: docker logs -f ${VLLM_CONTAINER}"
    return 1
}

start_vllm() {
    cd "$PROJECT_ROOT"
    load_env_if_exists
    refresh_runtime_config
    check_docker

    if [ ! -d "$VLLM_MODEL_DIR" ]; then
        log_error "Model directory not found: $VLLM_MODEL_DIR"
        log_error "Override with ICTREK_DEV_VLLM_MODEL_DIR=/path/to/model $0 start-vllm"
        return 1
    fi

    ensure_docker_network "lexai"

    if docker ps -a --format '{{.Names}}' | grep -Fxq "$VLLM_CONTAINER"; then
        if docker ps --format '{{.Names}}' | grep -Fxq "$VLLM_CONTAINER"; then
            log_success "Container already running: $VLLM_CONTAINER"
        else
            docker start "$VLLM_CONTAINER" >/dev/null
            log_success "Started existing container: $VLLM_CONTAINER"
        fi
        connect_network_if_exists "lexai"
        connect_network_if_exists "lexai_WeKnora-network-dev"
        wait_for_vllm || true
        return 0
    fi

    log_info "Starting ${VLLM_CONTAINER} on localhost:${VLLM_PORT}"
    docker run -d \
        --name "$VLLM_CONTAINER" \
        --gpus all \
        --ipc host \
        --network lexai \
        -p "${VLLM_PORT}:8000" \
        -v "${VLLM_MODEL_DIR}:/model:ro" \
        -e HF_HOME="$VLLM_HF_HOME" \
        "$VLLM_IMAGE" \
        --host 0.0.0.0 \
        --port 8000 \
        --model /model \
        --max-num-batched-tokens "${ICTREK_DEV_VLLM_MAX_NUM_BATCHED_TOKENS:-4096}" \
        --gpu-memory-utilization "${ICTREK_DEV_VLLM_GPU_MEMORY_UTILIZATION:-0.65}" \
        --served-model-name "$VLLM_SERVED_MODEL" \
        --trust-remote-code \
        --max-num-seqs "${ICTREK_DEV_VLLM_MAX_NUM_SEQS:-4}" \
        --enforce-eager \
        --reasoning-parser qwen3 \
        --tool-call-parser qwen3_xml \
        --enable-auto-tool-choice >/dev/null

    connect_network_if_exists "lexai_WeKnora-network-dev"
    log_success "Started container: $VLLM_CONTAINER"
    wait_for_vllm || true
}

start_app() {
    cd "$PROJECT_ROOT"
    load_env_if_exists

    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed"
        return 1
    fi

    export DB_HOST=localhost
    export DOCREADER_ADDR=localhost:${DOCREADER_PORT:-50051}
    export DOCREADER_TRANSPORT=grpc
    export MINIO_ENDPOINT=localhost:${MINIO_PORT:-9000}
    export REDIS_ADDR=localhost:${REDIS_PORT:-6379}
    export MILVUS_ADDRESS=localhost:${MILVUS_PORT:-19530}
    export NEO4J_URI=${NEO4J_URI:-bolt://localhost:7687}
    export QDRANT_HOST=localhost

    if [ -z "${LOCAL_STORAGE_BASE_DIR:-}" ] || [ "$LOCAL_STORAGE_BASE_DIR" = "/data/files" ]; then
        export LOCAL_STORAGE_BASE_DIR="$PROJECT_ROOT/.local-data/files"
    fi
    mkdir -p "$LOCAL_STORAGE_BASE_DIR"

    log_info "Starting Go backend with tc232 local-dev ports"
    log_info "Postgres: ${DB_HOST}:${DB_PORT:-5432}"
    log_info "Redis: ${REDIS_ADDR}"
    log_info "DocReader: ${DOCREADER_ADDR}"
    log_info "Neo4j: ${NEO4J_URI}"

    export CGO_CFLAGS="-Wno-deprecated-declarations -Wno-gnu-folding-constant"
    if [[ "$(uname)" == "Darwin" ]]; then
        export CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries"
    fi

    if command -v air >/dev/null 2>&1; then
        air
    else
        local ldflags
        ldflags="$(./scripts/get_version.sh ldflags) -X 'google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn'"
        go run -ldflags="$ldflags" ./cmd/server
    fi
}

check_url() {
    local name="$1"
    local url="$2"

    if ! command -v curl >/dev/null 2>&1; then
        log_warning "curl is not installed; skip ${name}"
        return 0
    fi
    if curl -fsS "$url" >/dev/null 2>&1; then
        log_success "${name}: ok (${url})"
    else
        log_warning "${name}: unavailable (${url})"
    fi
}

check_setup() {
    cd "$PROJECT_ROOT"
    local vllm_check_base
    local ollama_check_base

    if [ ! -f "$ENV_FILE" ]; then
        log_error ".env is missing; run: $0 setup"
        return 1
    fi
    if [ ! -f "$PROJECT_ROOT/$DEV_MODEL_CONFIG" ]; then
        log_error "$DEV_MODEL_CONFIG is missing"
        return 1
    fi

    load_env_if_exists
    refresh_runtime_config

    log_info "Key .env values:"
    echo "  DB_PORT=${DB_PORT:-}"
    echo "  REDIS_PORT=${REDIS_PORT:-}"
    echo "  DOCREADER_PORT=${DOCREADER_PORT:-}"
    echo "  BUILTIN_MODELS_CONFIG=${BUILTIN_MODELS_CONFIG:-}"
    echo "  ICTREK_DEV_VLLM_BASE_URL=${ICTREK_DEV_VLLM_BASE_URL:-}"
    echo "  ICTREK_DEV_OLLAMA_BASE_URL=${ICTREK_DEV_OLLAMA_BASE_URL:-}"
    echo "  OLLAMA_BASE_URL=${OLLAMA_BASE_URL:-}"
    echo "  NEO4J_ENABLE=${NEO4J_ENABLE:-}"
    echo "  NEO4J_URI=${NEO4J_URI:-}"
    echo ""

    check_docker || true
    if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        if docker ps --format '{{.Names}}' | grep -Fxq "$VLLM_CONTAINER"; then
            log_success "vLLM container: running (${VLLM_CONTAINER})"
        else
            log_warning "vLLM container is not running: ${VLLM_CONTAINER}"
        fi
        if docker ps --format '{{.Names}}' | grep -Fxq "WeKnora-neo4j-dev"; then
            log_success "Neo4j dev container: running"
        else
            log_warning "Neo4j dev container is not running"
        fi
    fi

    vllm_check_base="${ICTREK_DEV_VLLM_BASE_URL:-$VLLM_BASE_URL}"
    ollama_check_base="${ICTREK_DEV_OLLAMA_BASE_URL:-$OLLAMA_BASE_URL_VALUE}"
    check_url "vLLM models" "${vllm_check_base%/}/models"
    check_url "Ollama tags" "${ollama_check_base%/}/api/tags"
}

case "${1:-help}" in
    setup)
        setup_env
        ;;
    start-vllm)
        start_vllm
        ;;
    app)
        start_app
        ;;
    check)
        check_setup
        ;;
    help|-h|--help)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
