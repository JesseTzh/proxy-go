#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if command -v pnpm >/dev/null 2>&1; then
  (cd web && pnpm install --frozen-lockfile=false && pnpm build)
elif command -v npm >/dev/null 2>&1; then
  (cd web && npm install && npm run build)
else
  echo "pnpm or npm is required to build frontend" >&2
  exit 1
fi

mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/proxy-go ./cmd/server
sha256sum dist/proxy-go > dist/proxy-go.sha256
