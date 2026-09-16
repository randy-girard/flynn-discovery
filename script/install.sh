#!/usr/bin/env bash
# Runs on a cluster host during `flynn-host plugin install`, before the app
# is deployed. Token minting and host registration happen in hooks.ready
# after the wait URL succeeds.
set -euo pipefail

: "${FLYNN_PLUGIN_NAME:?}"
: "${FLYNN_PLUGIN_KIND:?}"

echo "discovery plugin hook: app=${FLYNN_PLUGIN_APP:-discovery} kind=${FLYNN_PLUGIN_KIND}"
if [[ -n "${CLUSTER_DOMAIN:-}" ]]; then
  echo "discovery URL: https://discovery.${CLUSTER_DOMAIN}"
fi
echo "after the app is up, hooks.ready will print the join token"
exit 0
