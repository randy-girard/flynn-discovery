#!/usr/bin/env bash
# Runs on the cluster host after the discovery app is serving. Reuses the
# default cluster token, registers this flynn-host, and prints the command
# used to join additional nodes.
set -euo pipefail

: "${FLYNN_PLUGIN_NAME:?}"

internal="${DISCOVERY_INTERNAL_URL:-http://discovery.discoverd}"
public="${URL:-}"
if [[ -z "${public}" && -n "${CLUSTER_DOMAIN:-}" ]]; then
  public="https://discovery.${CLUSTER_DOMAIN}"
fi
token_file="${DISCOVERY_TOKEN_FILE:-/etc/flynn/discovery-token}"
host_status_url="${FLYNN_HOST_STATUS_URL:-http://127.0.0.1:1113/host/status}"

well_known="$(curl -fsS "${internal}/.well-known/cluster")"
token="$(WELL_KNOWN="${well_known}" python3 -c 'import json,os; print(json.loads(os.environ["WELL_KNOWN"])["data"]["url"])')"
if [[ "${token}" == /* ]]; then
  if [[ -z "${public}" ]]; then
    echo "discovery token is relative (${token}) and URL/CLUSTER_DOMAIN is unset" >&2
    exit 1
  fi
  token="${public%/}${token}"
fi

status="$(curl -fsS "${host_status_url}")"
host_id="$(STATUS_JSON="${status}" python3 -c 'import json,os; print(json.loads(os.environ["STATUS_JSON"]).get("id",""))')"
host_url="$(STATUS_JSON="${status}" python3 -c 'import json,os; print(json.loads(os.environ["STATUS_JSON"]).get("url",""))')"
if [[ -z "${host_id}" || -z "${host_url}" ]]; then
  echo "could not read flynn-host id/url from ${host_status_url}" >&2
  exit 1
fi

payload="$(HOST_ID="${host_id}" HOST_URL="${host_url}" python3 -c 'import json,os; print(json.dumps({"data":{"name":os.environ["HOST_ID"],"url":os.environ["HOST_URL"]}}))')"
tmp="$(mktemp)"
trap 'rm -f "${tmp}"' EXIT
http_code="$(curl -sS -o "${tmp}" -w '%{http_code}' \
  -X POST "${token}/instances" \
  -H 'Content-Type: application/json' \
  -d "${payload}")"
if [[ "${http_code}" != "201" && "${http_code}" != "409" ]]; then
  echo "register instance failed: HTTP ${http_code}" >&2
  cat "${tmp}" >&2 || true
  exit 1
fi

mkdir -p "$(dirname "${token_file}")"
printf '%s\n' "${token}" > "${token_file}"

echo "discovery cluster token: ${token}"
echo "this host registered as ${host_id} (${host_url})"
echo "join additional nodes with:"
echo "  sudo flynn-host init --discovery ${token}"
if [[ -n "${public}" ]]; then
  echo "or:"
  echo "  sudo DISCOVERY_SERVER=${public} flynn-host init --init-discovery"
fi
exit 0
