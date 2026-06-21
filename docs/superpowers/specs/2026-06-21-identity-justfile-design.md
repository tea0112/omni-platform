# Identity Service `just` Task Runner

## Overview

The identity service has no formal task runner today. Build, test, migrate, and run commands are scattered across `README.md`, `docker-compose.yml`, and developer memory. The README documents subcommands (`gen-jwk`, `migrate`) that don't exist in `main.go` — the docker-compose `identity-migrate` service is broken because of this.

This design introduces [`just`](https://github.com/casey/just) as the single source of truth for local development commands. It also fixes the missing subcommands and adds `.env.example` files to both the identity service and the web app.

## Goals

1. **Single source of truth**: `services/identity/justfile` is the canonical entry point for local development. `README.md` references `just <recipe>` instead of inline shell commands.
2. **Working subcommands**: `gen-jwk` and `migrate` are real Go subcommands in `cmd/server/main.go`. The docker-compose `identity-migrate` service works as designed.
3. **Zero-friction onboarding**: `just dev` brings up the full local stack (db, migrations, JWK, server) with no manual steps. Secrets are managed via `.env.local`; defaults are managed by `shared.Config`.
4. **Env documentation**: both `services/identity/.env.example` and `apps/web/.env.example` exist as committed templates showing every variable with its default.
5. **CI-ready**: a `just ci` recipe runs the same checks CI would (vet, test, build).

## Non-Goals

- A linter (e.g., golangci-lint) — not in the project today; `lint` recipe is a placeholder.
- A file watcher for hot-reload — developers use editor hot-reload or re-run `just dev`.
- A CLI library (cobra, urfave/cli) — `os.Args` switch is sufficient for two subcommands.
- Splitting the justfile into modules (per the brainstorm's "Approach C") — premature for one service.
- Applying the justfile pattern to the web app — explicitly out of scope.
- CI system integration (GitHub Actions, etc.) — the `ci` recipe defines what CI should run; wiring is separate.

## Architecture

### File Layout

```
services/identity/
├── cmd/server/
│   ├── main.go                  # MODIFY: add subcommand dispatch
│   └── main_test.go             # NEW: smoke tests for subcommand dispatch
├── internal/
│   ├── genkey/                  # NEW
│   │   ├── genkey.go            # Generate() Ed25519 key, return JWK string
│   │   └── genkey_test.go       # Unit test
│   └── migrate/
│       ├── migrate.go           # MODIFY: add Down() function
│       └── migrate_test.go      # NEW: unit test for Down()
├── .env.example                 # NEW (committed): template
├── .env.local                   # NEW (gitignored): local config with JWK
├── justfile                     # NEW: task runner
├── README.md                    # UPDATE: `just` invocations
└── docker-compose.yml           # UNCHANGED (now works because subcommands exist)

apps/web/
├── .env.example                 # NEW (committed): template
├── .env.local                   # (may already be gitignored) NEW: local config
└── README.md                    # UPDATE: env setup note
```

### Dependency Direction

```
        ┌──────────────────┐
        │     justfile     │  (recipes, no Go code)
        └────────┬─────────┘
                 │ invokes
                 ▼
        ┌──────────────────┐
        │  cmd/server       │  (subcommand dispatch)
        │  main.go          │
        └────────┬─────────┘
                 │ calls
                 ▼
        ┌──────────────────┐
        │ internal/genkey  │  (Generate)
        │ internal/migrate │  (Run, Down)
        │ internal/shared  │  (Config)
        └──────────────────┘
```

The justfile depends on Go binaries (via `go run`, `go build`, `go test`) and Docker (for postgres). The Go subcommands depend only on the existing `internal/` packages.

### Cross-Service Rules

- `services/identity/justfile` only operates on the identity service — never reaches into `apps/web/`
- The web app has its own `.env.example` and `.env.local` (per Next.js convention) — no justfile
- The docker-compose `identity-migrate` service depends on the new `migrate` subcommand in the identity binary

## Recipe Catalog

The justfile has three layers: **phony workflow targets** (entry points), **granular recipes** (one per operation), and **private helpers** (prefixed with `_`, repeated logic).

### Phony Workflow Targets

```just
# Default: just dev
default: dev

# Full local stack: db-up → migrate → gen-jwk (if needed) → run
dev: db-up migrate
    @if [ ! -f .env.local ]; then just gen-jwk; fi
    just run

# What CI runs: vet → test → build
ci: vet test build

# Tear everything down (containers, build artifacts, .env.local)
reset: db-down clean
    rm -f .env.local
```

### Granular Recipes

| Recipe | What it does | Depends on |
|---|---|---|
| `db-up` | `docker compose up -d identity-postgres` | docker |
| `db-down` | `docker compose down` | docker |
| `db-logs` | `docker compose logs -f identity-postgres` | docker |
| `db-ps` | `docker compose ps` | docker |
| `db-reset` | `db-down && docker volume rm identity-pgdata && db-up` | docker, `--confirm` |
| `migrate` | `go run ./cmd/server migrate` | db-up |
| `migrate-down` | `go run ./cmd/server migrate down` (rollback one step) | db-up |
| `gen-jwk` | `go run ./cmd/server gen-jwk` (writes `.env.local`) | — |
| `test` | `go test ./...` | — |
| `test-unit` | `go test -short ./...` | — |
| `test-integration` | `go test -tags=integration ./...` | db-up |
| `test-coverage` | `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out` | — |
| `build` | `go build -trimpath -o server ./cmd/server` | — |
| `run` | `./server` (binary built by `build`) | build, `.env.local` |
| `vet` | `go vet ./...` | — |
| `fmt` | `go fmt ./...` | — |
| `generate` | `go generate ./...` | — |
| `lint` | placeholder: prints "no linter configured" | — |
| `clean` | `rm -f server coverage.out` | — |
| `docker-up` | `docker compose up -d` (full stack including server) | docker, JWK |
| `docker-down` | `docker compose down` | docker |
| `docker-logs` | `docker compose logs -f` | docker |
| `list` | `just --list` (default justfile behavior) | — |

### Private Helpers

```just
_docker := 'docker compose'

_go_flags := '-trimpath'

# Run a Go subcommand from the server binary (no fx setup)
_go_run cmd *args:
    go run ./cmd/server {{ cmd }} {{ args }}

# Build the server binary (idempotent)
_build_server:
    go build {{ _go_flags }} -o server ./cmd/server
```

## Environment Management

### Files Per Service

| Service | `.env.example` (committed) | `.env.local` (gitignored) |
|---|---|---|
| `services/identity/` | All env vars documented with defaults | Local config including the JWK |
| `apps/web/` | All env vars documented with defaults | Local config (only `IDENTITY_SERVICE_URL` if non-default) |

### `services/identity/.env.example`

```bash
# --- Database ---
IDENTITY_DB_HOST=localhost
IDENTITY_DB_PORT=5432
IDENTITY_DB_USER=identity
IDENTITY_DB_PASSWORD=identity
IDENTITY_DB_NAME=identity
IDENTITY_DB_SSLMODE=disable

# --- Server ---
IDENTITY_SERVER_PORT=8080

# --- Auth ---
# Generate via `just gen-jwk`
# IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK={"kty":"OKP","crv":"Ed25519","d":"...","x":"..."}
IDENTITY_AUTH_ACCESS_TOKEN_TTL=15m
IDENTITY_AUTH_REFRESH_TOKEN_TTL=672h
IDENTITY_AUTH_BCRYPT_COST=12

# --- Email ---
IDENTITY_EMAIL_PROVIDER=log

# --- Observability ---
IDENTITY_OTEL_ENDPOINT=localhost:4317
```

### `apps/web/.env.example`

```bash
# Identity service URL (default: http://localhost:8080)
# IDENTITY_SERVICE_URL=http://localhost:8080
```

### `.env.local` Behavior

- **Identity service**: created by `just gen-jwk`. Contains only the JWK line. Sourced by the justfile via `set dotenv-load := true` so every recipe sees the JWK. The server binary also reads it via the existing `viper`/env-loading code.
- **Web app**: created by the developer if they need to override `IDENTITY_SERVICE_URL`. Next.js loads `.env.local` automatically per its built-in convention.

### `.gitignore` Updates

- `services/identity/.gitignore` — add `.env.local`
- `apps/web/.gitignore` — verify `.env.local` is already there (Next.js convention); add if missing

## Go Subcommands

### `gen-jwk`

**New file** `services/identity/internal/genkey/genkey.go`:

```go
package genkey

// Generate returns a freshly-generated Ed25519 key as a JWK JSON string.
// Format: {"kty":"OKP","crv":"Ed25519","d":"<base64url-seed>","x":"<base64url-pub>"}
func Generate() (string, error)
```

**`cmd/server/main.go` dispatch**:
```go
case "gen-jwk":
    os.Exit(runGenJwk())
```

`runGenJwk()`:
1. Calls `genkey.Generate()` to get the JWK string
2. Writes to `services/identity/.env.local` (creates the file with `IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK=<jwk>`)
3. Prints a one-line success message to stdout

### `migrate`

**Modify** `services/identity/internal/migrate/migrate.go` — add:
```go
// Down rolls back the most recent migration.
func Down(dbURL string) error
```

**`cmd/server/main.go` dispatch**:
```go
case "migrate":
    os.Exit(runMigrate(os.Args[2:]))
```

`runMigrate(args []string)`:
- No args → `migrate.Run(dbURL)` (existing `Up`)
- `down` → `migrate.Down(dbURL)` (new)
- `force <version>` → forward to the migrate library's force command

This fixes the existing docker-compose `identity-migrate` service which already calls `/bin/server migrate`.

### `main.go` Subcommand Dispatch

```go
func main() {
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "gen-jwk":
            os.Exit(runGenJwk())
        case "migrate":
            os.Exit(runMigrate(os.Args[2:]))
        }
    }
    // existing fx.New(...) + Run() for the server
}
```

No CLI library. Subcommands don't initialize the full fx app — `gen-jwk` doesn't need a DB connection, `migrate` just needs the DB config from `shared.Config`.

## Data Flow

### `just dev` Happy Path

1. `just dev` → `db-up` (recipe dependency)
2. `db-up` → `docker compose up -d identity-postgres` → postgres container starts; healthcheck passes within ~5s
3. `migrate` → `go run ./cmd/server migrate` → reads `IDENTITY_DB_*` from env (defaults work for local) → calls `internal/migrate.Run` → migrations applied
4. `gen-jwk` (only if `.env.local` doesn't exist) → generates a JWK, writes to `.env.local`
5. `run` → `set dotenv-load` sources `.env.local` → `./server` starts with `IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK` populated
6. Server listens on `:8080`

### Subcommand Flow: `gen-jwk`

1. User runs `just gen-jwk`
2. Justfile invokes `go run ./cmd/server gen-jwk`
3. `main.go` sees `os.Args[1] == "gen-jwk"`, calls `runGenJwk()`
4. `runGenJwk()` calls `genkey.Generate()` → returns JWK JSON string
5. Writes to `services/identity/.env.local` (overwrites if exists; prints warning)
6. Exits 0

### Subcommand Flow: `migrate`

1. User runs `just migrate`
2. Justfile invokes `go run ./cmd/server migrate`
3. `main.go` sees `os.Args[1] == "migrate"`, calls `runMigrate([]string{})`
4. `runMigrate` reads `shared.Config` (just the DB fields) via env vars
5. Calls `migrate.Run(dbURL)` → migrations applied
6. Exits 0

## Error Handling

| Failure | Recipe behavior |
|---|---|
| `docker` not on PATH | `db-up` prints "docker not found; install Docker or use a system Postgres" and exits non-zero |
| Postgres container won't start | `db-up` exits non-zero; `migrate` and downstream recipes don't run |
| `.env.local` missing, JWK required | `run` fails fast with "run `just gen-jwk` first" |
| `gen-jwk` overwrites existing key | `just gen-jwk --confirm` (just's built-in) or interactive prompt; warns "this invalidates all existing tokens" |
| `migrate` fails | recipe exits non-zero; full error trace printed |
| `test-integration` without db-up | fails with "run `just db-up` first" |
| `go build` fails | recipe exits non-zero; build error printed |

**Safety rails**:
- `gen-jwk` only writes to `.env.local` (never to `.env.example`)
- `db-reset` requires `--confirm` (deletes the docker volume)
- `reset` requires `--confirm` (deletes `.env.local` and `server` binary)

## Testing Strategy

| Layer | Test type | File |
|---|---|---|
| `genkey.Generate()` | Unit | `services/identity/internal/genkey/genkey_test.go` |
| `migrate.Down()` | Integration (real Postgres) | `services/identity/internal/migrate/migrate_test.go` |
| Subcommand dispatch | Smoke (writes tmp file, runs migrate against a real test DB) | `services/identity/cmd/server/main_test.go` |
| Justfile recipes | Manual end-to-end smoke test by running `just dev` | — |
| Justfile private helpers | No test (trivially thin) | — |

The justfile itself isn't unit-tested; it's exercised once during implementation (in the SDD-implementer smoke test) and any time the justfile changes (also in the smoke test).

## README & Documentation Updates

### `services/identity/README.md`

- Replace "Setup", "Run", "Run with Docker Compose" sections with `just <recipe>` invocations
- Add a "Recipes" section listing every recipe with one-line description
- Keep the "Environment Variables" table (documents the variables; the justfile is how you set them)

### `apps/web/README.md`

- Add a "Setup" note pointing to `.env.example` and explaining `IDENTITY_SERVICE_URL`
- Otherwise unchanged

### Spec documents

- `docs/superpowers/specs/2026-06-20-identity-service-design.md` — no changes (documents architecture, not build process)
- `docs/superpowers/specs/2026-06-20-web-app-design.md` — no changes

## Migration Plan

| Phase | What | Files |
|---|---|---|
| 1 | Add `internal/genkey` package | `services/identity/internal/genkey/genkey.go` + `_test.go` |
| 2 | Add `migrate.Down()` to existing migrate package | `services/identity/internal/migrate/migrate.go` + `_test.go` |
| 3 | Add subcommand dispatch to `main.go` | `services/identity/cmd/server/main.go` |
| 4 | Add smoke test for subcommand dispatch | `services/identity/cmd/server/main_test.go` |
| 5 | Add `.env.example` (committed) and update `.gitignore` for both services | `services/identity/.env.example`, `services/identity/.gitignore`, `apps/web/.env.example`, `apps/web/.gitignore` |
| 6 | Add the justfile | `services/identity/justfile` |
| 7 | Update READMEs | `services/identity/README.md`, `apps/web/README.md` |
| 8 | End-to-end smoke test: `just dev` brings up the stack, `/healthz` returns 200, register + login work, `just ci` passes | — |

Each Go phase ends with `go build ./...` and `go test ./...`. The final phase runs the full `just dev` smoke test plus `just ci`.

## File Summary

**New files** (8):
- `services/identity/internal/genkey/genkey.go`
- `services/identity/internal/genkey/genkey_test.go`
- `services/identity/cmd/server/main_test.go`
- `services/identity/internal/migrate/migrate_test.go`
- `services/identity/.env.example`
- `services/identity/justfile`
- `apps/web/.env.example`
- `docs/superpowers/plans/2026-06-21-identity-justfile-plan.md` (the implementation plan)

**Modified files** (5):
- `services/identity/cmd/server/main.go` — add subcommand dispatch
- `services/identity/internal/migrate/migrate.go` — add `Down()` function
- `services/identity/.gitignore` — add `.env.local`
- `services/identity/README.md` — replace manual commands with just invocations
- `apps/web/README.md` — add env setup note
- `apps/web/.gitignore` — add `.env.local` (if not already there)

Plus this spec doc: `docs/superpowers/specs/2026-06-21-identity-justfile-design.md`

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| `just` not installed on developer's machine | Document the install in the README; `just --version` is the first check in onboarding |
| `.env.local` accidentally committed | `.env.local` in `.gitignore` from day one; CI lint could check for committed secrets |
| `gen-jwk` overwrites a working JWK and breaks all tokens | Justfile's `--confirm` flag required to overwrite; warning printed |
| `db-reset` deletes production data | `db-reset` requires `--confirm`; only operates on the local docker volume `identity-pgdata` |
| `migrate` subcommand diverges from `internal/migrate.Run` | Same function, no duplication; `migrate down` calls a new `Down()` function that mirrors `Run()`'s structure |
| CI runs `just ci` differently from local | `ci` recipe is the same locally and in CI; no environment-specific branches |
| Subcommand dispatch pollutes the server binary | Subcommands exit before `fx.New(...)` runs; no fx setup overhead; no DB connection needed for `gen-jwk` |
