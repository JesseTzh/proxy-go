#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEV_DIR="${ROOT_DIR}/.dev/proxy-go"
CONFIG_FILE="${DEV_DIR}/config.yml"

BACKEND_ADDR="${PROXY_GO_DEV_BACKEND_ADDR:-127.0.0.1:30080}"
INTERNAL_ADDR="${PROXY_GO_DEV_INTERNAL_ADDR:-127.0.0.1:30081}"
FRONTEND_HOST="${PROXY_GO_DEV_FRONTEND_HOST:-127.0.0.1}"
FRONTEND_PORT="${PROXY_GO_DEV_FRONTEND_PORT:-5173}"
INITIAL_PASSWORD="${PROXY_GO_INITIAL_PASSWORD:-local-test-password}"
ACME_EMAIL="${PROXY_GO_ACME_EMAIL:-dev@example.com}"

backend_pid=""
frontend_pid=""

cleanup() {
  set +e
  if [[ -n "${frontend_pid}" ]] && kill -0 "${frontend_pid}" 2>/dev/null; then
    kill "${frontend_pid}" 2>/dev/null
  fi
  if [[ -n "${backend_pid}" ]] && kill -0 "${backend_pid}" 2>/dev/null; then
    kill "${backend_pid}" 2>/dev/null
  fi
  wait 2>/dev/null
}
trap cleanup EXIT INT TERM

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

port_in_use() {
  local addr="$1"
  local port="${addr##*:}"
  lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
}

require_command go
require_command pnpm
require_command lsof

if [[ "${BACKEND_ADDR}" != "127.0.0.1:30080" ]]; then
  echo "Warning: web/vite.config.ts proxies /api to http://127.0.0.1:30080." >&2
  echo "         Use PROXY_GO_DEV_BACKEND_ADDR only after updating the Vite proxy too." >&2
fi

if port_in_use "${BACKEND_ADDR}"; then
  echo "Backend port already in use: ${BACKEND_ADDR}" >&2
  exit 1
fi

if lsof -nP -iTCP:"${FRONTEND_PORT}" -sTCP:LISTEN >/dev/null 2>&1; then
  echo "Frontend port already in use: ${FRONTEND_PORT}" >&2
  exit 1
fi

mkdir -p \
  "${DEV_DIR}/data" \
  "${DEV_DIR}/logs" \
  "${DEV_DIR}/bin" \
  "${DEV_DIR}/certs" \
  "${DEV_DIR}/nginx" \
  "${DEV_DIR}/xray"

cat >"${CONFIG_FILE}" <<EOF
server:
  initial_addr: "${BACKEND_ADDR}"
  internal_addr: "${INTERNAL_ADDR}"
  public_https_port: 18443
  public_http_port: 18080
  managed_https_addr: "127.0.0.1:30443"
  start_initial_port: true
  cookie_secure: false
paths:
  data_dir: "${DEV_DIR}/data"
  log_dir: "${DEV_DIR}/logs"
  db_file: "${DEV_DIR}/data/proxy-go.db"
  cert_dir: "${DEV_DIR}/certs"
  bin_dir: "${DEV_DIR}/bin"
  nginx_conf_dir: "${DEV_DIR}/nginx"
  xray_conf_dir: "${DEV_DIR}/xray"
  web_root: "${ROOT_DIR}/web/dist"
security:
  initial_password: "${INITIAL_PASSWORD}"
  session_ttl_hours: 24
  bcrypt_cost: 10
acme:
  email: "${ACME_EMAIL}"
  directory_url: "https://acme-staging-v02.api.letsencrypt.org/directory"
  renew_before_days: 30
runtime:
  start_children: false
EOF

if [[ ! -d "${ROOT_DIR}/web/node_modules" ]]; then
  echo "web/node_modules not found. Run: cd web && pnpm install" >&2
  exit 1
fi

echo "Starting proxy-go backend: http://${BACKEND_ADDR}"
echo "Starting Vite frontend:   http://${FRONTEND_HOST}:${FRONTEND_PORT}"
echo "Login password:           ${INITIAL_PASSWORD}"
echo "Local data dir:           ${DEV_DIR}"
echo

(
  cd "${ROOT_DIR}"
  PROXY_GO_INITIAL_PASSWORD="${INITIAL_PASSWORD}" \
  PROXY_GO_ACME_EMAIL="${ACME_EMAIL}" \
  go run ./cmd/server --config "${CONFIG_FILE}"
) &
backend_pid="$!"

(
  cd "${ROOT_DIR}/web"
  pnpm dev --host "${FRONTEND_HOST}" --port "${FRONTEND_PORT}"
) &
frontend_pid="$!"

while true; do
  if ! kill -0 "${backend_pid}" 2>/dev/null; then
    wait "${backend_pid}"
    exit $?
  fi
  if ! kill -0 "${frontend_pid}" 2>/dev/null; then
    wait "${frontend_pid}"
    exit $?
  fi
  sleep 1
done
