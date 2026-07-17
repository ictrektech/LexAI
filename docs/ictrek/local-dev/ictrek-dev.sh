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
DEPLOY_DEV_MODEL_CONFIG="docs/ictrek/local-dev/config/builtin_models.tc232.deploy-dev.yaml"
DEPLOY_DIR="${ICTREK_DEV_DEPLOY_DIR:-/data/jhu/lexai-tc232-deploy}"
DEPLOY_ENV_FILE="${ICTREK_DEV_DEPLOY_ENV_FILE:-${DEPLOY_DIR}/.env.tc232}"
DEPLOY_COMPOSE_FILE="${ICTREK_DEV_DEPLOY_COMPOSE_FILE:-${DEPLOY_DIR}/docker-compose.tc232.yml}"
DEPLOY_DEV_COMPOSE_FILE="${ICTREK_DEV_DEPLOY_COMPOSE_FILE_OVERRIDE:-${SCRIPT_DIR}/docker-compose.tc232.deploy-dev.yml}"

VLLM_CONTAINER="${ICTREK_DEV_VLLM_CONTAINER:-qwen35-9b-awq-vllm}"
VLLM_IMAGE="${ICTREK_DEV_VLLM_IMAGE:-vllm/vllm-openai:v0.18.1-cu130}"
VLLM_MODEL_DIR="${ICTREK_DEV_VLLM_MODEL_DIR:-/data/jhu/models/hf/QuantTrio--Qwen3.5-9B-AWQ}"
VLLM_PORT="${ICTREK_DEV_VLLM_PORT:-38118}"
VLLM_SERVED_MODEL="${ICTREK_DEV_VLLM_SERVED_MODEL:-qwen3.5-9b-awq}"
VLLM_BASE_URL="${ICTREK_DEV_VLLM_BASE_URL:-http://localhost:${VLLM_PORT}/v1}"
VLLM_HF_HOME="${ICTREK_DEV_VLLM_HF_HOME:-/tmp/hf-home}"
VLLM_MAX_MODEL_LEN="${ICTREK_DEV_VLLM_MAX_MODEL_LEN:-${WEKNORA_CHAT_MODEL_CONTEXT_TOKENS:-65536}}"
VLLM_MAX_NUM_SEQS="${ICTREK_DEV_VLLM_MAX_NUM_SEQS:-${WEKNORA_MAIN_QA_MODEL_CONCURRENCY:-20}}"
VLLM_MAX_NUM_BATCHED_TOKENS="${ICTREK_DEV_VLLM_MAX_NUM_BATCHED_TOKENS:-4096}"
VLLM_GPU_MEMORY_UTILIZATION="${ICTREK_DEV_VLLM_GPU_MEMORY_UTILIZATION:-0.65}"
VLLM_ENFORCE_EAGER="${ICTREK_DEV_VLLM_ENFORCE_EAGER:-true}"
VLLM_ENABLE_PREFIX_CACHING="${ICTREK_DEV_VLLM_ENABLE_PREFIX_CACHING:-true}"
VLLM_ENABLE_CHUNKED_PREFILL="${ICTREK_DEV_VLLM_ENABLE_CHUNKED_PREFILL:-true}"
VLLM_ASYNC_SCHEDULING="${ICTREK_DEV_VLLM_ASYNC_SCHEDULING:-false}"
VLLM_KV_CACHE_DTYPE="${ICTREK_DEV_VLLM_KV_CACHE_DTYPE:-}"
VLLM_ATTENTION_BACKEND="${ICTREK_DEV_VLLM_ATTENTION_BACKEND:-}"
VLLM_MOE_BACKEND="${ICTREK_DEV_VLLM_MOE_BACKEND:-}"
VLLM_LOAD_FORMAT="${ICTREK_DEV_VLLM_LOAD_FORMAT:-}"
BGE_VLLM_HOST_PORT="${BGE_VLLM_HOST_PORT:-32223}"
BGE_VLLM_BASE_URL="${ICTREK_DEV_BGE_VLLM_BASE_URL:-http://localhost:${BGE_VLLM_HOST_PORT}/v1}"
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
  $0 setup             Create/update .env values for tc232 fast dev
  $0 deploy-setup      Copy deploy .env.tc232 to root .env and apply dev overrides
  $0 deploy-start      Start deploy-dir infra with host dev ports
  $0 deploy-stop       Stop deploy-dir infra used by deploy-start
  $0 deploy-app        Run deploy-setup, then start the Go backend from source
  $0 start-vllm        Start or reuse qwen35-9b-awq vLLM on localhost:${VLLM_PORT}
  $0 app               Start the Go backend with tc232 dev port overrides
  $0 check             Check tc232 fast-development dependencies
  $0 deploy-check      Check deploy-parity dependencies
  $0 help              Show this help

Common flow:
  ./docs/ictrek/local-dev/ictrek-dev.sh setup
  make dev-start DEV_ARGS="--no-langfuse --neo4j"
  ./docs/ictrek/local-dev/ictrek-dev.sh start-vllm
  ./docs/ictrek/local-dev/ictrek-dev.sh app
  make dev-frontend

Deploy-parity flow:
  ./docs/ictrek/local-dev/ictrek-dev.sh deploy-setup
  ./docs/ictrek/local-dev/ictrek-dev.sh deploy-start
  ./docs/ictrek/local-dev/ictrek-dev.sh deploy-app
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
    VLLM_MAX_MODEL_LEN="${ICTREK_DEV_VLLM_MAX_MODEL_LEN:-${WEKNORA_CHAT_MODEL_CONTEXT_TOKENS:-65536}}"
    VLLM_MAX_NUM_SEQS="${ICTREK_DEV_VLLM_MAX_NUM_SEQS:-${WEKNORA_MAIN_QA_MODEL_CONCURRENCY:-20}}"
    VLLM_MAX_NUM_BATCHED_TOKENS="${ICTREK_DEV_VLLM_MAX_NUM_BATCHED_TOKENS:-4096}"
    VLLM_GPU_MEMORY_UTILIZATION="${ICTREK_DEV_VLLM_GPU_MEMORY_UTILIZATION:-0.65}"
    VLLM_ENFORCE_EAGER="${ICTREK_DEV_VLLM_ENFORCE_EAGER:-true}"
    VLLM_ENABLE_PREFIX_CACHING="${ICTREK_DEV_VLLM_ENABLE_PREFIX_CACHING:-true}"
    VLLM_ENABLE_CHUNKED_PREFILL="${ICTREK_DEV_VLLM_ENABLE_CHUNKED_PREFILL:-true}"
    VLLM_ASYNC_SCHEDULING="${ICTREK_DEV_VLLM_ASYNC_SCHEDULING:-false}"
    VLLM_KV_CACHE_DTYPE="${ICTREK_DEV_VLLM_KV_CACHE_DTYPE:-}"
    VLLM_ATTENTION_BACKEND="${ICTREK_DEV_VLLM_ATTENTION_BACKEND:-}"
    VLLM_MOE_BACKEND="${ICTREK_DEV_VLLM_MOE_BACKEND:-}"
    VLLM_LOAD_FORMAT="${ICTREK_DEV_VLLM_LOAD_FORMAT:-}"
    BGE_VLLM_HOST_PORT="${BGE_VLLM_HOST_PORT:-32223}"
    BGE_VLLM_BASE_URL="${ICTREK_DEV_BGE_VLLM_BASE_URL:-http://localhost:${BGE_VLLM_HOST_PORT}/v1}"
    OLLAMA_PORT="${ICTREK_DEV_OLLAMA_PORT:-21436}"
    OLLAMA_BASE_URL_VALUE="${ICTREK_DEV_OLLAMA_BASE_URL:-http://localhost:${OLLAMA_PORT}}"
}

is_enabled() {
    case "$1" in
        1|true|TRUE|yes|YES|on|ON) return 0 ;;
        *) return 1 ;;
    esac
}

resolve_vllm_model_dir() {
    local candidate="$1"
    local resolved

    if [ -f "$candidate/config.json" ]; then
        printf "%s\n" "$candidate"
        return 0
    fi

    resolved="$(find "$candidate/snapshots" \
        -mindepth 1 -maxdepth 1 -type d -exec test -f '{}/config.json' ';' \
        -print 2>/dev/null | sort | tail -n 1 || true)"
    if [ -n "$resolved" ]; then
        printf "%s\n" "$resolved"
        return 0
    fi

    return 1
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
    grep -E "^[[:space:]]*${key}=" "$ENV_FILE" 2>/dev/null | tail -n 1 | sed 's/^[^=]*=//' || true
}

remove_env_values() {
    local tmp key
    tmp="$(mktemp)"
    awk -F= -v keys="$*" '
        BEGIN {
            split(keys, arr, " ")
            for (i in arr) {
                remove[arr[i]] = 1
            }
        }
        {
            line = $0
            key = $1
            sub(/^[[:space:]]+/, "", key)
            sub(/[[:space:]]+$/, "", key)
            if (line ~ /^[[:space:]]*#/ || line !~ /=/ || !(key in remove)) {
                print line
            }
        }
    ' "$ENV_FILE" > "$tmp"
    mv "$tmp" "$ENV_FILE"
}

append_env_value() {
    local key="$1"
    local value="$2"
    printf "%s=%s\n" "$key" "$value" >> "$ENV_FILE"
}

get_deploy_env_value() {
    local key="$1"
    local default_value="${2:-}"

    if [ ! -r "$DEPLOY_ENV_FILE" ]; then
        printf "%s\n" "$default_value"
        return 0
    fi

    awk -F= -v key="$key" -v default_value="$default_value" '
        $1 == key {
            sub(/^[^=]*=/, "")
            print
            found = 1
            exit
        }
        END {
            if (!found) {
                print default_value
            }
        }
    ' "$DEPLOY_ENV_FILE"
}

find_bge_hf_models_dir() {
    local deploy_hf_dir
    local candidate
    deploy_hf_dir="$(get_deploy_env_value MODEL_HUB_HF_MODELS_DIR /data/jhu/models/huggingface)"

    for candidate in \
        "${ICTREK_DEV_BGE_VLLM_HF_MODELS_DIR:-}" \
        "$deploy_hf_dir" \
        /data/jhu/cache/huggingface \
        /data/jhu/models/huggingface; do
        if [ -z "$candidate" ]; then
            continue
        fi
        if find "$candidate/hub/models--BAAI--bge-m3/snapshots" \
            -mindepth 1 -maxdepth 1 -type d -exec test -f '{}/config.json' ';' \
            -print -quit 2>/dev/null | grep -q .; then
            printf "%s\n" "$candidate"
            return 0
        fi
    done

    printf "%s\n" "$deploy_hf_dir"
}

require_deploy_files() {
    if [ ! -d "$DEPLOY_DIR" ]; then
        log_error "Deploy directory not found: $DEPLOY_DIR"
        log_error "Override with ICTREK_DEV_DEPLOY_DIR=/path/to/lexai-tc232-deploy"
        return 1
    fi
    if [ ! -r "$DEPLOY_ENV_FILE" ]; then
        log_error "Deploy env is not readable: $DEPLOY_ENV_FILE"
        log_error "Make it readable by this user, or override ICTREK_DEV_DEPLOY_ENV_FILE"
        return 1
    fi
    if [ ! -f "$DEPLOY_COMPOSE_FILE" ]; then
        log_error "Deploy compose file not found: $DEPLOY_COMPOSE_FILE"
        return 1
    fi
    if [ ! -f "$DEPLOY_DEV_COMPOSE_FILE" ]; then
        log_error "Deploy-dev compose override not found: $DEPLOY_DEV_COMPOSE_FILE"
        return 1
    fi
    if [ ! -f "$PROJECT_ROOT/$DEPLOY_DEV_MODEL_CONFIG" ]; then
        log_error "Deploy-dev model config not found: $DEPLOY_DEV_MODEL_CONFIG"
        return 1
    fi
}

ensure_deploy_files_dir() {
    local files_dir="${ICTREK_DEV_DEPLOY_FILES_DIR:-${DEPLOY_DIR}/data/files}"
    if [ -d "$files_dir" ] && [ -w "$files_dir" ]; then
        return 0
    fi
    log_error "Local file storage directory is not writable: $files_dir"
    log_info "Create/fix it once with:"
    echo "  sudo mkdir -p \"$files_dir\""
    echo "  sudo chown -R $(id -u):$(id -g) \"$files_dir\""
    return 1
}

docker_compose() {
    local output rc
    set +e
    output="$(docker compose "$@" 2>&1)"
    rc=$?
    set -e
    if [ -n "$output" ]; then
        printf "%s\n" "$output"
    fi
    if [ "$rc" -ne 0 ] && printf "%s" "$output" | grep -q "panic: runtime error: invalid memory address"; then
        if printf "%s" "$output" | grep -q "error while interpolating"; then
            return "$rc"
        fi
        log_warning "docker compose hit a known plugin shutdown panic; continuing"
        return 0
    fi
    if [ "$rc" -eq 0 ]; then
        return 0
    fi
    if command -v docker-compose >/dev/null 2>&1; then
        docker-compose "$@"
        return $?
    fi
    log_error "Docker Compose is not installed"
    return 1
}

stop_deploy_runtime_containers() {
    local project service cids
    project="$(get_deploy_env_value COMPOSE_PROJECT_NAME lexai-tc232)"
    for service in frontend app deploy-updater; do
        cids="$(docker ps -q \
            --filter "label=com.docker.compose.project=${project}" \
            --filter "label=com.docker.compose.service=${service}")"
        if [ -n "$cids" ]; then
            log_info "Stopping deploy container service=${service}"
            docker stop $cids >/dev/null
        fi
    done
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
    set_env_value "WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL" "${WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL:-admin@lexai.local}"
    set_env_value "BUILTIN_MODELS_CONFIG" "$DEV_MODEL_CONFIG"
    set_env_value "ICTREK_DEV_VLLM_BASE_URL" "$VLLM_BASE_URL"
    set_env_value "ICTREK_DEV_VLLM_MAX_MODEL_LEN" "$VLLM_MAX_MODEL_LEN"
    set_env_value "ICTREK_DEV_VLLM_MAX_NUM_SEQS" "$VLLM_MAX_NUM_SEQS"
    set_env_value "ICTREK_DEV_VLLM_GPU_MEMORY_UTILIZATION" "$VLLM_GPU_MEMORY_UTILIZATION"
    set_env_value "ICTREK_DEV_VLLM_MAX_NUM_BATCHED_TOKENS" "$VLLM_MAX_NUM_BATCHED_TOKENS"
    set_env_value "WEKNORA_CHAT_MODEL_CONTEXT_TOKENS" "$VLLM_MAX_MODEL_LEN"
    set_env_value "WEKNORA_MAIN_QA_MODEL_CONCURRENCY" "$VLLM_MAX_NUM_SEQS"
    set_env_value "WEKNORA_MODEL_MAX_CONCURRENCY" "${WEKNORA_MODEL_MAX_CONCURRENCY:-14}"
    set_env_value "WEKNORA_CHAT_RESERVED_CONCURRENCY" "${WEKNORA_CHAT_RESERVED_CONCURRENCY:-6}"
    set_env_value "WEKNORA_CHAT_CONTEXT_SAFETY_TOKENS" "${WEKNORA_CHAT_CONTEXT_SAFETY_TOKENS:-768}"
    set_env_value "WEKNORA_CONVERSATION_MAX_COMPLETION_TOKENS" "${WEKNORA_CONVERSATION_MAX_COMPLETION_TOKENS:-24576}"
    set_env_value "WEKNORA_AGENT_FINAL_ANSWER_MAX_TOKENS" "${WEKNORA_AGENT_FINAL_ANSWER_MAX_TOKENS:-24576}"
    set_env_value "WEKNORA_ASYNQ_CORE_CONCURRENCY" "${WEKNORA_ASYNQ_CORE_CONCURRENCY:-4}"
    set_env_value "WEKNORA_ASYNQ_POSTPROCESS_CONCURRENCY" "${WEKNORA_ASYNQ_POSTPROCESS_CONCURRENCY:-2}"
    set_env_value "WEKNORA_ASYNQ_ENRICHMENT_CONCURRENCY" "${WEKNORA_ASYNQ_ENRICHMENT_CONCURRENCY:-2}"
    set_env_value "WEKNORA_ASYNQ_MAINTENANCE_CONCURRENCY" "${WEKNORA_ASYNQ_MAINTENANCE_CONCURRENCY:-1}"
    set_env_value "WEKNORA_ASYNQ_SHARED_CONCURRENCY" "${WEKNORA_ASYNQ_SHARED_CONCURRENCY:-0}"
    set_env_value "WEKNORA_WIKI_ASYNQ_CONCURRENCY" "${WEKNORA_WIKI_ASYNQ_CONCURRENCY:-4}"
    set_env_value "WEKNORA_GRAPH_LLM_CONCURRENCY" "${WEKNORA_GRAPH_LLM_CONCURRENCY:-2}"
    set_env_value "WEKNORA_WIKI_INGEST_MAP_PARALLEL" "${WEKNORA_WIKI_INGEST_MAP_PARALLEL:-4}"
    set_env_value "WEKNORA_WIKI_INGEST_REDUCE_PARALLEL" "${WEKNORA_WIKI_INGEST_REDUCE_PARALLEL:-4}"
    set_env_value "BATCH_EMBED_SIZE" "${BATCH_EMBED_SIZE:-8}"
    set_env_value "CONCURRENCY_POOL_SIZE" "${CONCURRENCY_POOL_SIZE:-8}"
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

setup_deploy_env() {
    cd "$PROJECT_ROOT"
    require_deploy_files
    local ssrf_whitelist
    local bge_vllm_host_port
    local bge_vllm_base_url
    local bge_hf_models_dir
    local qwen_vllm_context
    local qwen_vllm_seqs

    if [ -f "$ENV_FILE" ]; then
        cp "$ENV_FILE" "${ENV_FILE}.deploy-setup.bak"
        log_info "Backed up existing .env to ${ENV_FILE}.deploy-setup.bak"
    fi
    cp "$DEPLOY_ENV_FILE" "$ENV_FILE"
    log_info "Copied $DEPLOY_ENV_FILE to root .env"
    ssrf_whitelist="$(get_env_value SSRF_WHITELIST)"
    for item in localhost 127.0.0.1; do
        if [ -z "$ssrf_whitelist" ]; then
            ssrf_whitelist="$item"
        else
            case ",$ssrf_whitelist," in
                *",$item,"*) ;;
                *) ssrf_whitelist="${ssrf_whitelist},${item}" ;;
            esac
        fi
    done

    remove_env_values \
        DB_DRIVER \
        RETRIEVE_DRIVER \
        STORAGE_TYPE \
        STREAM_MANAGER_TYPE \
        DOCREADER_TRANSPORT \
        APP_PORT \
        DB_PORT \
        REDIS_PORT \
        DOCREADER_PORT \
        VITE_DEV_PROXY_TARGET \
        BUILTIN_MODELS_CONFIG \
        ICTREK_DEV_VLLM_BASE_URL \
        ICTREK_DEV_VLLM_MAX_MODEL_LEN \
        ICTREK_DEV_VLLM_MAX_NUM_SEQS \
        ICTREK_DEV_VLLM_GPU_MEMORY_UTILIZATION \
        ICTREK_DEV_VLLM_MAX_NUM_BATCHED_TOKENS \
        WEKNORA_CHAT_MODEL_CONTEXT_TOKENS \
        WEKNORA_MODEL_MAX_CONCURRENCY \
        WEKNORA_MAIN_QA_MODEL_CONCURRENCY \
        WEKNORA_CHAT_RESERVED_CONCURRENCY \
        WEKNORA_CHAT_CONTEXT_SAFETY_TOKENS \
        WEKNORA_CONVERSATION_MAX_COMPLETION_TOKENS \
        WEKNORA_AGENT_FINAL_ANSWER_MAX_TOKENS \
        WEKNORA_ASYNQ_CORE_CONCURRENCY \
        WEKNORA_ASYNQ_POSTPROCESS_CONCURRENCY \
        WEKNORA_ASYNQ_ENRICHMENT_CONCURRENCY \
        WEKNORA_ASYNQ_MAINTENANCE_CONCURRENCY \
        WEKNORA_ASYNQ_SHARED_CONCURRENCY \
        WEKNORA_WIKI_ASYNQ_CONCURRENCY \
        WEKNORA_GRAPH_LLM_CONCURRENCY \
        WEKNORA_WIKI_INGEST_MAP_PARALLEL \
        WEKNORA_WIKI_INGEST_REDUCE_PARALLEL \
        BATCH_EMBED_SIZE \
        CONCURRENCY_POOL_SIZE \
        ICTREK_DEV_BGE_VLLM_BASE_URL \
        ICTREK_DEV_BGE_VLLM_IMAGE \
        ICTREK_DEV_BGE_VLLM_HF_MODELS_DIR \
        BGE_VLLM_HOST_PORT \
        BGE_VLLM_MODEL_PATH \
        BGE_VLLM_SERVED_MODEL_NAME \
        BGE_VLLM_GPU_MEMORY_UTILIZATION \
        BGE_VLLM_MAX_MODEL_LEN \
        BGE_VLLM_MAX_NUM_SEQS \
        BGE_VLLM_MAX_NUM_BATCHED_TOKENS \
        ICTREK_DEV_OLLAMA_BASE_URL \
        OLLAMA_BASE_URL \
        NEO4J_URI \
        LOCAL_STORAGE_BASE_DIR \
        SSRF_WHITELIST

    {
        echo ""
        echo "# ===== local-dev overrides for /data/jhu/lexai-tc232-deploy ====="
        echo "# The lines above mirror .env.tc232. The values below adapt container"
        echo "# addresses and ports for host-side Go/Vite development."
    } >> "$ENV_FILE"
    append_env_value "DB_DRIVER" "postgres"
    append_env_value "RETRIEVE_DRIVER" "postgres"
    append_env_value "STORAGE_TYPE" "local"
    append_env_value "STREAM_MANAGER_TYPE" "redis"
    append_env_value "DOCREADER_TRANSPORT" "grpc"
    append_env_value "APP_PORT" "${ICTREK_DEV_DEPLOY_APP_PORT:-8080}"
    append_env_value "DB_PORT" "${ICTREK_DEV_DEPLOY_DB_PORT:-15432}"
    append_env_value "REDIS_PORT" "${ICTREK_DEV_DEPLOY_REDIS_PORT:-6380}"
    append_env_value "DOCREADER_PORT" "${ICTREK_DEV_DEPLOY_DOCREADER_PORT:-15051}"
    append_env_value "VITE_DEV_PROXY_TARGET" "http://localhost:${ICTREK_DEV_DEPLOY_APP_PORT:-8080}"
    append_env_value "BUILTIN_MODELS_CONFIG" "$DEPLOY_DEV_MODEL_CONFIG"
    append_env_value "ICTREK_DEV_VLLM_BASE_URL" "${ICTREK_DEV_VLLM_BASE_URL:-http://localhost:${VLLM_PORT}/v1}"
    qwen_vllm_context="${ICTREK_DEV_VLLM_MAX_MODEL_LEN:-$(get_env_value WEKNORA_CHAT_MODEL_CONTEXT_TOKENS)}"
    qwen_vllm_context="${qwen_vllm_context:-65536}"
    qwen_vllm_seqs="${ICTREK_DEV_VLLM_MAX_NUM_SEQS:-$(get_env_value WEKNORA_MAIN_QA_MODEL_CONCURRENCY)}"
    qwen_vllm_seqs="${qwen_vllm_seqs:-20}"
    append_env_value "ICTREK_DEV_VLLM_MAX_MODEL_LEN" "$qwen_vllm_context"
    append_env_value "ICTREK_DEV_VLLM_MAX_NUM_SEQS" "$qwen_vllm_seqs"
    append_env_value "ICTREK_DEV_VLLM_GPU_MEMORY_UTILIZATION" "${ICTREK_DEV_VLLM_GPU_MEMORY_UTILIZATION:-0.65}"
    append_env_value "ICTREK_DEV_VLLM_MAX_NUM_BATCHED_TOKENS" "${ICTREK_DEV_VLLM_MAX_NUM_BATCHED_TOKENS:-4096}"
    append_env_value "WEKNORA_CHAT_MODEL_CONTEXT_TOKENS" "$qwen_vllm_context"
    append_env_value "WEKNORA_MODEL_MAX_CONCURRENCY" "${WEKNORA_MODEL_MAX_CONCURRENCY:-14}"
    append_env_value "WEKNORA_MAIN_QA_MODEL_CONCURRENCY" "$qwen_vllm_seqs"
    append_env_value "WEKNORA_CHAT_RESERVED_CONCURRENCY" "${WEKNORA_CHAT_RESERVED_CONCURRENCY:-6}"
    append_env_value "WEKNORA_CHAT_CONTEXT_SAFETY_TOKENS" "${WEKNORA_CHAT_CONTEXT_SAFETY_TOKENS:-768}"
    append_env_value "WEKNORA_CONVERSATION_MAX_COMPLETION_TOKENS" "${WEKNORA_CONVERSATION_MAX_COMPLETION_TOKENS:-24576}"
    append_env_value "WEKNORA_AGENT_FINAL_ANSWER_MAX_TOKENS" "${WEKNORA_AGENT_FINAL_ANSWER_MAX_TOKENS:-24576}"
    append_env_value "WEKNORA_ASYNQ_CORE_CONCURRENCY" "${WEKNORA_ASYNQ_CORE_CONCURRENCY:-4}"
    append_env_value "WEKNORA_ASYNQ_POSTPROCESS_CONCURRENCY" "${WEKNORA_ASYNQ_POSTPROCESS_CONCURRENCY:-2}"
    append_env_value "WEKNORA_ASYNQ_ENRICHMENT_CONCURRENCY" "${WEKNORA_ASYNQ_ENRICHMENT_CONCURRENCY:-2}"
    append_env_value "WEKNORA_ASYNQ_MAINTENANCE_CONCURRENCY" "${WEKNORA_ASYNQ_MAINTENANCE_CONCURRENCY:-1}"
    append_env_value "WEKNORA_ASYNQ_SHARED_CONCURRENCY" "${WEKNORA_ASYNQ_SHARED_CONCURRENCY:-0}"
    append_env_value "WEKNORA_WIKI_ASYNQ_CONCURRENCY" "${WEKNORA_WIKI_ASYNQ_CONCURRENCY:-4}"
    append_env_value "WEKNORA_GRAPH_LLM_CONCURRENCY" "${WEKNORA_GRAPH_LLM_CONCURRENCY:-2}"
    append_env_value "WEKNORA_WIKI_INGEST_MAP_PARALLEL" "${WEKNORA_WIKI_INGEST_MAP_PARALLEL:-4}"
    append_env_value "WEKNORA_WIKI_INGEST_REDUCE_PARALLEL" "${WEKNORA_WIKI_INGEST_REDUCE_PARALLEL:-4}"
    append_env_value "BATCH_EMBED_SIZE" "${BATCH_EMBED_SIZE:-8}"
    append_env_value "CONCURRENCY_POOL_SIZE" "${CONCURRENCY_POOL_SIZE:-8}"
    bge_vllm_host_port="${BGE_VLLM_HOST_PORT:-32223}"
    bge_vllm_base_url="${ICTREK_DEV_BGE_VLLM_BASE_URL:-http://localhost:${bge_vllm_host_port}/v1}"
    append_env_value "BGE_VLLM_HOST_PORT" "$bge_vllm_host_port"
    append_env_value "ICTREK_DEV_BGE_VLLM_BASE_URL" "$bge_vllm_base_url"
    append_env_value "ICTREK_DEV_BGE_VLLM_IMAGE" "${ICTREK_DEV_BGE_VLLM_IMAGE:-vllm/vllm-openai:v0.18.1-cu130}"
    bge_hf_models_dir="$(find_bge_hf_models_dir)"
    append_env_value "ICTREK_DEV_BGE_VLLM_HF_MODELS_DIR" "$bge_hf_models_dir"
    append_env_value "BGE_VLLM_MODEL_PATH" "${BGE_VLLM_MODEL_PATH:-/data/huggingface/hub/models--BAAI--bge-m3}"
    append_env_value "BGE_VLLM_SERVED_MODEL_NAME" "${BGE_VLLM_SERVED_MODEL_NAME:-bge-m3}"
    append_env_value "BGE_VLLM_GPU_MEMORY_UTILIZATION" "${BGE_VLLM_GPU_MEMORY_UTILIZATION:-0.1}"
    append_env_value "BGE_VLLM_MAX_MODEL_LEN" "${BGE_VLLM_MAX_MODEL_LEN:-8192}"
    append_env_value "BGE_VLLM_MAX_NUM_SEQS" "${BGE_VLLM_MAX_NUM_SEQS:-16}"
    append_env_value "BGE_VLLM_MAX_NUM_BATCHED_TOKENS" "${BGE_VLLM_MAX_NUM_BATCHED_TOKENS:-8192}"
    append_env_value "ICTREK_DEV_OLLAMA_BASE_URL" "${ICTREK_DEV_OLLAMA_BASE_URL:-http://localhost:$(get_deploy_env_value OLLAMA_API_HOST_PORT 31434)}"
    append_env_value "OLLAMA_BASE_URL" "${ICTREK_DEV_OLLAMA_BASE_URL:-http://localhost:$(get_deploy_env_value OLLAMA_API_HOST_PORT 31434)}"
    append_env_value "NEO4J_URI" "bolt://localhost:$(get_deploy_env_value NEO4J_BOLT_PORT 30087)"
    append_env_value "LOCAL_STORAGE_BASE_DIR" "${ICTREK_DEV_DEPLOY_FILES_DIR:-${DEPLOY_DIR}/data/files}"
    append_env_value "SSRF_WHITELIST" "$ssrf_whitelist"

    log_success "Root .env now mirrors $DEPLOY_ENV_FILE with local-dev overrides"
    log_info "Deploy data files: ${ICTREK_DEV_DEPLOY_FILES_DIR:-${DEPLOY_DIR}/data/files}"
    log_info "Next: $0 deploy-start, then $0 deploy-app and make dev-frontend"
}

start_deploy_services() {
    require_deploy_files
    ensure_deploy_files_dir
    check_docker

    log_info "Starting deploy-dir infra with host dev ports"
    log_info "Deploy dir: $DEPLOY_DIR"
    stop_deploy_runtime_containers
    docker_compose \
        --env-file "$ENV_FILE" \
        -f "$DEPLOY_DEV_COMPOSE_FILE" \
        up -d postgres redis docreader neo4j bge-m3-vllm
}

stop_deploy_services() {
    require_deploy_files
    check_docker

    log_info "Stopping deploy-dir infra used by deploy-parity dev"
    docker_compose \
        --env-file "$ENV_FILE" \
        -f "$DEPLOY_DEV_COMPOSE_FILE" \
        stop postgres redis docreader neo4j bge-m3-vllm
}

check_deploy_setup() {
    require_deploy_files
    setup_deploy_env
    load_env_if_exists

    log_info "Deploy-parity endpoints:"
    echo "  Deploy dir: $DEPLOY_DIR"
    echo "  PostgreSQL: localhost:${DB_PORT:-15432}"
    echo "  Redis: localhost:${REDIS_PORT:-6380}"
    echo "  DocReader: localhost:${DOCREADER_PORT:-15051}"
    echo "  Neo4j: ${NEO4J_URI:-bolt://localhost:30087}"
    echo "  bge-m3 vLLM: ${ICTREK_DEV_BGE_VLLM_BASE_URL:-}"
    echo "  Ollama: ${ICTREK_DEV_OLLAMA_BASE_URL:-}"
    echo "  vLLM: ${ICTREK_DEV_VLLM_BASE_URL:-}"
    echo ""

    check_docker || true
    check_url "vLLM models" "${ICTREK_DEV_VLLM_BASE_URL%/}/models"
    check_url "bge-m3 vLLM models" "${ICTREK_DEV_BGE_VLLM_BASE_URL%/}/models"
    check_url "Ollama tags" "${ICTREK_DEV_OLLAMA_BASE_URL%/}/api/tags"
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
    local resolved_model_dir
    local vllm_args
    local extra_args

    if [ ! -d "$VLLM_MODEL_DIR" ]; then
        log_error "Model directory not found: $VLLM_MODEL_DIR"
        log_error "Override with ICTREK_DEV_VLLM_MODEL_DIR=/path/to/model $0 start-vllm"
        return 1
    fi
    if ! resolved_model_dir="$(resolve_vllm_model_dir "$VLLM_MODEL_DIR")"; then
        log_error "No config.json found under: $VLLM_MODEL_DIR"
        log_error "Point ICTREK_DEV_VLLM_MODEL_DIR at the model directory, or at an HF cache model directory with snapshots/*/config.json"
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
        log_info "Existing containers keep their original vLLM args; remove ${VLLM_CONTAINER} to apply changed tuning"
        connect_network_if_exists "lexai"
        connect_network_if_exists "lexai_WeKnora-network-dev"
        wait_for_vllm || true
        return 0
    fi

    log_info "Starting ${VLLM_CONTAINER} on localhost:${VLLM_PORT}"
    log_info "Model: ${resolved_model_dir}"
    log_info "vLLM tuning: max_model_len=${VLLM_MAX_MODEL_LEN}, max_num_seqs=${VLLM_MAX_NUM_SEQS}, gpu_memory_utilization=${VLLM_GPU_MEMORY_UTILIZATION}"
    vllm_args=(
        --host 0.0.0.0
        --port 8000
        --model /model
        --max-model-len "$VLLM_MAX_MODEL_LEN"
        --max-num-batched-tokens "$VLLM_MAX_NUM_BATCHED_TOKENS"
        --gpu-memory-utilization "$VLLM_GPU_MEMORY_UTILIZATION"
        --served-model-name "$VLLM_SERVED_MODEL"
        --trust-remote-code
        --max-num-seqs "$VLLM_MAX_NUM_SEQS"
        --reasoning-parser qwen3
        --tool-call-parser qwen3_xml
        --enable-auto-tool-choice
    )
    if [ -n "$VLLM_KV_CACHE_DTYPE" ]; then
        vllm_args+=(--kv-cache-dtype "$VLLM_KV_CACHE_DTYPE")
    fi
    if [ -n "$VLLM_ATTENTION_BACKEND" ]; then
        vllm_args+=(--attention-backend "$VLLM_ATTENTION_BACKEND")
    fi
    if [ -n "$VLLM_MOE_BACKEND" ]; then
        vllm_args+=(--moe-backend "$VLLM_MOE_BACKEND")
    fi
    if is_enabled "$VLLM_ENFORCE_EAGER"; then
        vllm_args+=(--enforce-eager)
    fi
    if is_enabled "$VLLM_ENABLE_PREFIX_CACHING"; then
        vllm_args+=(--enable-prefix-caching)
    fi
    if is_enabled "$VLLM_ENABLE_CHUNKED_PREFILL"; then
        vllm_args+=(--enable-chunked-prefill)
    fi
    if is_enabled "$VLLM_ASYNC_SCHEDULING"; then
        vllm_args+=(--async-scheduling)
    fi
    if [ -n "$VLLM_LOAD_FORMAT" ]; then
        vllm_args+=(--load-format "$VLLM_LOAD_FORMAT")
    fi
    if [ -n "${ICTREK_DEV_VLLM_EXTRA_ARGS:-}" ]; then
        read -r -a extra_args <<< "$ICTREK_DEV_VLLM_EXTRA_ARGS"
        vllm_args+=("${extra_args[@]}")
    fi
    docker run -d \
        --name "$VLLM_CONTAINER" \
        --gpus all \
        --ipc host \
        --network lexai \
        -p "${VLLM_PORT}:8000" \
        -v "${resolved_model_dir}:/model:ro" \
        -e HF_HOME="$VLLM_HF_HOME" \
        "$VLLM_IMAGE" \
        "${vllm_args[@]}" >/dev/null

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
    deploy-setup)
        setup_deploy_env
        ;;
    deploy-start)
        start_deploy_services
        ;;
    deploy-stop)
        stop_deploy_services
        ;;
    deploy-app)
        setup_deploy_env
        ensure_deploy_files_dir
        start_app
        ;;
    deploy-check)
        check_deploy_setup
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
