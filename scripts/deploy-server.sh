#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/root/proxy-go}"
SERVICE_NAME="${SERVICE_NAME:-proxy-go}"

cd "${APP_DIR}"

docker compose pull "${SERVICE_NAME}"
docker compose up -d --remove-orphans "${SERVICE_NAME}"
docker image prune -f
