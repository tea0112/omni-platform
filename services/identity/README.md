# Identity

Identity service for the Omni Platform. Handles authentication, users, sessions, and roles.

## Prerequisites

- Go 1.26+
- PostgreSQL (user: `identity`, password: `identity`, db: `identity`, port: `5432`) — easiest via `just db-up` (uses Docker)
- [`just`](https://github.com/casey/just) (`brew install just` or `apt install just`)

## Setup

```bash
go mod download
cp .env.example .env.local  # not needed if you use `just gen-jwk`
just db-up
just migrate
just gen-jwk   # writes a fresh Ed25519 JWK to .env.local
```

That's the full first-time setup.

## Run

```bash
just dev
```

The server starts on `http://localhost:8080`. Health check at `/healthz`. `just dev` brings up the database, runs migrations, and starts the server.

## Recipes

The justfile is the single source of truth for local development. Run `just` (no args) to see all available recipes.

Common ones:

- `just dev` — full local stack (db + migrate + JWK + server)
- `just ci` — run vet, test, build
- `just test` — run all tests
- `just test-integration` — integration tests (requires db-up)
- `just db-reset` — wipe and recreate the local database (requires `--confirm`)
- `just migrate` / `just migrate-down` — apply or roll back one migration
- `just gen-jwk` — generate a fresh Ed25519 JWK in `.env.local`
- `just reset` — tear down everything (containers, build, .env.local)
- `just docker-up` — start the full stack (postgres + server) via Docker Compose

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `IDENTITY_DB_HOST` | `localhost` | PostgreSQL host |
| `IDENTITY_DB_PORT` | `5432` | PostgreSQL port |
| `IDENTITY_DB_USER` | `identity` | PostgreSQL user |
| `IDENTITY_DB_PASSWORD` | `identity` | PostgreSQL password |
| `IDENTITY_DB_NAME` | `identity` | PostgreSQL database |
| `IDENTITY_DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `IDENTITY_SERVER_PORT` | `8080` | HTTP server port |
| `IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK` | *required* | Ed25519 JWK key pair JSON |
| `IDENTITY_AUTH_ACCESS_TOKEN_TTL` | `15m` | Access token TTL |
| `IDENTITY_AUTH_REFRESH_TOKEN_TTL` | `672h` | Refresh token TTL |
| `IDENTITY_AUTH_BCRYPT_COST` | `12` | Bcrypt cost factor |
| `IDENTITY_EMAIL_PROVIDER` | `log` | Email provider (`log` or `smtp`) |
| `IDENTITY_OTEL_ENDPOINT` | `localhost:4317` | OpenTelemetry collector endpoint |

## Run with Docker Compose

```bash
just gen-jwk
just docker-up
```

`just gen-jwk` writes a fresh Ed25519 JWK to `.env.local`. `set dotenv-load := true` (at the top of the justfile) propagates the JWK into the shell environment for every recipe, including `docker-up`. `docker compose` then reads `IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK` from the environment.
