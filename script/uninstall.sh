#!/usr/bin/env bash
# Runs on a cluster host during `flynn-host plugin uninstall`.
set -euo pipefail

: "${FLYNN_PLUGIN_NAME:?}"
: "${FLYNN_PLUGIN_KIND:?}"

echo "discovery plugin uninstall hook: app=${FLYNN_PLUGIN_APP:-discovery} kind=${FLYNN_PLUGIN_KIND}"
if [[ -f /etc/flynn/discovery-token ]]; then
  rm -f /etc/flynn/discovery-token
  echo "removed /etc/flynn/discovery-token"
fi
exit 0
