# Flynn plugin discovery

[![coverage](.github/badges/coverage.svg)](https://github.com/randy-girard/flynn-plugin-discovery/actions/workflows/ci.yml)

Cluster peer-discovery API for Flynn (`kind: app`). This is a Flynn plugin,
not a Docker Compose project: install it with **`flynn-host plugin install`**.

There is no public hosted discovery service. Use this plugin when you want
the same HTTP `/clusters` API **on your cluster**, typically after a
single-node bootstrap, so extra nodes can join without a separate discovery
host. `flynn-host init --init-discovery` also works against any compatible
API when you set `DISCOVERY_SERVER` (for example `https://discovery.example.com`).

```text
sudo flynn-host plugin install discovery
sudo flynn-host plugin install ../flynn-plugin-discovery
sudo flynn-host plugin install https://github.com/randy-girard/flynn-plugin-discovery.git
sudo flynn-host plugin uninstall discovery
```

Install attaches postgres, deploys the app, and adds
`https://discovery.${CLUSTER_DOMAIN}`. After the wait URL is up, `hooks.ready`
registers this host and prints the join token (also written to
`/etc/flynn/discovery-token`). This is a system app: it does not add a
`flynn discovery` command.

On additional nodes (Flynn installed, not yet started):

```text
sudo flynn-host init --discovery "$(cat /etc/flynn/discovery-token)"
sudo systemctl start flynn-host
```

Copy the token from the first node. `DISCOVERY_SERVER=https://discovery.${CLUSTER_DOMAIN} flynn-host init --init-discovery` against this plugin reuses the same cluster token.

## Layout

```text
flynn-plugin.json     Install contract (postgres, HTTP route, wait, hooks)
cmd/discovery/        HTTP API (same /clusters contract flynn-host already uses)
internal/server/      net/http handlers
internal/store/       memory (tests) and postgres
migrations/           schema
script/plugin-build   Squashfs image for GitHub Releases
script/install.sh     Log the public URL
script/ready.sh       Mint token + register this flynn-host after wait
script/uninstall.sh   Remove /etc/flynn/discovery-token
.github/workflows/    CI (coverage badge) and Build and Release
```

## Develop

```text
./script/run-unit-tests
```
