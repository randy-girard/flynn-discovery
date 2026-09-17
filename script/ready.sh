#!/usr/bin/env bash
# Runs on the cluster host after the discovery app is serving. Reuses the
# default cluster token, registers this flynn-host, and prints the command
# used to join additional nodes.
#
# Host systemd-resolved does not serve *.discoverd (Flynn's wait client does).
# Resolve overlay names through the discoverd HTTP API and curl --resolve, then
# POST the instance against that internal URL. The token file still stores the
# public cluster URL for flynn-host init --discovery.
set -euo pipefail

: "${FLYNN_PLUGIN_NAME:?}"

internal="${DISCOVERY_INTERNAL_URL:-http://discovery.discoverd}"
discoverd_api="${DISCOVERD_URL:-http://127.0.0.1:1111}"
public="${URL:-}"
if [[ -z "${public}" && -n "${CLUSTER_DOMAIN:-}" ]]; then
  public="https://discovery.${CLUSTER_DOMAIN}"
fi
token_file="${DISCOVERY_TOKEN_FILE:-/etc/flynn/discovery-token}"
host_status_url="${FLYNN_HOST_STATUS_URL:-http://127.0.0.1:1113/host/status}"

# Sets curl_url and curl_resolve for $1 (empty resolve when the host is not
# *.discoverd, so httptest URLs in unit tests keep working).
discovery_curl_args() {
  local raw=$1
  eval "$(DISCOVERD_URL="${discoverd_api}" python3 - "${raw}" <<'PY'
import json, os, shlex, sys, urllib.parse, urllib.request
raw = sys.argv[1]
u = urllib.parse.urlparse(raw)
host = u.hostname or ""
resolve = ""
if host.endswith(".discoverd"):
    svc = host[: -len(".discoverd")]
    api = os.environ.get("DISCOVERD_URL", "http://127.0.0.1:1111").rstrip("/")
    last = None
    for _ in range(30):
        try:
            with urllib.request.urlopen("%s/services/%s/instances" % (api, svc), timeout=5) as resp:
                inst = json.load(resp)
            if inst:
                addr = (inst[0] or {}).get("addr") or ""
                if ":" not in addr:
                    raise SystemExit("bad discoverd addr for %s: %r" % (svc, addr))
                ip, port = addr.rsplit(":", 1)
                url_port = str(u.port or port)
                resolve = "%s:%s:%s" % (host, url_port, ip)
                if u.port is None:
                    raw = urllib.parse.urlunparse(u._replace(netloc="%s:%s" % (host, url_port)))
                break
            last = "no discoverd instances for %s" % svc
        except Exception as e:
            last = str(e)
        import time
        time.sleep(1)
    else:
        raise SystemExit(last or "lookup %s failed" % host)
print("curl_url=%s" % shlex.quote(raw))
print("curl_resolve=%s" % shlex.quote(resolve))
PY
)"
}

discovery_curl() {
  local url=$1
  shift
  discovery_curl_args "${url}"
  local args=(-fsS)
  if [[ -n "${curl_resolve}" ]]; then
    args+=(--resolve "${curl_resolve}")
  fi
  curl "${args[@]}" "$@" "${curl_url}"
}

well_known="$(discovery_curl "${internal}/.well-known/cluster")"
token="$(WELL_KNOWN="${well_known}" python3 -c 'import json,os; print(json.loads(os.environ["WELL_KNOWN"])["data"]["url"])')"
cluster_id="$(WELL_KNOWN="${well_known}" python3 -c 'import json,os; print(json.loads(os.environ["WELL_KNOWN"])["data"]["id"])')"
if [[ -z "${cluster_id}" ]]; then
  echo "discovery well-known cluster id is empty" >&2
  exit 1
fi
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
discovery_curl_args "${internal}/clusters/${cluster_id}/instances"
post_args=(-sS -o "${tmp}" -w '%{http_code}' -X POST -H 'Content-Type: application/json' -d "${payload}")
if [[ -n "${curl_resolve}" ]]; then
  post_args+=(--resolve "${curl_resolve}")
fi
http_code="$(curl "${post_args[@]}" "${curl_url}")"
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
