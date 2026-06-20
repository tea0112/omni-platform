# Identity Service Design

## Overview

A Go User Management API providing email/password authentication, session management with refresh tokens, and RBAC authorization. Exposes both REST and gRPC on a single port. Lives under `services/identity/` in a monorepo, with database-per-service and per-service Docker Compose.

## Architecture

Layered monolith (Approach A): single binary with clean internal layering — transport (HTTP/gRPC), service/domain logic, data access. Shared domain models, one database.

```
Transport (chi + connect-go)
    │
    ▼
Service Layer (AuthSvc, UserSvc, SessionSvc, RoleSvc)
    │
    ▼
Repository Layer (UserRepo, SessionRepo, RoleRepo)
    │
    ▼
CockroachDB
```

Single `http.Server` on :8080. Chi router mounts REST handlers at `/api/v1/*` and connect-go mounts gRPC handlers at `/grpc.v1.*/*`. OTEL, slog, auth, rate-limiting middleware wraps once for all handlers.

RBAC at the service layer — transport only handles authentication (extracting principal from JWT), service layer handles authorization (`rbac.Authorize(ctx, permission)`). No permission config in routes or gRPC interceptors.

## Features

- **Basic auth**: email/password registration, login, logout, forgot/reset password flow
- **Sessions**: refresh tokens, device tracking, session list, individual/bulk revocation
- **RBAC**: roles, permissions, role assignment, fine-grained checks (own profile vs other users)

## Data Model

Six tables, all UUID primary keys (CockroachDB-compatible):

- **users**: id, email (unique), password_hash, display_name, email_verified, created_at, updated_at
- **sessions**: id, user_id (FK), refresh_token (unique, opaque UUID), device_info (JSONB), ip_address, expires_at, revoked_at (nullable), created_at
- **roles**: id, name (unique), description, created_at
- **user_roles**: user_id + role_id (composite PK)
- **role_permissions**: role_id + permission (composite PK)
- **password_reset_tokens**: id, user_id (FK), token (unique), expires_at, used_at (nullable)

### Token Strategy

- Access tokens: stateless JWTs (15min TTL), signed with Ed25519, contain `sub`, `roles`, `permissions`
- Refresh tokens: opaque UUIDs in sessions table (28 day TTL), rotated on every use (old revoked, new issued)
- Revocation: delete/revoke session row — instant, no deny list needed
- Token reuse detection: if a revoked refresh token is presented, all sessions for that user are immediately revoked (assumed compromised)

## API Surface

### REST Endpoints (`/api/v1`)

**Auth** (public):
- `POST /auth/register` — create account
- `POST /auth/login` — returns access + refresh tokens
- `POST /auth/refresh` — rotate refresh token, return new pair
- `POST /auth/forgot-password` — generates reset token, sends email via `EmailSender` interface (default: SMTP)
- `POST /auth/reset-password` — consumes reset token, sets new password

**Auth** (protected):
- `POST /auth/logout` — revokes session

**Users** (protected):
- `GET /users/{id}` — get user (own or admin)
- `PATCH /users/{id}` — update profile
- `GET /users` — list users (admin)

**Sessions** (protected):
- `GET /users/{id}/sessions` — list active sessions
- `DELETE /users/{id}/sessions/{sid}` — revoke one session
- `DELETE /users/{id}/sessions` — revoke all sessions

**Roles** (admin):
- `GET /roles`, `POST /roles`, `DELETE /roles/{id}`
- `POST /roles/{id}/permissions`, `DELETE /roles/{id}/permissions/{perm}`
- `POST /users/{id}/roles`, `DELETE /users/{id}/roles/{rid}`

### gRPC Services

Single proto file at `proto/identity/v1/service.proto` defining `AuthService`, `UserService`, `SessionService`, `RoleService`. Mounted via connect-go on the same chi mux.

## Project Structure

```
services/identity/
├── cmd/
│   └── server/
│       └── main.go              ← FX-based entrypoint
├── internal/
│   ├── transport/
│   │   ├── rest/
│   │   │   ├── handler.go       ← chi route setup + handler structs
│   │   │   ├── auth.go
│   │   │   ├── user.go
│   │   │   ├── session.go
│   │   │   └── role.go
│   │   ├── grpc/
│   │   │   ├── auth.go          ← connect-go handler
│   │   │   ├── user.go
│   │   │   ├── session.go
│   │   │   └── role.go
│   │   └── errors.go            ← MapError (transport edge only)
│   ├── service/
│   │   ├── auth.go              ← AuthService
│   │   ├── user.go              ← UserService
│   │   ├── session.go           ← SessionService
│   │   ├── role.go              ← RoleService
│   │   └── interfaces.go        ← interfaces + go:generate for mocks
│   ├── domain/
│   │   ├── user.go              ← User, Credentials, AuthResult structs
│   │   ├── session.go
│   │   ├── role.go
│   │   └── errors.go            ← Plain sentinel errors (no HTTP/gRPC codes)
│   ├── repo/
│   │   ├── user.go              ← UserRepository (pgx)
│   │   ├── session.go
│   │   ├── role.go
│   │   └── migrations/
│   │       ├── 001_create_users.up.sql
│   │       ├── 002_create_sessions.up.sql
│   │       ├── 003_create_roles.up.sql
│   │       └── 004_create_password_resets.up.sql
│   ├── auth/
│   │   ├── jwt.go               ← JWT sign/verify (Ed25519)
│   │   ├── password.go          ← bcrypt hash/compare
│   │   └── middleware.go        ← Authenticate middleware (extracts principal)
│   ├── rbac/
│   │   └── rbac.go              ← Can(ctx, action, resource...) error
│   ├── email/
│   │   └── email.go             ← EmailSender interface + SMTP implementation
│   ├── otel/
│   │   └── otel.go              ← TracerProvider setup
│   └── config/
│       └── config.go            ← Viper-based, validated at startup
├── proto/
│   └── identity/
│       └── v1/
│           └── service.proto
├── Dockerfile
├── docker-compose.yml           ← identity + cockroachdb + migrate
├── go.mod                       ← Independent Go module
└── go.sum
```

Root `docker-compose.yml` handles cross-cutting infra only (OTEL collector).

Each container is named with service prefix: `identity-crdb`, `identity-server`, `identity-migrate` — for easy identification in Podman dashboard.

## Error Handling

### Domain Layer

Plain sentinel errors with NO transport concerns:

```go
var (
    ErrNotFound        = errors.New("not found")
    ErrDuplicate       = errors.New("already exists")
    ErrUnauthenticated = errors.New("unauthenticated")
    ErrForbidden       = errors.New("forbidden")
    ErrTokenExpired    = errors.New("token expired")
    ErrTokenRevoked    = errors.New("token revoked")
)

type ValidationError struct {
    Fields map[string]string
}
```

### Transport Edge

Single `MapError` function at the transport layer maps domain errors to HTTP/gRPC status codes:

```go
func MapError(err error) (int, codes.Code, map[string]any) { ... }
```

Service layer never sees HTTP/gRPC codes. Handlers call `MapError` and write the result.

## Testing

### Unit Tests (Uber GoMock)

Interfaces defined in `service/interfaces.go`. Mocks generated via `go:generate mockgen` into `service/mocks/`. Service methods tested with mocked repos and RBAC.

### Integration Tests (testcontainers-go)

Real CockroachDB spun up via testcontainers. Repo layer tested against real DB. Service + real repo tested together.

### E2E Tests

`docker compose up` the service, then a Go test binary hits REST + gRPC with real clients. Happy paths only: register → login → refresh → CRUD.

## Middleware Chain

Applied once on chi mux, covers both REST and gRPC:

```
RequestID → RealIP → slog → OTEL → Recoverer → Timeout(30s)
```

Public routes (register, login, refresh, forgot/reset password) skip auth. Protected routes additionally run `Authenticate` middleware which validates JWT and injects principal into context. `Authenticate` does NOT check permissions — that's the service layer's job.

## Dependency Injection (Uber FX)

```go
app := fx.New(
    fx.Provide(config.New, otel.NewTracerProvider, repo.NewUserRepo, ...),
    fx.Invoke(repo.RunMigrations, transport.Serve),
    fx.WithLogger(slogfx.New),
)
app.Run() // blocks until SIGINT/SIGTERM
```

FX builds the DAG from constructor signatures. Fails at startup if dependencies are missing or circular. Integration tests swap implementations via `fx.Replace`.

## Email Sending

`EmailSender` interface with two implementations, selected by config:

```go
type EmailSender interface {
    SendPasswordReset(ctx context.Context, to, token string) error
}
```

- **SMTP** (`email.NewSMTP`) — production, connects to configured SMTP server
- **Log** (`email.NewLog`) — local dev, prints reset link to slog (no real email sent)

FX wires per environment:

```go
fx.Provide(func(cfg config.Config, logger *slog.Logger) email.EmailSender {
    if cfg.Email.Provider == "smtp" {
        return email.NewSMTP(cfg.Email.SMTP)
    }
    return email.NewLog(logger)
})
```

Docker Compose sets `IDENTITY_EMAIL_PROVIDER=log` locally.

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/go-chi/chi/v5` | HTTP routing |
| `github.com/connectrpc/connect-go` | gRPC/Connect handler + client |
| `go.uber.org/fx` | Dependency injection |
| `github.com/jackc/pgx/v5` | PostgreSQL/CockroachDB driver |
| `github.com/golang-migrate/migrate/v4` | Schema migrations |
| `github.com/golang-jwt/jwt/v5` | JWT signing/verification |
| `github.com/spf13/viper` | Configuration |
| `go.opentelemetry.io/otel` | Tracing |
| `go.uber.org/mock` | Mock generation |
| `github.com/testcontainers/testcontainers-go` | Integration test infra |
| `github.com/stretchr/testify` | Test assertions |
| `golang.org/x/crypto` | bcrypt |

## Deployable

### Dockerfile

Multi-stage, distroless. `CGO_ENABLED=0`, runs as non-root.

### Docker Compose (per-service)

```yaml
services:
  identity-crdb:
    image: cockroachdb/cockroach:latest
    command: start-single-node --insecure
  identity-migrate:
    build: . ; command: ["/bin/server", "migrate"]
    depends_on: identity-crdb (healthy)
  identity-server:
    build: . ; ports: ["8080:8080"]
    environment: IDENTITY_DB_HOST=identity-crdb
    depends_on: identity-migrate (completed)
```

Each future service under `services/<name>/` gets its own compose with its own database.

## Future Services

Additional services live under `services/<name>/` with their own `go.mod`, `Dockerfile`, and `docker-compose.yml`. Shared infrastructure (OTEL collector) at root `docker-compose.yml`.
