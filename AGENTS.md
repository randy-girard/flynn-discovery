# Agent notes

This is the Flynn **discovery** plugin (`kind: app`). Go 1.24, `GOFLAGS=-mod=mod`.
Images are Linux squashfs layered on Flynn’s published ubuntu-noble. There is
no Vite UI. `compose.yaml` is local development only (Postgres + Air) and is
not part of `flynn-host plugin install`.

Install deploys `https://discovery.${CLUSTER_DOMAIN}` and a postgres resource.
`hooks.ready` (after wait) registers the current flynn-host and prints the
join token. This is a `kind: app` system plugin: it must not publish a user
`flynn` command. Operators use `flynn-host plugin`.

## Tests are required

Do not land behavior without tests in the **same change**.

- New or changed Go logic: `*_test.go` next to the code (`go test ./...`).
- `cmd/plugin-build` (manifest, Flynn base selection, layer verify): unit tests in `cmd/plugin-build/*_test.go`.
- Install/ready hooks: keep `script/ready.sh` covered by `internal/server/ready_script_test.go`. `plugin-build` must copy declared hooks into `dist/` as flat GitHub asset names (`script/install.sh` → `script-install.sh`, `script/ready.sh` → `script-ready.sh`, `script/uninstall.sh` → `script-uninstall.sh`).
- `compose.yaml` / `dev/` are local-only; keep `script/test-compose.sh` in sync if ports or the Air command change.
- Run `./script/run-unit-tests` (native on Linux; Docker on macOS). `gofmt -s` must be clean. Unit tests write HTML coverage under `coverage/` (gitignored).

Skip tests only when the change cannot regress (typo in comments, LICENSE). Say so in the commit body.

## Semantic git commits

Use [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(optional-scope): <imperative summary>
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `ci`, `build`, `chore`.

- One concern per commit. Do not mix a feature and an unrelated cleanup.
- Subject is why it matters, not a file list. No trailing period required; keep it to one line.
- Put tests in the same commit as the behavior they cover (`feat`/`fix` with tests), not a later `test:` dump unless the commit is tests-only.
- Do not commit `dist/`, `.plugin-build-cache/`, or `script/docker/dev/.image-built`.

## Plugin contract

- Do not rebuild Ubuntu from a cloud image; `plugin-build` must pull Flynn’s ubuntu-noble layer.
- Pin Flynn with `build.base.version` / `-flynn-version` for published releases; `latest` is for local builds.
- Import Flynn APIs as `github.com/randy-girard/flynn/...`. `go.mod` must `require github.com/randy-girard/flynn`. Do not vendor Flynn and do not `replace` it with a sibling `../flynn`.
- Keep the flynn-host discovery HTTP contract (`POST /clusters`, `POST|GET /clusters/:id/instances`).
- `flynn-host plugin install` is operator-only; the user `flynn` CLI does not install plugins.
