# Developing Libredesk

How to run Libredesk from source for development. This guide is written for
Windows (via WSL2), but every step from "Start the dependencies" onward applies
unchanged to Linux and macOS.

The stack: a Go backend (`cmd/`, `internal/`), two Vite apps in `frontend/`
(`apps/main` is the agent UI, `apps/widget` is the live-chat widget), PostgreSQL
17 and Redis 7.

## Prerequisites (Windows)

The `Makefile` relies on Unix tooling (`grep -oP`, `date -u`, `export … && cd …`,
`$(HOME)`), so it does not run under cmd.exe or PowerShell. Use WSL2.

1. **WSL2** with Ubuntu 22.04 or newer:
   ```powershell
   wsl --install -d Ubuntu
   ```
2. **Docker Desktop** with *Settings → Resources → WSL integration* enabled for
   your Ubuntu distro, so `docker` works from inside WSL.
3. Inside WSL:
   - **Go 1.25.0** (see `go.mod`).
   - **Node 20+** and `corepack enable`. `frontend/package.json` pins
     `pnpm@9.15.3`; corepack downloads that exact version on first use.
     > If corepack fails with `Cannot find matching keyid`, your corepack is
     > older than the npm signing key rotation. Either update it
     > (`npm install -g corepack@latest`) or prefix commands with
     > `COREPACK_INTEGRITY_KEYS=0`.
   - `make` and `git` (`sudo apt install make git`).
4. **Clone inside the WSL filesystem** (for example `~/code/libredesk`), never
   under `/mnt/c/...`. `node_modules` and Vite's file watching are orders of
   magnitude slower across the Windows/Linux filesystem boundary.
5. Line endings: the repo does not force `eol=lf` in `.gitattributes`, so set
   ```bash
   git config core.autocrlf input
   ```
   in the clone to keep commits LF-only.

## Start the dependencies

Only Postgres and Redis run in Docker. The `app` service in `docker-compose.yml`
uses the published image and is **not** what you want while developing.

```bash
make dev-db          # docker compose up -d db redis
```

This gives you Postgres 17 on `localhost:5432` and Redis 7 on `localhost:6379`,
both with user/password `libredesk` / `libredesk`. Stop them with `make dev-db-down`.

## Configure

```bash
cp config.sample.toml config.toml
mkdir -p uploads     # the app does NOT create this; file uploads fail without it
```

The sample config points at the Docker network hostnames. Edit `config.toml`:

| Key | Value | Why |
|---|---|---|
| `[db] host` | `"localhost"` | sample says `"db"` |
| `[redis] address` | `"localhost:6379"` | sample says `"redis:6379"` |
| `[app] encryption_key` | any **exactly 32-character** string (`openssl rand -hex 16`) | the app refuses to boot otherwise |
| `[app.server] disable_secure_cookies` | `true` | secure cookies are never sent over `http://localhost`, so login silently fails |

Any key can also be set through environment variables instead, which keeps
`config.toml` untouched. The prefix is `LIBREDESK_` and `__` separates nesting
levels (`cmd/init.go`):

```bash
export LIBREDESK_DB__HOST=localhost
export LIBREDESK_REDIS__ADDRESS=localhost:6379
export LIBREDESK_APP__SERVER__DISABLE_SECURE_COOKIES=true
export LIBREDESK_APP__ENCRYPTION_KEY=$(openssl rand -hex 16)
```

## Install the schema

```bash
LIBREDESK_SYSTEM_USER_PASSWORD='YourStrongPassword123!' \
  go run ./cmd/ --install --idempotent-install --yes --config config.toml
go run ./cmd/ --upgrade --yes --config config.toml
```

- `--install` runs `schema.sql`. It needs the `pg_trgm` extension, which the
  `postgres:17-alpine` image ships with.
- `--idempotent-install` skips the install when the schema already exists, so
  the command is safe to re-run. Without it, `--install` **wipes the database**.
- `LIBREDESK_SYSTEM_USER_PASSWORD` sets the password of the built-in `System`
  user on first install. It must be a strong password. To change it later:
  `go run ./cmd/ --set-system-user-password --config config.toml`.
- Migrations are Go files under `internal/migrations/`, applied by `--upgrade`.
  The app refuses to start while migrations are pending, so re-run `--upgrade`
  after pulling changes that add one.

## Run (two terminals)

```bash
# Terminal A — Go backend on :9000
make run-backend
```

```bash
# Terminal B — main frontend on :8000
make run-frontend-main
```

Open **http://localhost:8000** and log in as `System` with the password from
the install step.

- Vite proxies `/api`, `/ws`, `/uploads`, `/static` and `/logout` to `:9000`
  (`frontend/vite.config.js`). Override the targets with `LD_API_TARGET` /
  `LD_WS_TARGET`, and the ports with `LD_DEV_PORT` / `LD_WIDGET_DEV_PORT`.
- The live-chat widget runs on **:8001** with `make run-frontend-widget`.
- Go changes need a restart of `make run-backend`; the frontend hot-reloads.

Two things that trip people up:

- **Start the backend from the repository root.** In dev the binary has no
  embedded assets, so stuffbin falls back to a local filesystem rooted at the
  working directory (`i18n`, `static`, `frontend/dist/*`). From anywhere else
  it fails to find the i18n files.
- **Opening :9000 directly returns a 404** for the HTML. The UI is served by
  Vite on :8000; the backend only serves built assets from `frontend/dist`,
  which don't exist until you run `make frontend-build`.

## Tests

```bash
go test ./...                       # backend; integration tests need `make test-db`
cd frontend && pnpm test:run        # vitest
cd frontend && pnpm lint            # eslint (with --fix)
```

`make test` runs both suites.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `encryption_key must be exactly 32 characters` on boot | Set `[app] encryption_key` to a 32-char string. `openssl rand -hex 16` gives exactly 32. |
| Attachments/avatars fail to upload | `uploads/` does not exist in the working directory. `mkdir -p uploads`. |
| Login succeeds but you're bounced back to the login page | Secure cookies over plain HTTP. Set `[app.server] disable_secure_cookies = true`. |
| `bind: address already in use` / port conflicts | 5432 (Postgres), 6379 (Redis), 8000 (Vite), 9000 (backend). Stop the other process or change the port (`LD_DEV_PORT` for Vite, `[app.server] address` for the backend). |
| `error initializing local FS` or missing i18n on boot | Backend started outside the repo root. `cd` to the root and rerun `make run-backend`. |
| `pnpm install` or Vite dev server is painfully slow | The clone lives under `/mnt/c`. Move it into the WSL filesystem (`~/…`). |
| App refuses to start, mentions pending migrations | `go run ./cmd/ --upgrade --yes --config config.toml`. |
| `Cannot find matching keyid` from corepack | Update corepack, or run with `COREPACK_INTEGRITY_KEYS=0`. |

## Known limitation: native Windows builds

`cmd/handlers.go` serves the frontend assets with `filepath.Join`, which
produces `\` separators on Windows, while the stuffbin filesystem is indexed
with `/`. A binary built **natively** on Windows therefore 404s on `/assets/*`.
This does not affect the WSL2 workflow above. Tracked separately.
