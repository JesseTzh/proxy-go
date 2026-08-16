#!/usr/bin/env bash
set -euo pipefail

image="${1:-}"
if [[ -z "${image}" ]]; then
  echo "usage: $0 <docker-image>" >&2
  exit 2
fi

container_name="${TEST_CONTAINER:-proxy-go-ci-startup-$$}"
volume_name="${TEST_VOLUME:-proxy-go-ci-data-$$}"

cleanup() {
  exit_code=$?
  set +e

  if [[ "${exit_code}" -ne 0 ]]; then
    echo "::group::Container logs"
    docker logs "${container_name}" 2>&1 || true
    echo "::endgroup::"
    docker inspect "${container_name}" 2>/dev/null || true
  fi

  docker rm -fv "${container_name}" >/dev/null 2>&1 || true
  docker volume rm "${volume_name}" >/dev/null 2>&1 || true
  exit "${exit_code}"
}
trap cleanup EXIT

wait_for_healthy() {
  local attempt status running

  for attempt in $(seq 1 60); do
    status="$(docker inspect --format '{{.State.Health.Status}}' "${container_name}" 2>/dev/null || true)"
    if [[ "${status}" == "healthy" ]]; then
      return 0
    fi

    running="$(docker inspect --format '{{.State.Running}}' "${container_name}" 2>/dev/null || true)"
    if [[ "${running}" != "true" ]]; then
      echo "Container stopped before becoming healthy" >&2
      return 1
    fi

    echo "Waiting for container health check (${attempt}/60): ${status:-unknown}"
    sleep 5
  done

  echo "Timed out waiting for a healthy container" >&2
  return 1
}

start_container() {
  local initial_password="${1:-}"
  local password_args=()

  if [[ -n "${initial_password}" ]]; then
    password_args+=(--env "PROXY_GO_INITIAL_PASSWORD=${initial_password}")
  fi

  docker run --detach \
    --platform linux/amd64 \
    --name "${container_name}" \
    "${password_args[@]}" \
    --volume "${volume_name}:/var/lib/proxy-go" \
    "${image}" >/dev/null

  wait_for_healthy
  docker exec "${container_name}" curl -fsS \
    http://127.0.0.1:30080/api/init/status >/dev/null
}

docker volume create "${volume_name}" >/dev/null

echo "Validating startup with a fresh database"
start_container ci-startup-password
docker exec "${container_name}" test -s /var/lib/proxy-go/proxy-go.db

echo "Validating restart with the existing database"
docker rm -fv "${container_name}" >/dev/null
start_container

echo "Application startup smoke tests passed"
