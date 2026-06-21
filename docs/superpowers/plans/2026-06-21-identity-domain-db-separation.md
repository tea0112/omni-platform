# Identity Service Domain/DB Model Separation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Separate domain types from database row types in the identity service. Domain types become pure (no `json` tags, no DB-shape concerns); row types live next to the pgx implementation with explicit `toDomain()` conversion methods; services do the conversion; transactions become possible via a `TxRunner` interface without leaking pgx into the service layer.

**Architecture:** Single package per module (`auth`, `user`, `session`, `role`). Each module's `domain.go` holds pure types. Each module's `repo.go` defines unexported row types and returns them from the repository interface. Services call `row.toDomain()` at the boundary. A new `internal/identityuser` package owns the shared `User` type used by both `auth` and `user`. A new `internal/shared/querier.go` defines a `Querier` interface and a context-based lookup; a new `internal/shared/tx.go` defines a `TxRunner` interface with a pgx implementation in `internal/shared/pg_tx_runner.go`. Repositories look up the active querier (pool or tx) from context.

**Tech Stack:** Go (same version as the existing module), pgx/v5, go.uber.org/mock/gomock, testcontainers-go/modules/postgres (for integration), uber fx (for DI).

## Global Constraints

- Go version: matches `services/identity/go.mod`
- Existing module: `github.com/tea0112/omni-platform/services/identity`
- Existing JSON wire format must be preserved (no client-visible changes)
- All four modules (`auth`, `user`, `session`, `role`) refactored in this single change
- Mock regeneration is part of every module refactor: `cd services/identity && go generate ./...`
- The service layer must not import `github.com/jackc/pgx/v5` or `github.com/jackc/pgx/v5/pgxpool`
- Domain types must not carry `json` tags
- Domain types must not import pgx, pgxpool, encoding/json (except where strictly required for DTOs at the transport boundary)
- Test files use `//go:build integration` for testcontainers tests
- Sentinel errors live in `internal/shared/errors.go` (unchanged)
- All commits use the `refactor(identity):` prefix
- After each task, run `go build ./...` and `go test ./...` from `services/identity`
- Working directory for every step: `services/identity`

---

## Task 1: Add `Querier` and `TxRunner` to `internal/shared`

**Files:**
- Create: `services/identity/internal/shared/querier.go`
- Create: `services/identity/internal/shared/tx.go`
- Create: `services/identity/internal/shared/pg_tx_runner.go`
- Test: `services/identity/internal/shared/pg_tx_runner_integration_test.go`

**Interfaces:**
- Produces: `Querier` interface, `WithQuerier(ctx, q) ctx`, `QuerierFromContext(ctx) Querier` (consumed by every module's `repo.go` in later tasks)
- Produces: `TxRunner` interface with `RunInTx(ctx, fn) error` (consumed by `auth.NewAuthService` in Task 4)
- Produces: `NewPgxTxRunner(pool) TxRunner` (consumed by `cmd/server/main.go` in Task 7)

- [ ] **Step 1: Create `internal/shared/querier.go`**

Create the file `services/identity/internal/shared/querier.go` with this exact content:

```go
package shared

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type ctxKey struct{}

var querierKey ctxKey

// WithQuerier attaches a Querier (typically a pgx.Tx) to ctx so that
// repository methods called downstream will use it instead of the
// repository's default pool. Callers MUST pass the returned ctx to every
// repo call that should run inside the transaction. Forgetting to do so
// will cause those calls to run outside the tx and silently bypass it.
func WithQuerier(ctx context.Context, q Querier) context.Context {
	return context.WithValue(ctx, querierKey, q)
}

func QuerierFromContext(ctx context.Context) Querier {
	q, _ := ctx.Value(querierKey).(Querier)
	return q
}
```

- [ ] **Step 2: Create `internal/shared/tx.go`**

Create the file `services/identity/internal/shared/tx.go` with this exact content:

```go
package shared

import "context"

type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

- [ ] **Step 3: Create `internal/shared/pg_tx_runner.go`**

Create the file `services/identity/internal/shared/pg_tx_runner.go` with this exact content:

```go
package shared

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxTxRunner struct {
	pool *pgxpool.Pool
}

func NewPgxTxRunner(pool *pgxpool.Pool) TxRunner {
	return &pgxTxRunner{pool: pool}
}

func (r *pgxTxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(WithQuerier(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Compile-time check that pgx.Tx satisfies Querier.
var _ Querier = (pgx.Tx)(nil)
```

- [ ] **Step 4: Verify the package compiles**

Run: `cd services/identity && go build ./internal/shared/...`
Expected: no output, exit code 0.

- [ ] **Step 5: Create the integration test scaffold**

Create the file `services/identity/internal/shared/pg_tx_runner_integration_test.go` with this exact content (no real DB calls yet — this is a placeholder that compiles; Task 8 expands it):

```go
//go:build integration

package shared_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/tea0112/omni-platform/services/identity/internal/migrate"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:18.4-alpine",
		postgres.WithDatabase("identity"),
		postgres.WithUsername("identity"),
		postgres.WithPassword("identity"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { container.Terminate(ctx) })

	dsn, _ := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, migrate.Run(dsn))

	pool, err := shared.NewDBPool(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func TestPgxTxRunner_Commits(t *testing.T) {
	pool := newTestPool(t)
	runner := shared.NewPgxTxRunner(pool)
	called := false
	err := runner.RunInTx(context.Background(), func(ctx context.Context) error {
		called = true
		// The querier in ctx should be a pgx.Tx
		q := shared.QuerierFromContext(ctx)
		require.NotNil(t, q)
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestPgxTxRunner_RollsBackOnError(t *testing.T) {
	pool := newTestPool(t)
	runner := shared.NewPgxTxRunner(pool)
	sentinel := context.Canceled
	err := runner.RunInTx(context.Background(), func(ctx context.Context) error {
		// Insert a row, then return an error so the tx rolls back.
		_, err := shared.QuerierFromContext(ctx).Exec(ctx,
			`INSERT INTO users (id, email, password_hash) VALUES (gen_random_uuid(), 'tx@test.com', 'x')`)
		require.NoError(t, err)
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)
	// Verify the row was rolled back.
	var count int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM users WHERE email = 'tx@test.com'`).Scan(&count))
	require.Equal(t, 0, count)
}
```

- [ ] **Step 6: Run the integration test**

Run: `cd services/identity && go test -tags=integration ./internal/shared/... -run TestPgxTxRunner -v`
Expected: both `TestPgxTxRunner_Commits` and `TestPgxTxRunner_RollsBackOnError` PASS.

If `migrate.Run` or `shared.NewDBPool` don't exist with those signatures, look up the existing call sites in `internal/auth/repo_integration_test.go` and adjust the test to match (the existing test in the repo uses the same pattern).

- [ ] **Step 7: Commit**

```bash
cd services/identity && git add internal/shared/querier.go internal/shared/tx.go internal/shared/pg_tx_runner.go internal/shared/pg_tx_runner_integration_test.go
git commit -m "refactor(identity): add Querier and TxRunner to shared"
```

---

## Task 2: Create the `internal/identityuser` package

**Files:**
- Create: `services/identity/internal/identityuser/domain.go`

**Interfaces:**
- Produces: `identityuser.User` struct (consumed by `auth.UserCredentials` in Task 4, by `user.service` in Task 3, and by handlers/grpc in Tasks 3 and 4)

- [ ] **Step 1: Create the package directory**

Run: `mkdir -p services/identity/internal/identityuser`

- [ ] **Step 2: Create `internal/identityuser/domain.go`**

Create the file with this exact content:

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

Note: no `json` tags, no DB tags, no methods. This is a pure data type.

- [ ] **Step 3: Verify the package compiles**

Run: `cd services/identity && go build ./internal/identityuser/...`
Expected: no output, exit code 0.

- [ ] **Step 4: Verify the rest of the module still builds**

Run: `cd services/identity && go build ./...`
Expected: no output, exit code 0. (The new package is not yet imported anywhere, so this just confirms the package itself is valid Go.)

- [ ] **Step 5: Commit**

```bash
cd services/identity && git add internal/identityuser/domain.go
git commit -m "refactor(identity): add identityuser package with shared User type"
```

---

## Task 3: Refactor the `internal/user` module

**Files:**
- Modify: `services/identity/internal/user/domain.go`
- Modify: `services/identity/internal/user/repo.go`
- Modify: `services/identity/internal/user/service.go`
- Modify: `services/identity/internal/user/handler.go`
- Modify: `services/identity/internal/user/grpc.go`
- Regenerate: `services/identity/internal/user/mocks/repo_mock.go` (via `go generate ./...`)

**Interfaces:**
- Consumes: `identityuser.User` (from Task 2), `shared.Querier`, `shared.WithQuerier`, `shared.QuerierFromContext` (from Task 1)
- Produces: `user.UserRepository` interface returning `*UserRow`, `user.UserService` returning `*identityuser.User`

- [ ] **Step 1: Read the current files to understand the existing structure**

Read in full:
- `services/identity/internal/user/domain.go`
- `services/identity/internal/user/repo.go`
- `services/identity/internal/user/service.go`
- `services/identity/internal/user/handler.go`
- `services/identity/internal/user/grpc.go`

Confirm that the existing `User` struct has fields `ID, Email, DisplayName, EmailVerified, CreatedAt, UpdatedAt` (matching what `identityuser.User` will provide).

- [ ] **Step 2: Rewrite `internal/user/domain.go`**

Replace the entire file with:

```go
package user

type UpdateUserRequest struct {
	DisplayName *string
}
```

The old `User` struct is removed; this module now imports `identityuser.User` when it needs a user value.

- [ ] **Step 3: Rewrite `internal/user/repo.go`**

Replace the entire file with:

```go
package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tea0112/omni-platform/services/identity/internal/identityuser"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . UserRepository

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*UserRow, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*UserRow, error)
	List(ctx context.Context, offset, limit int) ([]UserRow, error)
}

type UserRow struct {
	ID            uuid.UUID
	Email         string
	DisplayName   string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (r UserRow) toDomain() identityuser.User {
	return identityuser.User{
		ID:            r.ID,
		Email:         r.Email,
		DisplayName:   r.DisplayName,
		EmailVerified: r.EmailVerified,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

type UserPGRepository struct {
	defaultQuerier shared.Querier
}

var _ UserRepository = (*UserPGRepository)(nil)

func NewUserPGRepository(pool *pgxpool.Pool) *UserPGRepository {
	return &UserPGRepository{defaultQuerier: pool}
}

func (r *UserPGRepository) q(ctx context.Context) shared.Querier {
	if txQ := shared.QuerierFromContext(ctx); txQ != nil {
		return txQ
	}
	return r.defaultQuerier
}

func (r *UserPGRepository) GetByID(ctx context.Context, id uuid.UUID) (*UserRow, error) {
	row := &UserRow{}
	err := r.q(ctx).QueryRow(ctx,
		`SELECT id, email, display_name, email_verified, created_at, updated_at FROM users WHERE id = $1`, id,
	).Scan(&row.ID, &row.Email, &row.DisplayName, &row.EmailVerified, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return row, nil
}

func (r *UserPGRepository) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*UserRow, error) {
	if req.DisplayName != nil {
		_, err := r.q(ctx).Exec(ctx,
			`UPDATE users SET display_name = $1, updated_at = now() WHERE id = $2`, *req.DisplayName, id)
		if err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
	}
	return r.GetByID(ctx, id)
}

func (r *UserPGRepository) List(ctx context.Context, offset, limit int) ([]UserRow, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT id, email, display_name, email_verified, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []UserRow
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}
```

- [ ] **Step 4: Rewrite `internal/user/service.go`**

Replace the entire file with:

```go
package user

import (
	"context"

	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/identityuser"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type UserService struct {
	repo UserRepository
	rbac *shared.RBAC
}

func NewUserService(repo UserRepository, rbac *shared.RBAC) *UserService {
	return &UserService{repo: repo, rbac: rbac}
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*identityuser.User, error) {
	p, _ := shared.GetPrincipal(ctx)
	if p.UserID != id.String() {
		if err := s.rbac.Can(ctx, "users.read"); err != nil {
			return nil, err
		}
	}
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	u := row.toDomain()
	return &u, nil
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*identityuser.User, error) {
	p, _ := shared.GetPrincipal(ctx)
	if p.UserID != id.String() {
		if err := s.rbac.Can(ctx, "users.write", id.String()); err != nil {
			return nil, err
		}
	}
	row, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	u := row.toDomain()
	return &u, nil
}

func (s *UserService) List(ctx context.Context, offset, limit int) ([]identityuser.User, error) {
	if err := s.rbac.Can(ctx, "users.read"); err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	users := make([]identityuser.User, len(rows))
	for i, r := range rows {
		users[i] = r.toDomain()
	}
	return users, nil
}
```

- [ ] **Step 5: Rewrite `internal/user/handler.go`**

Replace the entire file with:

```go
package user

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/identityuser"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type Handler struct {
	svc *UserService
}

func NewHandler(svc *UserService) *Handler {
	return &Handler{svc: svc}
}

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
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

type updateUserRequestDTO struct {
	DisplayName *string `json:"display_name"`
}

func (d updateUserRequestDTO) toDomain() UpdateUserRequest {
	return UpdateUserRequest{DisplayName: d.DisplayName}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/users/{id}", h.GetByID)
	r.Patch("/api/v1/users/{id}", h.Update)
	r.Get("/api/v1/users", h.List)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, toUserResponse(u))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	var dto updateUserRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		shared.WriteErr(w, err)
		return
	}
	u, err := h.svc.Update(r.Context(), id, dto.toDomain())
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, toUserResponse(u))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	users, err := h.svc.List(r.Context(), offset, limit)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = toUserResponse(&u)
	}
	shared.WriteJSON(w, http.StatusOK, resp)
}
```

- [ ] **Step 6: Rewrite `internal/user/grpc.go`**

Replace the entire file with:

```go
package user

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	identityv1 "github.com/tea0112/omni-platform/services/identity/gen/identity/v1"
	"github.com/tea0112/omni-platform/services/identity/gen/identity/v1/identityv1connect"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

var _ identityv1connect.UserServiceHandler = (*UserGrpcHandler)(nil)

type UserGrpcHandler struct {
	svc *UserService
}

func NewUserGrpcHandler(svc *UserService) (string, http.Handler) {
	handler := &UserGrpcHandler{svc: svc}
	return identityv1connect.NewUserServiceHandler(handler)
}

func (h *UserGrpcHandler) GetUser(ctx context.Context, req *connect.Request[identityv1.GetUserRequest]) (*connect.Response[identityv1.GetUserResponse], error) {
	id, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"user_id": "invalid uuid"}})
	}
	u, err := h.svc.GetByID(ctx, id)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.GetUserResponse{
		UserId:      u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
	}), nil
}

func (h *UserGrpcHandler) UpdateUser(ctx context.Context, req *connect.Request[identityv1.UpdateUserRequest]) (*connect.Response[identityv1.UpdateUserResponse], error) {
	id, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"user_id": "invalid uuid"}})
	}
	u, err := h.svc.Update(ctx, id, UpdateUserRequest{DisplayName: req.Msg.DisplayName})
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.UpdateUserResponse{
		UserId:      u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
	}), nil
}

func (h *UserGrpcHandler) ListUsers(ctx context.Context, req *connect.Request[identityv1.ListUsersRequest]) (*connect.Response[identityv1.ListUsersResponse], error) {
	offset := int(req.Msg.Offset)
	limit := int(req.Msg.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	users, err := h.svc.List(ctx, offset, limit)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	resp := &identityv1.ListUsersResponse{}
	for i := range users {
		u := users[i]
		resp.Users = append(resp.Users, &identityv1.GetUserResponse{
			UserId:      u.ID.String(),
			Email:       u.Email,
			DisplayName: u.DisplayName,
		})
	}
	return connect.NewResponse(resp), nil
}
```

- [ ] **Step 7: Regenerate mocks**

Run: `cd services/identity && go generate ./...`
Expected: regenerates `internal/user/mocks/repo_mock.go` and any other mock files. The new mock returns `*UserRow` instead of `*user`.

- [ ] **Step 8: Build and check for errors**

Run: `cd services/identity && go build ./...`
Expected: failure. The `cmd/server/main.go` still uses the old `auth.NewAuthRepository` and other old constructors; that's expected. The error should be ONLY in `cmd/server/main.go` and the auth/session/role modules. Confirm by reading the error output.

If the user module itself fails to build, fix it before proceeding.

- [ ] **Step 9: Commit the user module refactor**

```bash
cd services/identity && git add internal/user/
git commit -m "refactor(identity): separate domain and row types in user module"
```

---

## Task 4: Refactor the `internal/auth` module

**Files:**
- Modify: `services/identity/internal/auth/domain.go`
- Modify: `services/identity/internal/auth/repo.go`
- Modify: `services/identity/internal/auth/repo_user.go`
- Modify: `services/identity/internal/auth/repo_session.go`
- Modify: `services/identity/internal/auth/service.go`
- Modify: `services/identity/internal/auth/service_*.go` (8 files)
- Modify: `services/identity/internal/auth/handler.go`, `handler_change_email.go`, `handler_change_password.go`, `handler_forgot_password.go`, `handler_login.go`, `handler_logout.go`, `handler_refresh.go`, `handler_register.go`, `handler_reset_password.go`
- Modify: `services/identity/internal/auth/grpc.go`
- Modify: `services/identity/internal/auth/service_test.go`, `service_change_email_test.go`, `service_change_password_test.go`
- Modify: `services/identity/internal/auth/repo_integration_test.go`
- Regenerate: `services/identity/internal/auth/mocks/repo_mock.go`

This is the largest task. The approach:
1. Replace `domain.go` with the new types
2. Replace `repo.go` with the new interfaces
3. Replace `repo_user.go` and `repo_session.go` with row types and pgx implementations
4. Update each `service_*.go` to convert rows and use `identityuser.User` / `UserCredentials`
5. Update each `handler_*.go` with DTOs
6. Update `grpc.go` to build responses from domain types
7. Update test files
8. Regenerate mocks
9. Build, fix, commit

**Interfaces:**
- Consumes: `identityuser.User` (from Task 2), `shared.Querier`, `shared.TxRunner` (from Task 1)
- Produces: `auth.UserCredentials`, `auth.UserRepository` (returns `*UserCredentialsRow`), `auth.SessionRepository` (returns `*SessionContextRow`), `auth.NewAuthService` accepting `shared.TxRunner`

- [ ] **Step 1: Read all current auth files**

Read in full:
- `services/identity/internal/auth/domain.go`
- `services/identity/internal/auth/repo.go`
- `services/identity/internal/auth/repo_user.go`
- `services/identity/internal/auth/repo_session.go`
- `services/identity/internal/auth/service.go` and all `service_*.go`
- `services/identity/internal/auth/handler.go` and all `handler_*.go`
- `services/identity/internal/auth/grpc.go`
- `services/identity/internal/auth/service_test.go`, `service_change_email_test.go`, `service_change_password_test.go`
- `services/identity/internal/auth/repo_integration_test.go`

Make a list of every method on `UserRepository` and `SessionRepository` so you can preserve them.

- [ ] **Step 2: Rewrite `internal/auth/domain.go`**

Replace the entire file with:

```go
package auth

import (
	"time"

	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/identityuser"
)

type UserCredentials struct {
	user         identityuser.User
	passwordHash string
}

func NewUserCredentials(u identityuser.User, hash string) *UserCredentials {
	return &UserCredentials{user: u, passwordHash: hash}
}

func (c *UserCredentials) User() identityuser.User { return c.user }
func (c *UserCredentials) PasswordHash() string    { return c.passwordHash }

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	User         identityuser.User
}

type Credentials struct {
	Email    string
	Password string
}

type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

type ChangeEmailInput struct {
	CurrentPassword string
	NewEmail        string
}

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type SessionContext struct {
	RefreshToken string
	DeviceInfo   map[string]any
	IPAddress    string
}

// SessionWithContext is a flat struct the auth service populates by
// combining the row's toSession() and toContext() conversion methods. We
// keep it flat (not embedded) so the auth package does not need to import
// the session package — that would invert the dependency direction.
type SessionWithContext struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	RefreshToken string
	DeviceInfo   map[string]any
	IPAddress    string
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}
```

- [ ] **Step 3: Rewrite `internal/auth/repo.go`**

Replace the entire file with:

```go
package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . UserRepository,SessionRepository

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*UserCredentialsRow, error)
	GetByEmail(ctx context.Context, email string) (*UserCredentialsRow, error)
	GetByID(ctx context.Context, id uuid.UUID) (*UserCredentialsRow, error)
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	GetPasswordResetToken(ctx context.Context, token string) (userID uuid.UUID, expiresAt time.Time, usedAt *time.Time, err error)
	MarkPasswordResetTokenUsed(ctx context.Context, token string) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	UpdateEmail(ctx context.Context, userID uuid.UUID, email string) error
	GetUserRolesAndPermissions(ctx context.Context, userID uuid.UUID) (roles []string, perms []string, err error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, userID uuid.UUID, refreshToken string, deviceInfo map[string]any, ipAddress string, expiresAt time.Time) (*SessionContextRow, error)
	GetByRefreshToken(ctx context.Context, refreshToken string) (*SessionContextRow, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]SessionContextRow, error)
}

// AuthPGRepository implements both UserRepository and SessionRepository.
// It is constructed once and shared via fx as both interfaces.
type AuthPGRepository struct {
	defaultQuerier shared.Querier
}

var _ UserRepository = (*AuthPGRepository)(nil)
var _ SessionRepository = (*AuthPGRepository)(nil)

func NewAuthUserPGRepository(pool *pgxpool.Pool) *AuthPGRepository {
	return &AuthPGRepository{defaultQuerier: pool}
}

// NewAuthSessionPGRepository returns the same struct. In fx wiring both
// User and Session repos are provided by the same struct instance.
func NewAuthSessionPGRepository(repo *AuthPGRepository) *AuthPGRepository {
	return repo
}

func (r *AuthPGRepository) q(ctx context.Context) shared.Querier {
	if txQ := shared.QuerierFromContext(ctx); txQ != nil {
		return txQ
	}
	return r.defaultQuerier
}
```

- [ ] **Step 4: Rewrite `internal/auth/repo_user.go`**

Replace the entire file with:

```go
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type UserCredentialsRow struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  string
	DisplayName   string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (r UserCredentialsRow) toDomain() *UserCredentials {
	return NewUserCredentials(
		identityuser.User{
			ID:            r.ID,
			Email:         r.Email,
			DisplayName:   r.DisplayName,
			EmailVerified: r.EmailVerified,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		},
		r.PasswordHash,
	)
}

func (r *AuthPGRepository) Create(ctx context.Context, email, passwordHash string) (*UserCredentialsRow, error) {
	id := uuid.Must(uuid.NewV7())
	_, err := r.q(ctx).Exec(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		id, email, passwordHash,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *AuthPGRepository) GetByEmail(ctx context.Context, email string) (*UserCredentialsRow, error) {
	row := &UserCredentialsRow{}
	err := r.q(ctx).QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, email_verified, created_at, updated_at FROM users WHERE email = $1`,
		email,
	).Scan(&row.ID, &row.Email, &row.PasswordHash, &row.DisplayName, &row.EmailVerified, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return row, nil
}

func (r *AuthPGRepository) GetByID(ctx context.Context, id uuid.UUID) (*UserCredentialsRow, error) {
	row := &UserCredentialsRow{}
	err := r.q(ctx).QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, email_verified, created_at, updated_at FROM users WHERE id = $1`,
		id,
	).Scan(&row.ID, &row.Email, &row.PasswordHash, &row.DisplayName, &row.EmailVerified, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return row, nil
}

func (r *AuthPGRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	_, err := r.q(ctx).Exec(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.Must(uuid.NewV7()), userID, token, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

func (r *AuthPGRepository) GetPasswordResetToken(ctx context.Context, token string) (uuid.UUID, time.Time, *time.Time, error) {
	var userID uuid.UUID
	var expiresAt time.Time
	var usedAt *time.Time
	err := r.q(ctx).QueryRow(ctx,
		`SELECT user_id, expires_at, used_at FROM password_reset_tokens WHERE token = $1`,
		token,
	).Scan(&userID, &expiresAt, &usedAt)
	if err != nil {
		return uuid.UUID{}, time.Time{}, nil, fmt.Errorf("get password reset token: %w", err)
	}
	return userID, expiresAt, usedAt, nil
}

func (r *AuthPGRepository) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = now() WHERE token = $1`,
		token,
	)
	if err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}

func (r *AuthPGRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`,
		passwordHash, userID,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *AuthPGRepository) UpdateEmail(ctx context.Context, userID uuid.UUID, email string) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE users SET email = $1, email_verified = false, updated_at = now() WHERE id = $2`,
		email, userID,
	)
	if err != nil {
		return fmt.Errorf("update email: %w", err)
	}
	return nil
}

func (r *AuthPGRepository) GetUserRolesAndPermissions(ctx context.Context, userID uuid.UUID) ([]string, []string, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT r.name, rp.permission
		 FROM user_roles ur
		 JOIN roles r ON r.id = ur.role_id
		 JOIN role_permissions rp ON rp.role_id = r.id
		 WHERE ur.user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("get user roles and permissions: %w", err)
	}
	defer rows.Close()

	roleSet := make(map[string]struct{})
	permSet := make(map[string]struct{})
	for rows.Next() {
		var role, perm string
		if err := rows.Scan(&role, &perm); err != nil {
			return nil, nil, fmt.Errorf("scan role/permission: %w", err)
		}
		roleSet[role] = struct{}{}
		permSet[perm] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration: %w", err)
	}

	roles := make([]string, 0, len(roleSet))
	for r := range roleSet {
		roles = append(roles, r)
	}
	perms := make([]string, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}
	return roles, perms, nil
}
```

Add `import "github.com/tea0112/omni-platform/services/identity/internal/identityuser"` to the imports.

Note: the `r` variable in `for r := range roleSet` shadows the receiver `r` of the method. Go allows this but the lint may flag it. If you see a warning, rename the loop variable to `name`.

- [ ] **Step 5: Rewrite `internal/auth/repo_session.go`**

Replace the entire file with:

```go
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SessionContextRow struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	RefreshToken string
	DeviceInfo   []byte
	IPAddress    string
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}

func (r SessionContextRow) toSessionWithContext() (*SessionWithContext, error) {
	var device map[string]any
	if len(r.DeviceInfo) > 0 {
		if err := json.Unmarshal(r.DeviceInfo, &device); err != nil {
			return nil, fmt.Errorf("unmarshal device info: %w", err)
		}
	}
	return &SessionWithContext{
		ID:           r.ID,
		UserID:       r.UserID,
		RefreshToken: r.RefreshToken,
		DeviceInfo:   device,
		IPAddress:    r.IPAddress,
		ExpiresAt:    r.ExpiresAt,
		RevokedAt:    r.RevokedAt,
		CreatedAt:    r.CreatedAt,
	}, nil
}

func (r *AuthPGRepository) CreateSession(ctx context.Context, userID uuid.UUID, refreshToken string, deviceInfo map[string]any, ipAddress string, expiresAt time.Time) (*SessionContextRow, error) {
	deviceJSON, _ := json.Marshal(deviceInfo)
	id := uuid.Must(uuid.NewV7())
	_, err := r.q(ctx).Exec(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, device_info, ip_address, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, userID, refreshToken, deviceJSON, ipAddress, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return r.GetByRefreshToken(ctx, refreshToken)
}

func (r *AuthPGRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*SessionContextRow, error) {
	row := &SessionContextRow{}
	var deviceJSON []byte
	err := r.q(ctx).QueryRow(ctx,
		`SELECT id, user_id, refresh_token, device_info, ip_address, expires_at, revoked_at, created_at FROM sessions WHERE refresh_token = $1`,
		refreshToken,
	).Scan(&row.ID, &row.UserID, &row.RefreshToken, &deviceJSON, &row.IPAddress, &row.ExpiresAt, &row.RevokedAt, &row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get session by refresh token: %w", err)
	}
	row.DeviceInfo = deviceJSON
	return row, nil
}

func (r *AuthPGRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`,
		id,
	)
	return err
}

func (r *AuthPGRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	return err
}

func (r *AuthPGRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]SessionContextRow, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT id, user_id, refresh_token, device_info, ip_address, expires_at, revoked_at, created_at FROM sessions WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionContextRow
	for rows.Next() {
		var s SessionContextRow
		var deviceJSON []byte
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshToken, &deviceJSON, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.DeviceInfo = deviceJSON
		sessions = append(sessions, s)
	}
	return sessions, nil
}
```

- [ ] **Step 6: Rewrite `internal/auth/service.go`**

Replace the entire file with:

```go
package auth

import (
	"time"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type AuthService struct {
	userRepo        UserRepository
	sessionRepo     SessionRepository
	txr             shared.TxRunner
	hasher          *shared.PasswordHasher
	tokenSvc        *shared.TokenService
	rbac            *shared.RBAC
	emailSender     shared.EmailSender
	refreshTokenTTL time.Duration
}

func NewAuthService(
	userRepo UserRepository,
	sessionRepo SessionRepository,
	txr shared.TxRunner,
	hasher *shared.PasswordHasher,
	tokenSvc *shared.TokenService,
	rbac *shared.RBAC,
	emailSender shared.EmailSender,
	refreshTokenTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		sessionRepo:     sessionRepo,
		txr:             txr,
		hasher:          hasher,
		tokenSvc:        tokenSvc,
		rbac:            rbac,
		emailSender:     emailSender,
		refreshTokenTTL: refreshTokenTTL,
	}
}
```

Note: `txr` is stored but not used in this refactor; follow-up work will wrap `Register` and `Login` in `s.txr.RunInTx`.

- [ ] **Step 7: Update each `service_*.go`**

For each file (`service_change_email.go`, `service_change_password.go`, `service_forgot_password.go`, `service_login.go`, `service_logout.go`, `service_refresh.go`, `service_register.go`, `service_reset_password.go`), apply these transformations:

- Replace `*auth.User` with `*UserCredentials` everywhere it's the result of a repo call
- Call `creds.User()` to get the `*identityuser.User` for the response
- For services that take `identityuser.User` as input, the type signature already matches
- Update any helper that referenced `user.PasswordHash` to `creds.PasswordHash()`

A worked example for `service_login.go`:

```go
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (s *AuthService) Login(ctx context.Context, email, password, ipAddress string, deviceInfo map[string]any) (*AuthResult, error) {
	if email == "" || password == "" {
		return nil, &shared.ValidationError{Fields: map[string]string{
			"email":    "required",
			"password": "required",
		}}
	}

	creds, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, shared.ErrUnauthenticated
	}

	if err := s.hasher.Compare(creds.PasswordHash(), password); err != nil {
		return nil, shared.ErrUnauthenticated
	}

	roles, perms, err := s.userRepo.GetUserRolesAndPermissions(ctx, creds.User().ID)
	if err != nil {
		return nil, fmt.Errorf("get roles and permissions: %w", err)
	}

	accessToken, expiresAt, err := s.tokenSvc.GenerateAccessToken(creds.User().ID.String(), roles, perms)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken := uuid.Must(uuid.NewV7()).String()
	_, err = s.sessionRepo.CreateSession(ctx, creds.User().ID, refreshToken, deviceInfo, ipAddress, time.Now().Add(s.refreshTokenTTL))
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         creds.User(),
	}, nil
}
```

For `service_refresh.go`, replace `*auth.Session` with `*SessionWithContext` (from the row's `toSessionWithContext()`). For `service_logout.go` and `service_change_password.go`/`service_change_email.go`, the patterns are similar.

- [ ] **Step 8: Update each `handler_*.go`**

Each handler file gets DTOs added. For `handler_login.go`:

```go
type loginRequestDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
```

For `handler_register.go`:

```go
type registerRequestDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
```

For `handler_change_password.go`:

```go
type changePasswordRequestDTO struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
```

For `handler_change_email.go`:

```go
type changeEmailRequestDTO struct {
	CurrentPassword string `json:"current_password"`
	NewEmail        string `json:"new_email"`
}
```

For `handler_forgot_password.go` and `handler_reset_password.go`:

```go
type forgotPasswordRequestDTO struct {
	Email string `json:"email"`
}
type resetPasswordRequestDTO struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}
```

For `handler_refresh.go`:

```go
type refreshRequestDTO struct {
	RefreshToken string `json:"refresh_token"`
}
```

For `handler_logout.go`: no DTO needed (the request is empty).

The response DTOs are similar to before but built from `identityuser.User` and `*SessionWithContext`. Example for login/register response:

```go
type authResponseDTO struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         userResponse `json:"user"`
}
type userResponse struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	DisplayName   string    `json:"display_name"`
	EmailVerified bool      `json:"email_verified"`
}
```

The JSON shape is identical to today's output (no `created_at`/`updated_at` in the auth response, matching current behavior).

- [ ] **Step 9: Rewrite `internal/auth/grpc.go`**

Replace the entire file with:

```go
package auth

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	identityv1 "github.com/tea0112/omni-platform/services/identity/gen/identity/v1"
	"github.com/tea0112/omni-platform/services/identity/gen/identity/v1/identityv1connect"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

var _ identityv1connect.AuthServiceHandler = (*AuthGrpcHandler)(nil)

type AuthGrpcHandler struct {
	svc *AuthService
}

func NewAuthGrpcHandler(svc *AuthService) (string, http.Handler) {
	return identityv1connect.NewAuthServiceHandler(&AuthGrpcHandler{svc: svc})
}

func (h *AuthGrpcHandler) Register(ctx context.Context, req *connect.Request[identityv1.RegisterRequest]) (*connect.Response[identityv1.RegisterResponse], error) {
	creds, err := h.svc.Register(ctx, req.Msg.Email, req.Msg.Password)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.RegisterResponse{
		UserId: creds.User().ID.String(),
		Email:  creds.User().Email,
	}), nil
}

func (h *AuthGrpcHandler) Login(ctx context.Context, req *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error) {
	result, err := h.svc.Login(ctx, req.Msg.Email, req.Msg.Password, "", nil)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt.Unix(),
	}), nil
}

func (h *AuthGrpcHandler) Refresh(ctx context.Context, req *connect.Request[identityv1.RefreshRequest]) (*connect.Response[identityv1.RefreshResponse], error) {
	result, err := h.svc.Refresh(ctx, req.Msg.RefreshToken, "", nil)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.RefreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt.Unix(),
	}), nil
}

func (h *AuthGrpcHandler) Logout(ctx context.Context, req *connect.Request[identityv1.LogoutRequest]) (*connect.Response[identityv1.LogoutResponse], error) {
	p, ok := shared.GetPrincipal(ctx)
	if !ok {
		return nil, shared.AsConnectError(shared.ErrUnauthenticated)
	}
	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"user_id": "invalid"}})
	}
	if err := h.svc.Logout(ctx, userID); err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.LogoutResponse{}), nil
}
```

Match the existing gRPC handler structure. The proto types embed user fields directly in each response (no shared `identityv1.User` message). `ExpiresAt` is `int64` (Unix seconds), not a protobuf timestamp — use `.Unix()`.

- [ ] **Step 10: Update test files**

For `service_test.go`, `service_change_email_test.go`, `service_change_password_test.go`:

- Update mock setup to add the `txr` argument to `NewAuthService`:
  ```go
  svc := auth.NewAuthService(userRepo, sessionRepo, nil, hasher, nil, shared.NewRBAC(), &mockEmailSender{}, 30*24*time.Hour)
  ```
  The `nil` is for `txr` (no tx in unit tests). When transactions are added, this becomes a `&fakeTxRunner{}`.

- Update mock return values from `&auth.User{...}` to `&auth.UserCredentialsRow{...}`:
  ```go
  userRepo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(&auth.UserCredentialsRow{
      ID: userID, Email: "test@example.com", PasswordHash: hash,
      DisplayName: "", EmailVerified: false, CreatedAt: time.Time{}, UpdatedAt: time.Time{},
  }, nil)
  ```

- Update test assertions that read `user.PasswordHash` to `creds.PasswordHash()` (after capturing the result via `creds, err := svc.X(...)`).

- For `service_refresh.go` tests, update `&auth.Session{...}` to `&auth.SessionWithContext{...}` (with the same fields).

- [ ] **Step 11: Update `repo_integration_test.go`**

Update mock constructor calls: `auth.NewAuthRepository(pool)` → `auth.NewAuthUserPGRepository(pool)`. The test then calls methods on the returned `*AuthPGRepository`.

- [ ] **Step 12: Regenerate mocks**

Run: `cd services/identity && go generate ./...`
Expected: regenerates `internal/auth/mocks/repo_mock.go` to match the new interfaces.

- [ ] **Step 13: Build and check for errors**

Run: `cd services/identity && go build ./...`
Expected: failure only in `cmd/server/main.go` (still using old constructors). Confirm by reading the error output. If the auth module itself fails, fix it.

- [ ] **Step 14: Run unit tests**

Run: `cd services/identity && go test ./internal/auth/...`
Expected: all tests pass.

- [ ] **Step 15: Commit**

```bash
cd services/identity && git add internal/auth/
git commit -m "refactor(identity): separate domain and row types in auth module"
```

---

## Task 5: Refactor the `internal/session` module

**Files:**
- Modify: `services/identity/internal/session/domain.go`
- Modify: `services/identity/internal/session/repo.go`
- Modify: `services/identity/internal/session/service.go`
- Modify: `services/identity/internal/session/handler.go`
- Modify: `services/identity/internal/session/grpc.go`
- Regenerate: `services/identity/internal/session/mocks/repo_mock.go`

**Interfaces:**
- Consumes: `shared.Querier`, `shared.WithQuerier`, `shared.QuerierFromContext` (from Task 1)
- Produces: `session.SessionRepository` returning `[]SessionRow`, `session.NewSessionPGRepository(pool)` constructor

- [ ] **Step 1: Read current files**

Read `internal/session/domain.go`, `repo.go`, `service.go`, `handler.go`, `grpc.go` to understand the current shape.

- [ ] **Step 2: Rewrite `internal/session/domain.go`**

```go
package session

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
```

The `DeviceInfo`, `IPAddress`, and `RefreshToken` fields are removed. They live in `auth.SessionContext` (defined in Task 4).

- [ ] **Step 3: Rewrite `internal/session/repo.go`**

```go
package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . SessionRepository

type SessionRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]SessionRow, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type SessionRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (r SessionRow) toDomain() Session {
	return Session{
		ID:        r.ID,
		UserID:    r.UserID,
		ExpiresAt: r.ExpiresAt,
		RevokedAt: r.RevokedAt,
		CreatedAt: r.CreatedAt,
	}
}

type SessionPGRepository struct {
	defaultQuerier shared.Querier
}

var _ SessionRepository = (*SessionPGRepository)(nil)

func NewSessionPGRepository(pool *pgxpool.Pool) *SessionPGRepository {
	return &SessionPGRepository{defaultQuerier: pool}
}

func (r *SessionPGRepository) q(ctx context.Context) shared.Querier {
	if txQ := shared.QuerierFromContext(ctx); txQ != nil {
		return txQ
	}
	return r.defaultQuerier
}

func (r *SessionPGRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]SessionRow, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT id, user_id, expires_at, revoked_at, created_at FROM sessions WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get sessions: %w", err)
	}
	defer rows.Close()
	var sessions []SessionRow
	for rows.Next() {
		var s SessionRow
		if err := rows.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *SessionPGRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE id = $1`, id)
	return err
}

func (r *SessionPGRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}
```

- [ ] **Step 4: Rewrite `internal/session/service.go`**

```go
package session

import (
	"context"

	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type SessionService struct {
	repo SessionRepository
	rbac *shared.RBAC
}

func NewSessionService(repo SessionRepository, rbac *shared.RBAC) *SessionService {
	return &SessionService{repo: repo, rbac: rbac}
}

func (s *SessionService) ListByUser(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	if err := s.rbac.Can(ctx, "sessions.read", userID.String()); err != nil {
		return nil, err
	}
	rows, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	sessions := make([]Session, len(rows))
	for i, r := range rows {
		sessions[i] = r.toDomain()
	}
	return sessions, nil
}

func (s *SessionService) Revoke(ctx context.Context, id uuid.UUID) error {
	if err := s.rbac.Can(ctx, "sessions.write"); err != nil {
		return err
	}
	return s.repo.Revoke(ctx, id)
}

func (s *SessionService) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.rbac.Can(ctx, "sessions.write", userID.String()); err != nil {
		return err
	}
	return s.repo.RevokeAllForUser(ctx, userID)
}
```

Adjust the RBAC permission strings to match what the existing service uses (read the current `service.go` to confirm).

- [ ] **Step 5: Rewrite `internal/session/handler.go` and `grpc.go`**

Update both files to:
- Add DTOs with json tags (e.g., `sessionResponse` with `ID, UserID, ExpiresAt, RevokedAt, CreatedAt`)
- Convert domain `Session` to DTO in handlers
- Build protobuf responses from `Session` in grpc.go

Follow the same pattern as the user module's `handler.go` and `grpc.go` from Task 3.

- [ ] **Step 6: Regenerate mocks**

Run: `cd services/identity && go generate ./...`

- [ ] **Step 7: Build and check**

Run: `cd services/identity && go build ./...`
Expected: failure only in `cmd/server/main.go`. The session module itself must compile.

- [ ] **Step 8: Run unit tests**

Run: `cd services/identity && go test ./internal/session/...`
Expected: tests pass (if any exist; otherwise the test command exits 0).

- [ ] **Step 9: Commit**

```bash
cd services/identity && git add internal/session/
git commit -m "refactor(identity): separate domain and row types in session module"
```

---

## Task 6: Refactor the `internal/role` module

**Files:**
- Modify: `services/identity/internal/role/domain.go`
- Modify: `services/identity/internal/role/repo.go`
- Modify: `services/identity/internal/role/service.go`
- Modify: `services/identity/internal/role/handler.go`
- Modify: `services/identity/internal/role/grpc.go`
- Regenerate: `services/identity/internal/role/mocks/repo_mock.go`

**Interfaces:**
- Consumes: `shared.Querier`, `shared.WithQuerier`, `shared.QuerierFromContext` (from Task 1)
- Produces: `role.RoleRepository` returning `*RoleRow`, `role.NewRolePGRepository(pool)` constructor

- [ ] **Step 1: Read current files**

Read `internal/role/domain.go`, `repo.go`, `service.go`, `handler.go`, `grpc.go`.

- [ ] **Step 2: Rewrite `internal/role/domain.go`**

```go
package role

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
}

type CreateRoleRequest struct {
	Name        string
	Description string
}

type UpdateRoleRequest struct {
	Name        *string
	Description *string
}

type AddPermissionRequest struct {
	Permission string
}
```

No `json` tags on any field.

- [ ] **Step 3: Rewrite `internal/role/repo.go`**

Follow the same pattern as `internal/user/repo.go` (Task 3) and `internal/session/repo.go` (Task 5):
- `RoleRow` struct with the same fields as `Role` but used as the row type
- `(r RoleRow) toDomain() Role` conversion method
- `RolePGRepository` with `defaultQuerier` field and `q(ctx)` helper
- `NewRolePGRepository(pool *pgxpool.Pool) *RolePGRepository`
- All `RoleRepository` interface methods return `*RoleRow` or `[]RoleRow`
- `GetPermissions` continues to return `[]string` (single-column query, no row wrapper)
- All SQL uses `r.q(ctx).Query(...)` / `r.q(ctx).Exec(...)` / `r.q(ctx).QueryRow(...)`
- Translate `pgx.ErrNoRows` to `shared.ErrNotFound`

- [ ] **Step 4: Rewrite `internal/role/service.go`**

Update all methods to call `row.toDomain()` after each repo call. RBAC strings unchanged.

- [ ] **Step 5: Rewrite `internal/role/handler.go` and `grpc.go`**

Add DTOs in `handler.go` with json tags. Build them from `*role.Role` values. Same pattern as user module.

- [ ] **Step 6: Regenerate mocks**

Run: `cd services/identity && go generate ./...`

- [ ] **Step 7: Build and check**

Run: `cd services/identity && go build ./...`
Expected: failure only in `cmd/server/main.go`. The role module itself must compile.

- [ ] **Step 8: Run unit tests**

Run: `cd services/identity && go test ./internal/role/...`
Expected: tests pass.

- [ ] **Step 9: Commit**

```bash
cd services/identity && git add internal/role/
git commit -m "refactor(identity): separate domain and row types in role module"
```

---

## Task 7: Update `cmd/server/main.go` wiring

**Files:**
- Modify: `services/identity/cmd/server/main.go`

- [ ] **Step 1: Read the current `main.go`**

Read the full file. Identify every `fx.Provide` and `fx.Invoke` entry that references the changed packages.

- [ ] **Step 2: Update the `fx.Provide` block**

The new `fx.Provide` block should be:

```go
fx.Provide(
    shared.MustLoad,
    NewLogger,
    NewPasswordHasherFromConfig,
    shared.NewRBAC,
    NewTokenServiceFromConfig,
    NewRefreshTokenTTL,
    NewEmailSenderFromConfig,
    NewDBPoolFromConfig,
    NewTracerProviderFromConfig,
    shared.NewPgxTxRunner,
    fx.Annotate(user.NewUserPGRepository, fx.As(new(user.UserRepository))),
    fx.Annotate(auth.NewAuthUserPGRepository, fx.As(new(auth.UserRepository)), fx.As(new(auth.SessionRepository))),
    fx.Annotate(session.NewSessionPGRepository, fx.As(new(session.SessionRepository))),
    fx.Annotate(role.NewRolePGRepository, fx.As(new(role.RoleRepository))),
    user.NewUserService,
    auth.NewAuthService,
    session.NewSessionService,
    role.NewRoleService,
    auth.NewHandler,
    user.NewHandler,
    session.NewHandler,
    role.NewHandler,
    fx.Annotated{Group: "grpc-handlers", Target: func(svc *auth.AuthService) GrpcHandlerPair {
        p, h := auth.NewAuthGrpcHandler(svc)
        return GrpcHandlerPair{Path: p, Handler: h, SkipAuth: true}
    }},
    fx.Annotated{Group: "grpc-handlers", Target: func(svc *user.UserService) GrpcHandlerPair {
        p, h := user.NewUserGrpcHandler(svc)
        return GrpcHandlerPair{Path: p, Handler: h}
    }},
    fx.Annotated{Group: "grpc-handlers", Target: func(svc *session.SessionService) GrpcHandlerPair {
        p, h := session.NewSessionGrpcHandler(svc)
        return GrpcHandlerPair{Path: p, Handler: h}
    }},
    fx.Annotated{Group: "grpc-handlers", Target: func(svc *role.RoleService) GrpcHandlerPair {
        p, h := role.NewRoleGrpcHandler(svc)
        return GrpcHandlerPair{Path: p, Handler: h}
    }},
),
```

Notes:
- `shared.NewPgxTxRunner` is provided; it depends on `*pgxpool.Pool` (which is already provided by `NewDBPoolFromConfig`). fx wires them.
- The single `auth.NewAuthUserPGRepository` constructor returns `*AuthPGRepository` which implements both `auth.UserRepository` and `auth.SessionRepository`. fx injects the same instance for both `As(...)` annotations.
- All other entries are unchanged from the current main.go.

- [ ] **Step 3: Build the whole module**

Run: `cd services/identity && go build ./...`
Expected: no output, exit code 0.

- [ ] **Step 4: Run unit tests**

Run: `cd services/identity && go test ./...`
Expected: all tests pass.

- [ ] **Step 5: Run vet**

Run: `cd services/identity && go vet ./...`
Expected: no output, exit code 0.

- [ ] **Step 6: Commit**

```bash
cd services/identity && git add cmd/server/main.go
git commit -m "refactor(identity): update main.go fx wiring for new repo constructors"
```

---

## Task 8: Final verification

- [ ] **Step 1: Run the full test suite (unit)**

Run: `cd services/identity && go test ./...`
Expected: all tests pass.

- [ ] **Step 2: Run the full test suite (integration)**

Run: `cd services/identity && go test -tags=integration ./...`
Expected: all tests pass, including the new `TestPgxTxRunner_*` and the existing `TestAuthRepository_*`.

- [ ] **Step 3: Run `go vet`**

Run: `cd services/identity && go vet ./...`
Expected: no output.

- [ ] **Step 4: Check for accidental pgx imports in service files**

Run: `cd services/identity && grep -rn "jackc/pgx" internal/user/ internal/auth/ internal/session/ internal/role/ | grep -v "_test.go" | grep -v "repo"`
Expected: no output. The only pgx imports in those modules should be in `repo*.go` files (and `mocks/`).

- [ ] **Step 5: Check for json tags on domain types**

Run: `cd services/identity && grep -rn 'json:"' internal/identityuser/ internal/user/domain.go internal/session/domain.go internal/role/domain.go internal/auth/domain.go`
Expected: no output. Domain types must not have json tags.

- [ ] **Step 6: Build the server binary**

Run: `cd services/identity && go build -trimpath -o /tmp/identity-server ./cmd/server`
Expected: binary created, no errors.

- [ ] **Step 7: Manual smoke test**

Start the database:
Run: `cd services/identity && docker compose up -d identity-postgres`

Wait for the database to be healthy:
Run: `sleep 5 && docker compose ps`

Run migrations and start the server (replace the JWK with a real one — see the README's `gen-jwk` command for how to generate one):
Run: `cd services/identity && export IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK='<real-jwk>' && /tmp/identity-server &`

Hit the health endpoint:
Run: `curl http://localhost:8080/healthz`
Expected: `ok`

Register a user:
Run: `curl -X POST http://localhost:8080/api/v1/auth/register -H 'Content-Type: application/json' -d '{"email":"smoke@test.com","password":"smokepw123"}'`
Expected: 201 (or 200) with a JSON body containing `id`, `email`, `display_name`, `email_verified`.

Get the user (replace `<id>` with the id from the previous response):
Run: `curl http://localhost:8080/api/v1/users/<id> -H "Authorization: Bearer <token>"`
Expected: 200 with the same JSON shape as before the refactor (`id`, `email`, `display_name`, `email_verified`, `created_at`, `updated_at`).

Stop the server:
Run: `kill %1 || true`

Stop the database:
Run: `cd services/identity && docker compose down`

- [ ] **Step 8: Commit any final fixes**

If any smoke test revealed an issue, fix it and commit:
```bash
cd services/identity && git add -A
git commit -m "fix(identity): address smoke-test findings"
```

(Only run this if there were fixes. If the smoke test passed cleanly, skip this step.)

- [ ] **Step 9: Final commit summary**

Run: `cd services/identity && git log --oneline -10`
Expected: 8 commits (one per task) with messages matching the prefixes above, plus the spec commit from the brainstorming step.

---

## Self-Review Checklist

After writing this plan, verify against the spec:

- [ ] Spec goal "Domain types are free of transport and persistence concerns" → Tasks 2, 3, 4, 5, 6 all move `json` tags off domain types.
- [ ] Spec goal "A single `User` type exists" → Task 2 creates `identityuser.User`; Task 3 removes `user.User`; Task 4 removes `auth.User` and uses `UserCredentials` (which composes `identityuser.User`).
- [ ] Spec goal "Repository interfaces return row types" → Tasks 3, 4, 5, 6 all define `*XxxRow` types and update interfaces.
- [ ] Spec goal "Transactions can be added" → Task 1 introduces `Querier` + `TxRunner`; Task 4 stores `txr` on `AuthService` (unused for now).
- [ ] Spec goal "Service layer has zero knowledge of pgx" → Task 4 sub-step "Check for accidental pgx imports" enforces this; `service.go` files don't import pgx.
- [ ] Spec goal "Modules can be split" → The `identityuser` package is its own package with no upstream imports. Documented in the spec's "Splitting the service" future-work note.
- [ ] No placeholders: every step has full code or a clear action with verification.
- [ ] Type names consistent across tasks: `UserRow` (user), `UserCredentialsRow` (auth), `SessionRow` (session), `SessionContextRow` (auth session), `RoleRow` (role).
- [ ] Conversion methods consistent: `row.toDomain()` everywhere.
- [ ] `pgxTxRunner.RunInTx` is the only place that starts a tx and calls `WithQuerier`.
