#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
XRAY_VERSION="${XRAY_VERSION:-latest}"
CACHE_DIR="${ROOT_DIR}/.cache/xray-darwin-arm64-${XRAY_VERSION}"
DEFAULT_XRAY_BIN="${CACHE_DIR}/xray"
XRAY_BIN="${XRAY_BIN:-$DEFAULT_XRAY_BIN}"
SOCKS_PORT="${SOCKS_PORT:-10808}"
TEST_URL="${TEST_URL:-https://www.google.com/}"
VLESS_URI=""
XRAY_PID=""
TEST_TMP_DIR=""

usage() {
  cat <<'USAGE'
Usage:
  scripts/test-vless-xhttp-reality.sh 'vless://...'

Environment:
  SOCKS_PORT=10808
  TEST_URL=https://www.google.com/
  XRAY_BIN=/path/to/xray
  XRAY_VERSION=latest

The script downloads Xray-core for macOS arm64 into .cache/ when XRAY_BIN is not set,
starts a temporary local SOCKS proxy, and tests the VLESS XHTTP REALITY outbound.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -lt 1 ]]; then
  usage >&2
  exit 2
fi

VLESS_URI="$1"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 2
  fi
}

need_cmd curl
need_cmd node
need_cmd unzip

detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"
  if [[ "$os" != "Darwin" || "$arch" != "arm64" ]]; then
    echo "warning: expected macOS arm64, got ${os}/${arch}" >&2
  fi
}

download_xray() {
  if [[ "$XRAY_BIN" != "$DEFAULT_XRAY_BIN" ]]; then
    if [[ ! -x "$XRAY_BIN" ]]; then
      echo "XRAY_BIN is not executable: $XRAY_BIN" >&2
      exit 2
    fi
    return
  fi

  if [[ -x "$XRAY_BIN" ]]; then
    return
  fi

  mkdir -p "$CACHE_DIR"
  local meta asset_url zip_path
  echo "Downloading Xray-core ${XRAY_VERSION} macOS arm64 release..."
  if [[ "$XRAY_VERSION" == "latest" ]]; then
    meta="$(curl -fsSL https://api.github.com/repos/XTLS/Xray-core/releases/latest)"
  else
    meta="$(curl -fsSL "https://api.github.com/repos/XTLS/Xray-core/releases/tags/v${XRAY_VERSION}")"
  fi
  asset_url="$(META_JSON="$meta" node <<'NODE'
const release = JSON.parse(process.env.META_JSON);
const assets = release.assets || [];
const asset = assets.find((item) => /Xray-macos-arm64.*\.zip$/i.test(item.name))
  || assets.find((item) => /(darwin|macos).*arm64.*\.zip$/i.test(item.name));
if (!asset) {
  console.error("No macOS arm64 Xray zip asset found in latest release.");
  console.error(assets.map((item) => item.name).join("\n"));
  process.exit(1);
}
console.log(asset.browser_download_url);
NODE
)"
  zip_path="${CACHE_DIR}/xray.zip"
  curl -fL "$asset_url" -o "$zip_path"
  rm -rf "${CACHE_DIR}/unpacked"
  mkdir -p "${CACHE_DIR}/unpacked"
  unzip -q "$zip_path" -d "${CACHE_DIR}/unpacked"
  local found
  found="$(find "${CACHE_DIR}/unpacked" -type f -name xray -perm -111 | head -n 1 || true)"
  if [[ -z "$found" ]]; then
    found="$(find "${CACHE_DIR}/unpacked" -type f -name xray | head -n 1 || true)"
  fi
  if [[ -z "$found" ]]; then
    echo "downloaded archive did not contain xray binary" >&2
    exit 1
  fi
  cp "$found" "$XRAY_BIN"
  chmod 0755 "$XRAY_BIN"
}

write_config() {
  local config_path="$1"
  VLESS_URI="$VLESS_URI" SOCKS_PORT="$SOCKS_PORT" node >"$config_path" <<'NODE'
const uri = process.env.VLESS_URI;
const socksPort = Number(process.env.SOCKS_PORT || "10808");
const u = new URL(uri);
if (u.protocol !== "vless:") throw new Error("URI must start with vless://");
const q = u.searchParams;
const required = ["pbk", "sid", "sni", "type"];
for (const key of required) {
  if (!q.get(key)) throw new Error(`missing required query parameter: ${key}`);
}
const network = q.get("type");
const config = {
  log: { loglevel: "debug" },
  inbounds: [
    {
      tag: "local-socks",
      listen: "127.0.0.1",
      port: socksPort,
      protocol: "socks",
      settings: { udp: true, auth: "noauth" }
    }
  ],
  outbounds: [
    {
      tag: "proxy",
      protocol: "vless",
      settings: {
        vnext: [
          {
            address: u.hostname,
            port: Number(u.port || "443"),
            users: [
              {
                id: decodeURIComponent(u.username),
                encryption: q.get("encryption") || "none"
              }
            ]
          }
        ]
      },
      streamSettings: {
        network,
        security: q.get("security") || "reality",
        realitySettings: {
          serverName: q.get("sni"),
          fingerprint: q.get("fp") || "chrome",
          publicKey: q.get("pbk"),
          shortId: q.get("sid"),
          spiderX: q.get("spx") || ""
        }
      }
    }
  ],
  routing: { domainStrategy: "AsIs", rules: [] }
};
if (network === "xhttp") {
  config.outbounds[0].streamSettings.xhttpSettings = {
    path: q.get("path") || "/",
    mode: q.get("mode") || "auto",
    host: q.get("host") || u.hostname
  };
}
console.log(JSON.stringify(config, null, 2));
NODE
}

cleanup() {
  local status=$?
  if [[ -n "${XRAY_PID:-}" ]]; then
    kill "$XRAY_PID" >/dev/null 2>&1 || true
    # Ensure the proxy dies immediately so it cannot leak into later tests.
    for _ in 1 2 3 4 5; do
      kill -0 "$XRAY_PID" >/dev/null 2>&1 || break
      sleep 0.1
    done
    kill -9 "$XRAY_PID" >/dev/null 2>&1 || true
  fi
  if [[ -n "${TEST_TMP_DIR:-}" ]]; then
    rm -rf "$TEST_TMP_DIR"
  fi
  exit "$status"
}

run_test() {
  local config_path log_path body_path
  TEST_TMP_DIR="$(mktemp -d)"
  trap cleanup EXIT INT TERM
  config_path="${TEST_TMP_DIR}/client.json"
  log_path="${TEST_TMP_DIR}/xray.log"
  body_path="${TEST_TMP_DIR}/body.html"
  write_config "$config_path"

  echo "Using Xray: $XRAY_BIN"
  "$XRAY_BIN" version | head -n 1
  echo "Validating generated client config..."
  "$XRAY_BIN" run -test -config "$config_path"

  echo "Starting local SOCKS proxy on 127.0.0.1:${SOCKS_PORT}..."
  "$XRAY_BIN" run -config "$config_path" >"$log_path" 2>&1 &
  XRAY_PID="$!"
  sleep 1

  if ! kill -0 "$XRAY_PID" >/dev/null 2>&1; then
    echo "Xray exited before test request. Log:" >&2
    cat "$log_path" >&2
    exit 1
  fi

  echo "Testing through proxy: $TEST_URL"
  local http_code
  http_code="$(curl --socks5-hostname "127.0.0.1:${SOCKS_PORT}" -fsSL \
    --max-time 30 -w '%{http_code}' -o "$body_path" "$TEST_URL" 2>/dev/null)" || {
    local code=$?
    echo "Proxy test failed with curl exit code ${code}." >&2
    echo "Xray log tail:" >&2
    tail -n 120 "$log_path" >&2
    exit "$code"
  }

  if [[ "$http_code" != "200" ]]; then
    echo "Proxy test failed: expected HTTP 200, got ${http_code}." >&2
    echo "Response body (first 200 bytes):" >&2
    head -c 200 "$body_path" >&2; echo >&2
    exit 1
  fi

  # Require real page content, not just an empty 2xx. Case-insensitive because
  # Google's markup varies by locale/device.
  if ! grep -iq '<html' "$body_path" || ! grep -iq 'google' "$body_path"; then
    echo "Proxy test failed: response is not the expected Google homepage." >&2
    echo "Response body (first 500 bytes):" >&2
    head -c 500 "$body_path" >&2; echo >&2
    exit 1
  fi

  local bytes
  bytes="$(wc -c < "$body_path" | tr -d ' ')"
  echo "Proxy test succeeded."
  echo "  HTTP ${http_code}, ${bytes} bytes of HTML from ${TEST_URL}"
}

detect_platform
download_xray
run_test
