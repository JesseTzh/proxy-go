#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
ARTIFACTS="${CI_ARTIFACTS_DIR:-mock-page-artifacts}"
rm -rf "$ARTIFACTS"
mkdir -p "$ARTIFACTS"
python3 -m http.server 4173 --bind 127.0.0.1 --directory web/dist >"$ARTIFACTS/server.log" 2>&1 &
SERVER_PID=$!
trap 'kill "$SERVER_PID" 2>/dev/null || true' EXIT
ready=false
for attempt in {1..30}; do
  if curl --fail --silent http://127.0.0.1:4173/ >/dev/null; then ready=true; break; fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  echo "Mock frontend server did not become available at http://127.0.0.1:4173" >&2
  exit 1
fi
CI_BASE_URL=http://127.0.0.1:4173 CI_ARTIFACTS_DIR="$ARTIFACTS" node scripts/ci/web/mock-pages.mjs
CI_ARTIFACTS_DIR="$ARTIFACTS" GALLERY_SHA="${GITHUB_SHA:-local}" node scripts/ci/web/build-mock-gallery.mjs
