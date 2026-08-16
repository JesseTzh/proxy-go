#!/usr/bin/env sh
set -eu

if [ "${1:-}" != "--inside-container" ]; then
  image="${1:-}"
  if [ -z "${image}" ]; then
    echo "usage: $0 <docker-image>" >&2
    exit 2
  fi

  script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
  script_path="${script_dir}/$(basename -- "$0")"

  exec docker run --rm \
    --platform linux/amd64 \
    --cap-add NET_ADMIN \
    --entrypoint /bin/sh \
    --volume "${script_path}:/tmp/proxy-go-ci-smoke.sh:ro" \
    "${image}" \
    /tmp/proxy-go-ci-smoke.sh --inside-container
fi

work_dir="$(mktemp -d)"
nginx_pid=""
sing_box_pid=""
wg_interface="wgci0"
wg_config="${work_dir}/${wg_interface}.conf"

cleanup() {
  status=$?
  set +e

  if /usr/bin/wg show "${wg_interface}" >/dev/null 2>&1; then
    /usr/bin/wg-quick down "${wg_config}" >/dev/null 2>&1
  fi
  if [ -n "${sing_box_pid}" ]; then
    kill "${sing_box_pid}" >/dev/null 2>&1
    wait "${sing_box_pid}" >/dev/null 2>&1
  fi
  if [ -n "${nginx_pid}" ]; then
    kill "${nginx_pid}" >/dev/null 2>&1
    wait "${nginx_pid}" >/dev/null 2>&1
  fi

  if [ "${status}" -ne 0 ]; then
    for log_file in "${work_dir}"/*.log; do
      if [ -f "${log_file}" ]; then
        echo "===== $(basename "${log_file}") =====" >&2
        sed -n '1,200p' "${log_file}" >&2
      fi
    done
  fi

  rm -rf "${work_dir}"
  exit "${status}"
}
trap cleanup EXIT INT TERM

require_executable() {
  if [ ! -x "$1" ]; then
    echo "required executable is missing: $1" >&2
    return 1
  fi
}

wait_for_http() {
  url="$1"
  process_id="$2"
  attempts=0

  while [ "${attempts}" -lt 20 ]; do
    if ! kill -0 "${process_id}" 2>/dev/null; then
      echo "process ${process_id} stopped before ${url} became available" >&2
      return 1
    fi
    if curl -fsS --max-time 2 "${url}" >/dev/null 2>&1; then
      return 0
    fi
    attempts=$((attempts + 1))
    sleep 1
  done

  echo "timed out waiting for ${url}" >&2
  return 1
}

require_executable /usr/local/bin/nginx
require_executable /usr/local/bin/sing-box
require_executable /usr/bin/wg
require_executable /usr/bin/wg-quick
require_executable /usr/sbin/ip
require_executable /usr/sbin/iptables

echo "Testing Nginx startup"
mkdir -p "${work_dir}/nginx"
cat >"${work_dir}/nginx.conf" <<EOF
worker_processes 1;
pid ${work_dir}/nginx.pid;
error_log stderr notice;

events {
  worker_connections 64;
}

http {
  access_log off;
  server {
    listen 127.0.0.1:18080;
    location / {
      return 204;
    }
  }
}
EOF
/usr/local/bin/nginx -t -p "${work_dir}/nginx/" -c "${work_dir}/nginx.conf"
/usr/local/bin/nginx -p "${work_dir}/nginx/" -c "${work_dir}/nginx.conf" -g 'daemon off;' \
  >"${work_dir}/nginx.log" 2>&1 &
nginx_pid=$!
wait_for_http http://127.0.0.1:18080/ "${nginx_pid}"

echo "Testing sing-box startup and proxy path"
cat >"${work_dir}/sing-box.json" <<'EOF'
{
  "log": {
    "disabled": true
  },
  "inbounds": [
    {
      "type": "mixed",
      "tag": "ci-smoke-in",
      "listen": "127.0.0.1",
      "listen_port": 19080
    }
  ],
  "outbounds": [
    {
      "type": "direct",
      "tag": "ci-smoke-out"
    }
  ]
}
EOF
/usr/local/bin/sing-box check -c "${work_dir}/sing-box.json"
/usr/local/bin/sing-box run -c "${work_dir}/sing-box.json" \
  >"${work_dir}/sing-box.log" 2>&1 &
sing_box_pid=$!
attempts=0
while [ "${attempts}" -lt 20 ]; do
  if ! kill -0 "${sing_box_pid}" 2>/dev/null; then
    echo "sing-box stopped before its proxy became available" >&2
    exit 1
  fi
  if curl -fsS --max-time 2 --noproxy '' \
    --proxy http://127.0.0.1:19080 http://127.0.0.1:18080/ >/dev/null 2>&1; then
    break
  fi
  attempts=$((attempts + 1))
  sleep 1
done
if [ "${attempts}" -eq 20 ]; then
  echo "timed out waiting for the sing-box proxy" >&2
  exit 1
fi

echo "Testing WireGuard startup"
private_key="$(/usr/bin/wg genkey)"
cat >"${wg_config}" <<EOF
[Interface]
Address = 10.254.254.1/32
ListenPort = 51888
PrivateKey = ${private_key}
PostUp = iptables -I FORWARD -i %i -j ACCEPT
PostDown = iptables -D FORWARD -i %i -j ACCEPT
EOF
chmod 0600 "${wg_config}"
/usr/bin/wg-quick up "${wg_config}"
/usr/bin/wg show "${wg_interface}" >/dev/null
/usr/sbin/ip link show "${wg_interface}" >/dev/null

echo "All bundled service smoke tests passed"
