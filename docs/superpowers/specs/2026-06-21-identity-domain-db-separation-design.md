# Identity Service: Domain/Database Model Separation

## Overview

The current identity service has each feature module (`user`, `auth`, `session`, `role`) define its domain types (`User`, `Session`, `Role`) in `domain.go` and reuse those same types as the database row representation returned by the repository. This means domain types carry:

- `json` tags (transport concern)
- Field shapes that match DB columns one-to-one, including JSON round-tripping inside the repo (`DeviceInfo` is a `map[string]any` in the domain but `[]byte` from the DB)
- No boundary between "what the business logic reasons about" and "what the database stores"

Additionally, the `auth` and `user` modules each define their own `User` struct that maps to the same `users` table.

This refactor introduces a clear separation:

- **Domain types** are pure: no `json` tags, no DB-shape concerns, free to express business concepts.
- **Database row types** live next to the pgx implementation, with explicit conversion methods to the domain.
- **Repository interfaces return row types**. The service is the one place that maps row → domain.
- **Transactions are made possible** via a context-based querier lookup and a `TxRunner` interface — without leaking pgx into the service layer.
- **Cross-module dependencies are explicitly downward** (auth and user share `identityuser.User`; neither imports the other's service).

The goal is a clean architecture boundary: the persistence layer can be swapped (or split into a separate service) without touching the service or transport layers.

## Goals

1. Domain types are free of transport and persistence concerns.
2. A single `User` type exists for the `users` table, owned by a shared `identityuser` package that both `auth` and `user` import.
3. Repository interfaces return row types; conversion happens at the service boundary.
4. Transactions can be added to any service without changing the service's primary signature.
5. The service layer has zero knowledge of pgx, `pgxpool.Pool`, or any concrete database type.
6. Modules can be split into separate services later without rewriting domain or service code paths.

## Non-Goals

- Wiring actual transactions into existing flows (Login, Register, etc.). The `TxRunner` machinery is in place; using it is a follow-up.
- Changing the JSON wire format. The shape of every HTTP response and gRPC message is preserved.
- Changing the database schema or migrations.
- Replacing the persistence backend (Postgres stays).
- New features or endpoints.

## Architecture

### Package Layout

```
internal/
  identityuser/                 # NEW: shared User domain type
    domain.go                   # identityuser.User (pure)
  user/
    domain.go                   # UpdateUserRequest (no User, no json tags)
    handler.go / grpc.go        # DTOs with json tags, mapped to/from domain
    repo.go                     # userRow + interface returning userRow
    service.go                  # converts row → domain, RBAC
  auth/
    domain.go                   # UserCredentials, AuthResult, Credentials,
                                # ChangePasswordInput, ChangeEmailInput,
                                # PasswordResetToken, SessionContext
    handler_*.go / grpc.go
    repo.go                     # interfaces
    repo_user.go                # userCredentialsRow + UserCredentials mapping
    repo_session.go             # sessionContextRow + (Session, SessionContext) mapping
    service.go / service_*.go   # business logic, uses txr where needed
  session/
    domain.go                   # Session (pure, no DeviceInfo/IP/RefreshToken)
    handler.go / grpc.go
    repo.go                     # sessionRow
    service.go
  role/
    domain.go                   # Role + request types (no json tags)
    handler.go / grpc.go
    repo.go                     # roleRow
    service.go
  shared/
    querier.go                  # NEW: Querier interface, WithQuerier,
                                # QuerierFromContext
    tx.go                       # NEW: TxRunner interface
    pg_tx_runner.go             # NEW: pgx-backed TxRunner implementation
    errors.go, jwt.go, rbac.go, password.go, ...   # unchanged
cmd/server/main.go              # fx wiring updated
```

### Dependency Direction

```
        ┌──────────────┐
        │   handler    │  (HTTP/gRPC, depends on service)
        └──────┬───────┘
               ▼
        ┌──────────────┐
        │   service    │  (depends on repo interface, optional TxRunner)
        └──────┬───────┘
               ▼
        ┌──────────────┐
        │  repo (pg)   │  (pgx impl, returns row types, looks up tx from ctx)
        └──────┬───────┘
               ▼
        ┌──────────────┐
        │   domain     │  (pure types, no upstream imports)
        └──────────────┘
```

**Cross-module rules**:

- Domain types flow upward: anyone may import `identityuser.User`, `session.Session`, `role.Role`.
- Services and repositories stay within their module. `auth.Service` does not import `user.Service` and vice versa.
- If a service ever needs to call another service (not the case today), the *consumer* defines a small interface in its own package; the *producer's* service is injected as that interface. The producer has zero knowledge of the consumer. Wiring happens in `cmd/server/main.go`, the only place both modules are imported together.
- Both `auth` and `user` may import `identityuser`. Neither imports the other.
- All services depend on `shared.TxRunner` (interface), not the pgx-backed implementation, when they need transactions.

### Service-to-Service Pattern (Documented for Future Use)

No service-to-service calls exist today, but the pattern is reserved:

```go
// In the consumer module (e.g., user):
type ProfileReader interface {
    GetByID(ctx context.Context, id uuid.UUID) (*identityuser.User, error)
}
type UserService struct {
    profiles ProfileReader  // injected as an interface
    rbac     *shared.RBAC
}
```

The producer (`auth` or another module) implements the interface; `cmd/server/main.go` injects the concrete service. This keeps the dependency direction one-way: producer has no knowledge of the consumer.

## Domain Types

### `identityuser.User` (pure)

```go
package identityuser

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID            uuid.UUID
    Email         string
    DisplayName   string
    EmailVerified bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### `auth.UserCredentials` (composes User with password hash)

```go
package auth

import "github.com/tea0112/omni-platform/services/identity/internal/identityuser"

type UserCredentials struct {
    user         identityuser.User
    passwordHash string
}

func NewUserCredentials(u identityuser.User, hash string) *UserCredentials
func (c *UserCredentials) User() identityuser.User   // strips password
func (c *UserCredentials) PasswordHash() string
```

The password hash never appears on the public `identityuser.User`. Services that need the hash for credential checks hold a `*UserCredentials`; everywhere else, `*identityuser.User` is used.

### `auth.AuthResult` (no password)

```go
type AuthResult struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
    User         identityuser.User   // never UserCredentials
}
```

### `session.Session` (pure, lifecycle-only)

```go
package session

type Session struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    ExpiresAt time.Time
    RevokedAt *time.Time
    CreatedAt time.Time
}
```

`RefreshToken`, `DeviceInfo`, and `IPAddress` are auth-flow concerns, not session lifecycle. They move to `auth.SessionContext`.

### `auth.SessionContext` (auth-flow decorations)

```go
package auth

type SessionContext struct {
    RefreshToken string
    DeviceInfo   map[string]any
    IPAddress    string
}

type SessionWithContext struct {
    session.Session
    SessionContext
}
```

The auth session repo returns `*sessionContextRow`; the session repo returns `*sessionRow`. The two rows map to the same DB table; the auth row reads more columns. The service composes the auth's row via `row.toSession()` (lifecycle) and `row.toContext()` (auth-flow fields), then assembles a `SessionWithContext` if both are needed.

### `role.Role` (pure)

```go
type Role struct {
    ID          uuid.UUID
    Name        string
    Description string
    CreatedAt   time.Time
}
```

`CreateRoleRequest`, `UpdateRoleRequest`, `AddPermissionRequest` are domain input types — no `json` tags. They are passed from handlers (which parse DTOs) to the service.

## Row Types and Conversion

### Row Types (Unexported, in `repo.go`)

Each module defines its row type in the same file as the repository implementation. Rows are unexported; only the package's repository code references them.

```go
// internal/user/repo.go
type userRow struct {
    ID            uuid.UUID
    Email         string
    DisplayName   string
    EmailVerified bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// internal/auth/repo_user.go
type userCredentialsRow struct {
    ID            uuid.UUID
    Email         string
    PasswordHash  string
    DisplayName   string
    EmailVerified bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// internal/session/repo.go
type sessionRow struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    ExpiresAt time.Time
    RevokedAt *time.Time
    CreatedAt time.Time
}

// internal/auth/repo_session.go
type sessionContextRow struct {
    ID           uuid.UUID
    UserID       uuid.UUID
    RefreshToken string
    DeviceInfo   []byte   // raw JSONB
    IPAddress    string
    ExpiresAt    time.Time
    RevokedAt    *time.Time
    CreatedAt    time.Time
}

// internal/role/repo.go
type roleRow struct {
    ID          uuid.UUID
    Name        string
    Description string
    CreatedAt   time.Time
}
```

### Conversion Methods

Each row type has a `toDomain()` method that returns the corresponding domain type:

```go
func (r userRow) toDomain() identityuser.User             { ... }
func (r userCredentialsRow) toDomain() UserCredentials    { ... }
func (r sessionRow) toDomain() session.Session            { ... }
func (r sessionContextRow) toSession() session.Session    { ... }
func (r sessionContextRow) toContext() SessionContext     { ... }
func (r roleRow) toDomain() Role                          { ... }
```

Insert-side conversions live as `fromDomain(...)` functions or methods in `repo.go`, used for the few `Create` operations that take a domain value and produce a row.

### Repository Interfaces

Each method returns the row type (or a slice of row types). No `Querier` parameter — that's looked up from context internally.

```go
// user module
type UserRepository interface {
    GetByID(ctx context.Context, id uuid.UUID) (*userRow, error)
    Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*userRow, error)
    List(ctx context.Context, offset, limit int) ([]userRow, error)
}

// session module
type SessionRepository interface {
    GetByUserID(ctx context.Context, userID uuid.UUID) ([]sessionRow, error)
    Revoke(ctx context.Context, id uuid.UUID) error
    RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// auth module — auth's session repo reads more columns (refresh token, device, IP)
type SessionRepository interface {
    CreateSession(ctx context.Context, userID uuid.UUID, refreshToken string, deviceInfo map[string]any, ipAddress string, expiresAt time.Time) (*sessionContextRow, error)
    GetByRefreshToken(ctx context.Context, refreshToken string) (*sessionContextRow, error)
    Revoke(ctx context.Context, id uuid.UUID) error
    RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
    ListByUser(ctx context.Context, userID uuid.UUID) ([]sessionContextRow, error)
}

// role module
type RoleRepository interface {
    Create(ctx context.Context, req CreateRoleRequest) (*roleRow, error)
    GetByID(ctx context.Context, id uuid.UUID) (*roleRow, error)
    List(ctx context.Context) ([]roleRow, error)
    Update(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*roleRow, error)
    Delete(ctx context.Context, id uuid.UUID) error
    AddPermission(ctx context.Context, roleID uuid.UUID, permission string) error
    RemovePermission(ctx context.Context, roleID uuid.UUID, permission string) error
    GetPermissions(ctx context.Context, roleID uuid.UUID) ([]string, error)
    GetUserRoles(ctx context.Context, userID uuid.UUID) ([]roleRow, error)
    AssignToUser(ctx context.Context, roleID, userID uuid.UUID) error
    RemoveFromUser(ctx context.Context, roleID, userID uuid.UUID) error
}
```

For `GetPermissions` returning `[]string`: this is a one-column query (just permission strings). Wrapping a single string in a row struct adds no value, so the repo returns `[]string` directly. The "row type" rule applies to multi-column entities (User, Role, Session), not single-column lists.

## Context-Based Transactions

### The `Querier` Abstraction

```go
// internal/shared/querier.go
type Querier interface {
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type ctxKey struct{}
var querierKey ctxKey

func WithQuerier(ctx context.Context, q Querier) context.Context {
    return context.WithValue(ctx, querierKey, q)
}

func QuerierFromContext(ctx context.Context) Querier {
    q, _ := ctx.Value(querierKey).(Querier)
    return q
}
```

Both `*pgxpool.Pool` and `pgx.Tx` satisfy `Querier`. A repo picks the right one based on context.

### The `TxRunner` Interface

```go
// internal/shared/tx.go
type TxRunner interface {
    RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

### The pgx Implementation

```go
// internal/shared/pg_tx_runner.go
type pgxTxRunner struct{ pool *pgxpool.Pool }

func NewPgxTxRunner(pool *pgxpool.Pool) TxRunner { return &pgxTxRunner{pool: pool} }

func (r *pgxTxRunner) RunInTx(ctx context.Context, fn func(context.Context) error) error {
    tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil { return err }
    defer tx.Rollback(ctx)
    if err := fn(WithQuerier(ctx, tx)); err != nil { return err }
    return tx.Commit(ctx)
}
```

### Repository Internal Querier Lookup

```go
// internal/user/repo.go
type UserPGRepository struct {
    defaultQuerier Querier
}

func (r *UserPGRepository) q(ctx context.Context) Querier {
    if txQ := shared.QuerierFromContext(ctx); txQ != nil {
        return txQ
    }
    return r.defaultQuerier
}

func (r *UserPGRepository) GetByID(ctx context.Context, id uuid.UUID) (*userRow, error) {
    row := &userRow{}
    err := r.q(ctx).QueryRow(ctx, `SELECT id, email, display_name, email_verified, created_at, updated_at FROM users WHERE id = $1`, id).
        Scan(&row.ID, &row.Email, &row.DisplayName, &row.EmailVerified, &row.CreatedAt, &row.UpdatedAt)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) { return nil, shared.ErrNotFound }
        return nil, fmt.Errorf("get user: %w", err)
    }
    return row, nil
}
```

The pgx types are imported only by the repository, the `pgxTxRunner`, and `shared/querier.go`. Services never see pgx.

### Service Usage

A service that needs atomicity depends on `shared.TxRunner` (interface, not pgx type):

```go
type UserService struct {
    repo UserRepository
    txr  shared.TxRunner   // nil if not used
    rbac *shared.RBAC
}
```

When wiring a service that doesn't need transactions, the `txr` field is omitted entirely. No service in this refactor uses `txr`; the wiring is in place so a follow-up can opt in.

## Data Flow

### HTTP Request: `GET /api/v1/users/{id}`

1. **Mux** dispatches to `userHandler.GetByID(w, r)`.
2. **Auth middleware** (`shared.Authenticate`) parses the JWT, attaches `Principal` to `ctx`.
3. **Handler** parses the path param, calls `h.svc.GetByID(r.Context(), id)`.
4. **Service** checks RBAC, calls `s.repo.GetByID(ctx, id)`, converts the returned `*userRow` to `*identityuser.User` via `row.toDomain()`.
5. **Repository** looks up the querier from `ctx` (no tx → uses the default pool), runs the SELECT, returns `*userRow`.
6. **Handler** maps `*identityuser.User` to a `userResponse` DTO (with json tags), writes the JSON response.

### gRPC Request: equivalent for `GetUser`

Same flow steps 4–5. The gRPC handler maps `*identityuser.User` to the protobuf `identityv1.GetUserResponse` message.

### Auth Flow (illustrative, not wired in this refactor)

`auth.AuthService.Register` would do:

```go
err := s.txr.RunInTx(ctx, func(ctx context.Context) error {
    if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
        return shared.ErrDuplicate
    }
    hash, err := s.hasher.Hash(password)
    if err != nil { return fmt.Errorf("hash password: %w", err) }
    row, err := s.userRepo.Create(ctx, email, hash)
    if err != nil { return fmt.Errorf("create user: %w", err) }
    if err := s.userRepo.CreateEmailVerificationToken(ctx, row.ID); err != nil {
        return fmt.Errorf("create verification token: %w", err)
    }
    u := row.toDomain().User()
    created = &u
    return nil
})
```

The four repo calls inside `fn` all run in the same transaction because `ctx` carries the `pgx.Tx` via `WithQuerier`.

## Error Handling

Errors are defined in `internal/shared/errors.go` (unchanged) and produced at the boundary where they belong.

**Repository**: translates pgx errors to domain errors and wraps other errors with context.

| pgx error | Domain error |
|---|---|
| `pgx.ErrNoRows` | `shared.ErrNotFound` |
| `*pgconn.PgError` code `23505` | `shared.ErrDuplicate` |
| anything else | wrapped: `fmt.Errorf("op name: %w", err)` |

**Service**: returns the error as-is. Doesn't re-wrap. For business-rule errors, returns typed errors directly (`shared.ErrDuplicate`, `&shared.ValidationError{...}`).

**HTTP handler**: maps domain errors to status codes via `shared.WriteErr`:

| Error | Status |
|---|---|
| `shared.ErrNotFound` | 404 |
| `shared.ErrDuplicate` | 409 |
| `shared.ErrUnauthenticated` | 401 |
| `shared.ErrForbidden` | 403 |
| `*shared.ValidationError` | 400 with field details |
| default | 500 |

**gRPC handler**: maps via `shared.AsConnectError` to Connect/gRPC codes with the same rules.

**Safety**: raw DB error strings are never included in HTTP/gRPC responses. PII (email, password hash, etc.) never appears in error messages. The handler logs the underlying error (with request ID) and returns a generic message.

## Transport DTOs

DTOs carry `json` tags and live next to the handlers that use them. Inline in `handler.go` / `grpc.go` for small modules; in a small `dto.go` for modules with many DTOs (decided case-by-case).

**Example** (`internal/user/handler.go`):

```go
type userResponse struct {
    ID            uuid.UUID `json:"id"`
    Email         string    `json:"email"`
    DisplayName   string    `json:"display_name"`
    EmailVerified bool      `json:"email_verified"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

func toUserResponse(u *identityuser.User) userResponse {
    return userResponse{
        ID: u.ID, Email: u.Email, DisplayName: u.DisplayName,
        EmailVerified: u.EmailVerified, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
    }
}

type updateUserRequestDTO struct {
    DisplayName *string `json:"display_name"`
}

func (d updateUserRequestDTO) toDomain() user.UpdateUserRequest {
    return user.UpdateUserRequest{DisplayName: d.DisplayName}
}
```

The JSON wire format is identical to today's output.

## Testing Strategy

### Tier 1: Service Unit Tests with Mocks

- `go.uber.org/mock/gomock` for repository mocks.
- Mocks return row types (e.g., `*userRow`, `*userCredentialsRow`).
- Tests verify row → domain conversion and business logic.
- For services that use `TxRunner` (none in this refactor), a `fakeTxRunner` is used:

  ```go
  type fakeTxRunner struct{ called bool }
  func (f *fakeTxRunner) RunInTx(ctx context.Context, fn func(context.Context) error) error {
      f.called = true
      return fn(ctx)  // no real tx in unit tests
  }
  ```

- Existing tests (`internal/auth/service_test.go`, `service_change_email_test.go`, `service_change_password_test.go`) are updated to use the new types. Test logic is preserved; mock return values change to rows.
- Mocks are regenerated by `go generate ./...` from the updated interfaces.

### Tier 2: Repository Integration Tests with Testcontainers

- `//go:build integration` tag.
- `internal/auth/repo_integration_test.go` is the template.
- Spins up Postgres via `testcontainers-go/modules/postgres`, runs migrations, exercises the pgx repo against a real DB.
- New tests verify `TxRunner` integration: a deliberate rollback undoes both repo calls inside `fn`.

### Tier 3: `Querier` Resolution Tests (New, Small)

One test per repository, no DB required:

```go
func TestUserPGRepository_PrefersTxFromContext(t *testing.T) {
    defaultQ := &fakeQuerier{}
    txQ := &fakeQuerier{}
    repo := &UserPGRepository{defaultQuerier: defaultQ}

    ctx := shared.WithQuerier(context.Background(), txQ)
    _, _ = repo.GetByID(ctx, uuid.New())

    assert.Equal(t, 1, txQ.calls)
    assert.Equal(t, 0, defaultQ.calls)
}

func TestUserPGRepository_FallsBackToDefault(t *testing.T) {
    defaultQ := &fakeQuerier{}
    repo := &UserPGRepository{defaultQuerier: defaultQ}

    _, _ = repo.GetByID(context.Background(), uuid.New())

    assert.Equal(t, 1, defaultQ.calls)
}
```

### Coverage Targets

- All public service methods: covered by tier 1.
- All public repository methods: covered by tier 2.
- The `toDomain()` conversion methods: exercised by both tiers.
- `Querier` resolution: covered by tier 3.
- `TxRunner.RunInTx` commit/rollback paths: covered by tier 2.

## Migration Plan

### Phase 1: `internal/shared` Infrastructure

- New file `internal/shared/querier.go`: `Querier` interface, `WithQuerier`, `QuerierFromContext`.
- New file `internal/shared/tx.go`: `TxRunner` interface.
- New file `internal/shared/pg_tx_runner.go`: `pgxTxRunner` implementation.

### Phase 2: `internal/identityuser` (New Package)

- New file `internal/identityuser/domain.go`: `User` type.

### Phase 3: `internal/user` Refactor

| File | Change |
|---|---|
| `domain.go` | Remove `User`. Keep `UpdateUserRequest` (no json tags). |
| `repo.go` | Add unexported `userRow` and `(r userRow) toDomain() identityuser.User`. Add `defaultQuerier Querier` field. Add `q(ctx)` helper. Update `UserRepository` interface to return `*userRow`. Update pgx methods to use `r.q(ctx)`. |
| `service.go` | After each repo call, call `row.toDomain()`. Return `*identityuser.User` (not `*User`). |
| `handler.go` | Add inline DTOs (`userResponse`, `listUsersResponse`, `updateUserRequestDTO`) with json tags. Handler maps DTO ↔ domain. |
| `grpc.go` | Build protobuf responses from `*identityuser.User`. |
| `mocks/repo_mock.go` | Regenerated by `go generate ./...`. |

### Phase 4: `internal/auth` Refactor

| File | Change |
|---|---|
| `domain.go` | Remove `User`, `Session`. Add `UserCredentials`. Add `SessionContext`, `SessionWithContext`. Keep `AuthResult` (uses `identityuser.User`), `Credentials`, `ChangePasswordInput`, `ChangeEmailInput`, `PasswordResetToken`. |
| `repo.go` | Update interfaces to return `*userCredentialsRow` (User repo) and `*sessionContextRow` (Session repo). Add `defaultQuerier` field and `q(ctx)` helper. |
| `repo_user.go` | Rename current `User` references to `userCredentialsRow`. Add `(r userCredentialsRow) toDomain() UserCredentials`. |
| `repo_session.go` | Add unexported `sessionContextRow` (with `RefreshToken`, `DeviceInfo []byte`, `IPAddress`). Add `toSession()` and `toContext()` methods returning `session.Session` and `SessionContext`. |
| `service.go` | Add `txr shared.TxRunner` field to the struct (stored but unused in this refactor; follow-up work will use it). Update constructor signature to accept it. |
| `service_register.go`, `service_login.go` | Document intended `RunInTx` usage in code comments. Do not wire in this refactor. |
| `service_change_password.go`, `service_change_email.go`, `service_forgot_password.go`, `service_refresh.go`, `service_reset_password.go`, `service_logout.go` | Update conversions. |
| `handler_*.go` | Add DTOs with json tags. Map DTO ↔ service inputs. |
| `grpc.go` | Build protobuf responses from domain types. |
| `mocks/repo_mock.go` | Regenerated. |
| `service_test.go`, `service_change_email_test.go`, `service_change_password_test.go` | Mocks return rows. Service returns `*identityuser.User` or `*UserCredentials`. Existing assertions updated. |
| `repo_integration_test.go` | Minor updates — same test logic, new row types. |

### Phase 5: `internal/session` Refactor

| File | Change |
|---|---|
| `domain.go` | Pure `Session` (no `DeviceInfo`/`IPAddress`/`RefreshToken`). |
| `repo.go` | Add `sessionRow`, return rows. |
| `service.go` | Convert rows. |
| `handler.go`, `grpc.go` | DTOs. |
| `mocks/repo_mock.go` | Regenerated. |

### Phase 6: `internal/role` Refactor

| File | Change |
|---|---|
| `domain.go` | Keep `Role` and request types — drop json tags. |
| `repo.go` | Add `roleRow`, return rows. |
| `service.go` | Convert. |
| `handler.go`, `grpc.go` | DTOs. |
| `mocks/repo_mock.go` | Regenerated. |

### Phase 7: `cmd/server/main.go` Wiring

```go
fx.Provide(
    // ... existing providers ...
    shared.NewPgxTxRunner,
    fx.Annotate(user.NewUserPGRepository, fx.As(new(user.UserRepository))),
    fx.Annotate(auth.NewAuthUserPGRepository, fx.As(new(auth.UserRepository))),
    fx.Annotate(auth.NewAuthSessionPGRepository, fx.As(new(auth.SessionRepository))),
    fx.Annotate(session.NewSessionPGRepository, fx.As(new(session.SessionRepository))),
    fx.Annotate(role.NewRolePGRepository, fx.As(new(role.RoleRepository))),
    user.NewUserService,         // (repo, rbac)
    auth.NewAuthService,         // (userRepo, sessionRepo, txr, hasher, tokenSvc, rbac, emailSender, refreshTTL)
    session.NewSessionService,   // (repo, rbac)
    role.NewRoleService,         // (repo, rbac)
)
```

The `txr shared.TxRunner` is provided but unused in this refactor. `auth.NewAuthService` accepts the parameter and stores it; future work uses it.

### Phase 8: Verification

1. `go build ./...` — must compile.
2. `go vet ./...` — clean.
3. `go test ./...` — all unit tests pass.
4. `go test -tags=integration ./...` — integration tests pass.
5. `go generate ./...` — regenerates all mocks.
6. Manual smoke test: `docker compose up`, hit `/api/v1/users/{id}` and a login flow. JSON shape unchanged.

## Future Work (Out of Scope)

The following are deliberately not part of this refactor; the design supports them but they require follow-up work:

- **Wiring transactions into existing flows.** `auth.AuthService.Register` and `auth.AuthService.Login` are the natural first candidates. Once they're wrapped in `s.txr.RunInTx`, the integrity guarantees apply without any other code change.
- **Splitting the service.** The `identityuser` package is the natural seam: when auth and user become separate services, `identityuser` either becomes a shared library, gets duplicated, or one service owns it via RPC. The domain types are already transport-free, which makes any of these options straightforward.
- **Replacing Postgres.** The `Querier` interface is satisfied by `*pgxpool.Pool` and `pgx.Tx`. A new store would need a new `Querier` implementation; services and domain types would not change.
- **Removing `DeviceInfo` JSON round-tripping.** Currently the auth repo marshals/unmarshals `DeviceInfo` on every read/write. A future change could store it as a typed value, but that's a separate refactor.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Forgetting to call `WithQuerier` on a tx ctx, so queries run outside the tx | Doc comment on `WithQuerier` warns. Code review checks for `RunInTx` callers. Tier 2 integration tests cover the happy path; a future test deliberately injects a failure mid-`fn` to verify rollback. |
| Service accidentally importing pgx | Code review and `go vet` checks. The `txr` field type is `shared.TxRunner` (interface), not `*pgxpool.Pool`. |
| Mocks out of sync with interfaces | `go generate ./...` is part of the verification step. CI runs it. |
| Cross-module import direction inverted | Documented in this spec; code review. `goimports` and `go vet` don't catch it; a custom lint rule could be added later. |
| `DeviceInfo` JSONB shape change | The `[]byte` field on `sessionContextRow` and the unmarshal in `toContext()` isolate the JSON shape. Changing the DB column shape is a single-file change. |

## File Summary

**New files** (6):

- `internal/identityuser/domain.go`
- `internal/shared/querier.go`
- `internal/shared/tx.go`
- `internal/shared/pg_tx_runner.go`
- `docs/superpowers/specs/2026-06-21-identity-domain-db-separation-design.md` (this file)
- `docs/superpowers/plans/2026-06-21-identity-domain-db-separation-plan.md` (the implementation plan, written via writing-plans skill)

**Modified files** (~25): all of `internal/user/`, `internal/auth/`, `internal/session/`, `internal/role/`, and `cmd/server/main.go`. The exact set is enumerated in Phase 3–7 above.

**Regenerated files**: every `mocks/repo_mock.go` under the four modules.
