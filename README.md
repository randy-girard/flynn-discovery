# Flynn plugin discovery

[![coverage](.github/badges/coverage.svg)](https://github.com/randy-girard/flynn-discovery/actions/workflows/ci.yml)

Cluster peer-discovery API for Flynn (`kind: app`). This is a Flynn plugin,
not a Docker Compose project: install it with **`flynn-host plugin install`**.

The hosted service at `discovery.flynn.cloud.randygirard.com` is still the
default for `flynn-host init --init-discovery`. Use this plugin when you want
that same HTTP API **on your cluster**, typically after a single-node
bootstrap, so extra nodes can join without the cloud discovery server.

```text
sudo flynn-host plugin install discovery
sudo flynn-host plugin install ../flynn-discovery
sudo flynn-host plugin install https://github.com/randy-girard/flynn-discovery.git
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
