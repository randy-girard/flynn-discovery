#!/usr/bin/env bash
# Guard the local Compose stack so docker compose up stays documented and wired.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

need() {
  local needle=$1 file=$2
  if ! grep -qE -e "${needle}" "${file}"; then
    echo "missing /${needle}/ in ${file}" >&2
    exit 1
  fi
}

need 'name: flynn-discovery-dev' compose.yaml
need 'dockerfile: dev/Dockerfile.discovery' compose.yaml
need 'image: postgres:16-alpine' compose.yaml
need 'dev/air.toml' dev/Dockerfile.discovery
need '3081:3081' compose.yaml
need '5433:5432' compose.yaml
need 'DATABASE_URL' compose.yaml
need 'cmd/discovery' dev/air.toml
need 'docker compose up --build' README.md
echo "ok discovery compose stack"
