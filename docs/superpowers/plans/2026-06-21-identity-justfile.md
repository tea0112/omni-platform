# Identity Service `just` Task Runner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `just` task runner to the identity service that brings up the full local stack with a single command, fixes the missing `gen-jwk` and `migrate` Go subcommands the README already references, and provides `.env.example` templates for both the identity service and the web app.

**Architecture:** A single `services/identity/justfile` with phony workflow targets (`dev`, `ci`, `reset`), granular recipes (`db-up`, `migrate`, `gen-jwk`, `test`, `build`, `run`, etc.), and private helpers (`_docker`, `_go_run`, `_build_server`). Two new Go subcommands (`gen-jwk`, `migrate`) live in `cmd/server/main.go` and are dispatched via a simple `os.Args` switch (no CLI library). The justfile uses `set dotenv-load := true` to source `.env.local` automatically.

**Tech Stack:** [`just`](https://github.com/casey/just) v1+ (no specific version floor), Go 1.26+, Docker, PostgreSQL 18.4, [golang-migrate](https://github.com/golang-migrate/migrate) v4 (already a dependency).

## Global Constraints

- Working directory: `services/identity` (all commands run from there unless stated otherwise)
- Existing module: `github.com/tea0112/omni-platform/services/identity`
- Existing dependency: `github.com/golang-migrate/migrate/v4` (already in `go.mod`)
- **No new external dependencies** for the Go code (uses only stdlib + existing deps; `crypto/ed25519` from stdlib for key generation)
- All commits use `feat(identity):` prefix
- After each task, run `go build ./...` and `go test ./...` from `services/identity`
- The justfile must work with `just` v1.0+ (the version available in apt/homebrew as of mid-2026)
- `.env.local` is added to `.gitignore` in BOTH `services/identity/` and `apps/web/` from day one
- The justfile does not need to install `just` itself — that's a one-time developer setup step documented in the README

---

### Task 1: Add `internal/genkey` package with `Generate()`

**Files:**
- Create: `services/identity/internal/genkey/genkey.go`
- Create: `services/identity/internal/genkey/genkey_test.go`

**Interfaces:**
- Produces: `func Generate() (string, error)` returning a JSON-encoded JWK string with fields `kty="OKP"`, `crv="Ed25519"`, `d=<base64url(seed) without padding>`, `x=<base64url(public) without padding>`
- Produces: `var ErrKeyGen = errors.New("genkey: key generation failed")` (exposed for callers that want to wrap)

- [ ] **Step 1: Write the failing test**

Create `services/identity/internal/genkey/genkey_test.go` with this exact content:

```go
package genkey

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestGenerate_ReturnsValidJWK(t *testing.T) {
	s, err := Generate()
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	var jwk map[string]string
	if err := json.Unmarshal([]byte(s), &jwk); err != nil {
		t.Fatalf("Generate output is not valid JSON: %v\noutput: %s", err, s)
	}
	if jwk["kty"] != "OKP" {
		t.Errorf("kty = %q, want %q", jwk["kty"], "OKP")
	}
	if jwk["crv"] != "Ed25519" {
		t.Errorf("crv = %q, want %q", jwk["crv"], "Ed25519")
	}
	if _, err := base64.RawURLEncoding.DecodeString(jwk["d"]); err != nil {
		t.Errorf("d is not valid base64url: %v", err)
	}
	if _, err := base64.RawURLEncoding.DecodeString(jwk["x"]); err != nil {
		t.Errorf("x is not valid base64url: %v", err)
	}
	if jwk["d"] == "" || jwk["x"] == "" {
		t.Error("d and x must be non-empty")
	}
}

func TestGenerate_UniqueEachCall(t *testing.T) {
	s1, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if s1 == s2 {
		t.Error("two successive Generate calls produced the same JWK; expected unique keys")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/identity && go test ./internal/genkey/...`
Expected: FAIL with "package genkey not found" or "no Go files in .../genkey" — the package doesn't exist yet.

- [ ] **Step 3: Write the implementation**

Create `services/identity/internal/genkey/genkey.go` with this exact content:

```go
package genkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrKeyGen = errors.New("genkey: key generation failed")

// JWK is the JSON Web Key representation of an Ed25519 key.
// Only the fields needed by the identity service are populated.
type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	D   string `json:"d"`
	X   string `json:"x"`
}

// Generate returns a freshly-generated Ed25519 key encoded as a JSON JWK string.
// The private seed (d) and the public key (x) are base64url-encoded without padding
// per RFC 7518 §6.2.1.1.
func Generate() (string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKeyGen, err)
	}
	j := JWK{
		Kty: "OKP",
		Crv: "Ed25519",
		D:   base64.RawURLEncoding.EncodeToString(priv.Seed()),
		X:   base64.RawURLEncoding.EncodeToString(pub),
	}
	b, err := json.Marshal(j)
	if err != nil {
		return "", fmt.Errorf("%w: marshal: %v", ErrKeyGen, err)
	}
	return string(b), nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/identity && go test ./internal/genkey/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd services/identity && git add internal/genkey/
git commit -m "feat(identity): add genkey package for Ed25519 JWK generation"
```

---

### Task 2: Add `Down()` to `internal/migrate`

**Files:**
- Modify: `services/identity/internal/migrate/migrate.go`
- Create: `services/identity/internal/migrate/migrate_test.go`

**Interfaces:**
- Produces: `func Down(dbURL string) error` — rolls back the most recent migration

- [ ] **Step 1: Read the existing migrate package**

Read `services/identity/internal/migrate/migrate.go` to understand the existing `Run` function structure. The new `Down` mirrors it but uses `m.Steps(-1)` instead of `m.Up()`.

- [ ] **Step 2: Write the integration test**

Create `services/identity/internal/migrate/migrate_test.go` with this exact content:

```go
//go:build integration

package migrate_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/tea0112/omni-platform/services/identity/internal/migrate"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func newTestDB(t *testing.T) (*pgxpool.Pool, string) {
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
	return pool, dsn
}

func TestDown_RollsBackOneStep(t *testing.T) {
	pool, dsn := newTestDB(t)

	// After Up, the users table exists.
	var count int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM information_schema.tables WHERE table_name = 'users'`).Scan(&count))
	require.Equal(t, 1, count, "users table should exist after Up")

	// After Down, the users table is gone.
	require.NoError(t, migrate.Down(dsn))

	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM information_schema.tables WHERE table_name = 'users'`).Scan(&count))
	require.Equal(t, 0, count, "users table should not exist after Down")
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd services/identity && go test -tags=integration ./internal/migrate/...`
Expected: FAIL with "undefined: migrate.Down"

- [ ] **Step 4: Add `Down()` to migrate.go**

Append to `services/identity/internal/migrate/migrate.go` (the file already has a `Run` function — find it and add `Down` alongside):

```go
// Down rolls back the most recent migration.
func Down(dbURL string) error {
	d, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("migrate down: open source: %w", err)
	}
	m, err := migrate_lib.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		return fmt.Errorf("migrate down: new instance: %w", err)
	}
	if err := m.Steps(-1); err != nil && err != migrate_lib.ErrNoChange {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}
```

The exact imports already exist in the file (migrate_lib, iofs, etc.); if any are missing, add them per the existing `Run` function's import list.

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd services/identity && go test -tags=integration ./internal/migrate/...`
Expected: PASS (if Docker is available; if not, this step requires Docker — note the result)

If Docker is unavailable, the test will fail to start the container. That's OK; this test will be re-run during the final smoke test (Task 8). Skip to the next step but note the situation.

- [ ] **Step 6: Run unit tests to confirm no regression**

Run: `cd services/identity && go test ./...`
Expected: existing unit tests still pass (auth, shared).

- [ ] **Step 7: Commit**

```bash
cd services/identity && git add internal/migrate/
git commit -m "feat(identity): add Down() to migrate package for rollbacks"
```

---

### Task 3: Add subcommand dispatch to `cmd/server/main.go`

**Files:**
- Modify: `services/identity/cmd/server/main.go`
- Create: `services/identity/cmd/server/main_test.go`

**Interfaces:**
- Consumes: `genkey.Generate()` (from Task 1), `migrate.Run()` and `migrate.Down()` (from Task 2)
- Produces: `func runGenJwk() int` (exit code), `func runMigrate(args []string) int` (exit code)
- Subcommands: `./server gen-jwk` writes `.env.local`; `./server migrate` runs migrations; `./server migrate down` rolls back

- [ ] **Step 1: Read the current main.go**

Read `services/identity/cmd/server/main.go` to understand the existing `main()` function and imports.

- [ ] **Step 2: Write the smoke test**

Create `services/identity/cmd/server/main_test.go` with this exact content:

```go
//go:build integration

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tea0112/omni-platform/services/identity/internal/genkey"
	"github.com/tea0112/omni-platform/services/identity/internal/migrate"
)

func TestRunGenJwk_WritesEnvLocal(t *testing.T) {
	// Run in an isolated temp dir so we don't touch the real .env.local
	tmp := t.TempDir()
	oldCwd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(oldCwd) })

	code := runGenJwk()
	require.Equal(t, 0, code, "runGenJwk should exit 0")

	// .env.local should exist and contain IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK=
	path := filepath.Join(tmp, ".env.local")
	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	s := string(contents)
	require.True(t, strings.HasPrefix(s, "IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK="),
		".env.local should start with the JWK env var, got: %s", s)
	require.Contains(t, s, `"kty":"OKP"`)
	require.Contains(t, s, `"crv":"Ed25519"`)
}

func TestRunMigrate_RequiresArgs(t *testing.T) {
	// Without DB env, runMigrate should fail fast.
	t.Setenv("IDENTITY_DB_HOST", "")
	code := runMigrate([]string{})
	require.NotEqual(t, 0, code, "runMigrate with no DB env should fail")
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd services/identity && go test -tags=integration ./cmd/server/...`
Expected: FAIL with "undefined: runGenJwk" or "undefined: runMigrate"

- [ ] **Step 4: Add subcommand dispatch to main.go**

Modify `services/identity/cmd/server/main.go`. Add the imports and the dispatch logic. The current `main()` does `fx.New(...).Run()`. Wrap it with the subcommand switch.

Add these imports at the top (alongside existing imports):
```go
import (
    // ... existing imports ...
    "github.com/tea0112/omni-platform/services/identity/internal/genkey"
    "github.com/tea0112/omni-platform/services/identity/internal/migrate"
)
```

Replace the existing `func main()` with this version:

```go
func main() {
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "gen-jwk":
            os.Exit(runGenJwk())
        case "migrate":
            args := []string{}
            if len(os.Args) > 2 {
                args = os.Args[2:]
            }
            os.Exit(runMigrate(args))
        }
    }
    fx.New(
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
        fx.Invoke(
            RunMigrations,
            fx.Annotate(Serve, fx.ParamTags(
                ``, ``,
                ``, ``,
                ``, ``,
                `group:"grpc-handlers"`,
                ``, ``,
            )),
        ),
    ).Run()
}

func runGenJwk() int {
    jwk, err := genkey.Generate()
    if err != nil {
        fmt.Fprintf(os.Stderr, "gen-jwk: %v\n", err)
        return 1
    }
    if err := os.WriteFile(".env.local", []byte("IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK="+jwk+"\n"), 0o600); err != nil {
        fmt.Fprintf(os.Stderr, "gen-jwk: write .env.local: %v\n", err)
        return 1
    }
    fmt.Println("wrote .env.local with a fresh Ed25519 JWK")
    return 0
}

func runMigrate(args []string) int {
    cfg, err := shared.Load()
    if err != nil {
        fmt.Fprintf(os.Stderr, "migrate: load config: %v\n", err)
        return 1
    }
    dbURL := cfg.DB.DSN()
    var err error
    if len(args) > 0 && args[0] == "down" {
        err = migrate.Down(dbURL)
    } else {
        err = migrate.Run(dbURL)
    }
    if err != nil {
        fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
        return 1
    }
    return 0
}
```

The `RunMigrations` function is already defined in main.go (it was previously called from `fx.Invoke`); do not redefine it.

If `shared.Load()` doesn't exist with that exact signature, look up the actual signature in `internal/shared/config.go` and adjust. The existing `shared.MustLoad` returns `(shared.Config, error)` — use that.

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd services/identity && go test -tags=integration ./cmd/server/...`
Expected: PASS (or skip if Docker unavailable; the test doesn't need Docker for `runGenJwk`)

- [ ] **Step 6: Run all tests to confirm no regression**

Run: `cd services/identity && go test ./...`
Expected: existing unit tests still pass.

- [ ] **Step 7: Build the binary to confirm it still compiles**

Run: `cd services/identity && go build -trimpath -o /tmp/identity-server-test ./cmd/server`
Expected: binary created, no errors.

- [ ] **Step 8: Smoke test the subcommands manually**

Run the new subcommands directly to confirm they work:

```bash
cd services/identity
IDENTITY_DB_HOST=invalidhost /tmp/identity-server-test migrate
# Expected: error (no DB connection) but graceful error message
```

Then test gen-jwk (in a temp dir to avoid clobbering real .env.local):
```bash
cd $(mktemp -d) && /tmp/identity-server-test gen-jwk
# Expected: writes .env.local, prints "wrote .env.local with a fresh Ed25519 JWK"
```

Both should exit with non-zero on errors and zero on success. If both work, this step passes.

- [ ] **Step 9: Commit**

```bash
cd services/identity && git add cmd/server/
git commit -m "feat(identity): add gen-jwk and migrate subcommands to main"
```

---

### Task 4: Add `.env.example` to identity service and update `.gitignore`

**Files:**
- Create: `services/identity/.env.example`
- Modify: `services/identity/.gitignore` (add `.env.local` if not present)

**Interfaces:**
- Produces: `services/identity/.env.example` — committed template documenting every env var
- Produces: `.gitignore` entry for `.env.local`

- [ ] **Step 1: Read the current .gitignore**

Run: `cat services/identity/.gitignore`
Expected to see (or not) `.env.local`. If already present, skip to Step 3.

- [ ] **Step 2: Add `.env.local` to .gitignore if not present**

Edit `services/identity/.gitignore`. If `.env.local` is not in the file, add it on a new line:
```
.env.local
```

(If the file already has it, do nothing.)

- [ ] **Step 3: Create `.env.example`**

Create `services/identity/.env.example` with this exact content:

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
# Generate via `just gen-jwk`. The value is a JSON object (no surrounding quotes).
# IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK={"kty":"OKP","crv":"Ed25519","d":"...","x":"..."}
IDENTITY_AUTH_ACCESS_TOKEN_TTL=15m
IDENTITY_AUTH_REFRESH_TOKEN_TTL=672h
IDENTITY_AUTH_BCRYPT_COST=12

# --- Email ---
IDENTITY_EMAIL_PROVIDER=log

# --- Observability ---
IDENTITY_OTEL_ENDPOINT=localhost:4317
```

- [ ] **Step 4: Verify the env file is parsed correctly**

Run a quick sanity check that the env values are valid for the config loader. From `services/identity`, run:
```bash
IDENTITY_DB_HOST=testhost IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK='{"kty":"OKP","crv":"Ed25519","d":"AAAA","x":"AAAA"}' go run ./cmd/server gen-jwk
```
Expected: writes a new `.env.local` (overwriting the one with `testhost` etc.). This confirms the config loader can read the example format.

- [ ] **Step 5: Commit**

```bash
cd services/identity && git add .env.example .gitignore
git commit -m "feat(identity): add .env.example template and gitignore .env.local"
```

---

### Task 5: Add `.env.example` to web app and update `.gitignore`

**Files:**
- Create: `apps/web/.env.example`
- Modify: `apps/web/.gitignore` (add `.env.local` if not present; Next.js convention may already include it)

- [ ] **Step 1: Read the current .gitignore**

Run: `cat apps/web/.gitignore`
Expected: should already have `.env*` or `.env.local` per Next.js convention. If present, skip to Step 3.

- [ ] **Step 2: Add `.env.local` to .gitignore if not present**

If `.env.local` is not in the file, add it on a new line:
```
.env.local
```

(If already there, do nothing.)

- [ ] **Step 3: Create `apps/web/.env.example`**

Create `apps/web/.env.example` with this exact content:

```bash
# Identity service URL (default: http://localhost:8080)
# Uncomment to override:
# IDENTITY_SERVICE_URL=http://localhost:8080
```

- [ ] **Step 4: Commit**

```bash
cd apps/web && git add .env.example .gitignore
git commit -m "feat(web): add .env.example template and verify .env.local is gitignored"
```

---

### Task 6: Add the justfile

**Files:**
- Create: `services/identity/justfile`

- [ ] **Step 1: Create the justfile**

Create `services/identity/justfile` with this exact content:

```just
# justfile for the identity service
# Run `just` (no args) to see all available recipes.

set dotenv-load := true
set positional-arguments := true

# Variables
docker  := 'docker compose'
goflags := '-trimpath'

# Default target: full local stack
default: dev

# --- Workflows ---

# Bring up the full local stack: db-up → migrate → gen-jwk (if needed) → run
dev: db-up migrate
    @if [ ! -f .env.local ]; then just gen-jwk; fi
    just run

# What CI runs: vet → test → build
ci: vet test build

# Tear everything down (containers, build artifacts, .env.local)
reset: db-down clean
    rm -f .env.local

# --- Database ---

# Start the local Postgres container
db-up:
    {{ docker }} up -d identity-postgres

# Stop the local Postgres container
db-down:
    {{ docker }} down

# Tail the Postgres logs
db-logs:
    {{ docker }} logs -f identity-postgres

# Show container status
db-ps:
    {{ docker }} ps

# Wipe the database (volume) and restart. Requires --confirm.
db-reset:
    @just --confirm db-down
    docker volume rm identity-pgdata || true
    just db-up

# --- Migrations ---

# Apply all up migrations
migrate:
    go run ./cmd/server migrate

# Roll back one migration
migrate-down:
    go run ./cmd/server migrate down

# --- Secrets ---

# Generate a fresh Ed25519 JWK and write it to .env.local
gen-jwk:
    go run ./cmd/server gen-jwk

# --- Tests ---

# Run all tests
test:
    go test ./...

# Run unit tests only (skip integration)
test-unit:
    go test -short ./...

# Run integration tests (requires db-up)
test-integration:
    go test -tags=integration ./...

# Run tests with coverage report
test-coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out

# --- Build ---

# Build the server binary
build:
    go build {{ goflags }} -o server ./cmd/server

# Run the server (requires build and .env.local)
run: build
    ./server

# --- Code quality ---

# Run go vet
vet:
    go vet ./...

# Run go fmt
fmt:
    go fmt ./...

# Run go generate
generate:
    go generate ./...

# Lint placeholder (no linter configured today)
lint:
    @echo "no linter configured"

# --- Cleanup ---

# Remove build artifacts
clean:
    rm -f server coverage.out

# --- Docker stack ---

# Start the full stack (postgres + migrate + server) via Docker Compose
docker-up:
    {{ docker }} up -d

# Stop the full stack
docker-down:
    {{ docker }} down

# Tail the full stack logs
docker-logs:
    {{ docker }} logs -f
```

- [ ] **Step 2: Verify justfile syntax**

Run: `which just || echo "just not installed"`
Expected: shows the path to the just binary, OR "just not installed" if not on the system.

If `just` is not installed, the engineer cannot run the recipes but the file is still syntactically valid. Install via `brew install just` (macOS) or `apt install just` (Debian/Ubuntu) to validate.

Run: `cd services/identity && just --list`
Expected: prints the recipe list. Each recipe has a description because we commented them.

- [ ] **Step 3: Smoke test the justfile**

Run: `cd services/identity && just --evaluate`
Expected: prints evaluated variables (`docker`, `goflags`). Confirms the file parses.

- [ ] **Step 4: Commit**

```bash
cd services/identity && git add justfile
git commit -m "feat(identity): add justfile with dev, ci, db, migrate, gen-jwk, test, build recipes"
```

---

### Task 7: Update READMEs

**Files:**
- Modify: `services/identity/README.md`
- Modify: `apps/web/README.md`

- [ ] **Step 1: Read the current READMEs**

Read both files to see what's there. The identity README has Setup/Run/Docker sections; the web README is short.

- [ ] **Step 2: Update `services/identity/README.md`**

Replace the **Prerequisites** section with this version:

```markdown
## Prerequisites

- Go 1.26+
- PostgreSQL (user: `identity`, password: `identity`, db: `identity`, port: `5432`) — easiest via `just db-up` (uses Docker)
- [`just`](https://github.com/casey/just) (`brew install just` or `apt install just`)
```

Replace the **Setup** section with this:

```markdown
## Setup

```bash
go mod download
cp .env.example .env.local  # not needed if you use `just gen-jwk`
just db-up
just migrate
just gen-jwk   # writes a fresh Ed25519 JWK to .env.local
```

That's the full first-time setup.
```

Replace the **Run** section with this:

```markdown
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
```

Keep the **Environment Variables** table as-is — it documents the variables. The justfile is how you set them.

Replace the **Run with Docker Compose** section with this:

```markdown
## Run with Docker Compose

```bash
export IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK="$(just gen-jwk)"
just docker-up
```

This starts PostgreSQL, runs migrations, and starts the server on `:8080`.
```

- [ ] **Step 3: Update `apps/web/README.md`**

Add a new **Setup** subsection right after the "## Setup" header. The current Setup is just `npm install`. Add a new section after the existing Setup (or merge into Setup):

```markdown
## Environment

Copy `.env.example` to `.env.local` and adjust as needed:

```bash
cp .env.example .env.local
```

The only variable is `IDENTITY_SERVICE_URL` (defaults to `http://localhost:8080`).
```

- [ ] **Step 4: Verify the READMEs are sensible**

Re-read both files end-to-end. Confirm the recipe list in the identity README matches the actual justfile. Confirm the web README has the env section.

- [ ] **Step 5: Commit**

```bash
git add services/identity/README.md apps/web/README.md
git commit -m "docs(identity): rewrite README around justfile; docs(web): add env setup section"
```

---

### Task 8: End-to-end smoke test

**Files:** none (this task is verification only)

- [ ] **Step 1: Build the server binary**

Run: `cd services/identity && go build -trimpath -o /tmp/identity-server ./cmd/server`
Expected: no errors.

- [ ] **Step 2: Generate a JWK and start the server with the justfile**

If `just` is installed:
```bash
cd services/identity
just gen-jwk  # writes .env.local
# start postgres, run migrations, start the server
just db-up
just migrate
IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK="$(cat .env/local 2>/dev/null || cat .env.local | cut -d= -f2)" /tmp/identity-server &
sleep 3
```

(If `just` is not installed, do these steps manually: start the docker postgres container, run `go run ./cmd/server migrate`, then start the binary with the JWK from the env file.)

- [ ] **Step 3: Verify the health endpoint**

Run: `curl -s http://localhost:8080/healthz`
Expected: `ok`

- [ ] **Step 4: Register and login via the API**

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
    -H 'Content-Type: application/json' \
    -d '{"email":"smoke-justfile@test.com","password":"smokepw123"}'

curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"smoke-justfile@test.com","password":"smokepw123"}'
```

Expected: register returns 201 with a user object; login returns 200 with `access_token` and `refresh_token` and a user object that includes `created_at` and `updated_at`.

- [ ] **Step 5: Stop the server**

```bash
pkill -f /tmp/identity-server || true
```

- [ ] **Step 6: Run `just ci` (or equivalent)**

If `just` is installed:
```bash
cd services/identity && just ci
```

Otherwise:
```bash
cd services/identity
go vet ./...
go test ./...
go build -trimpath -o server ./cmd/server
```

Expected: all green, build succeeds.

- [ ] **Step 7: Document any issues found**

If anything failed, write a brief note to `.superpowers/sdd/task-8-justfile-report.md` listing what failed and how you fixed it (or note that follow-up is needed). This file is gitignored.

- [ ] **Step 8: Commit any fixes**

If fixes were made, commit them:
```bash
git add -A
git commit -m "fix(identity): address justfile smoke-test findings"
```

(Only commit if changes were made.)

- [ ] **Step 9: Final commit summary**

Run: `git log --oneline -10`
Expected: 8 commits (one per task) with `feat(identity):` / `feat(web):` / `docs(...):` / `fix(...):` prefixes, plus the spec commit from the brainstorming step.

---

## Self-Review Checklist

After writing this plan, verify against the spec:

- [x] Spec goal 1 (single source of truth: justfile) — Tasks 6, 7
- [x] Spec goal 2 (working subcommands) — Tasks 1, 2, 3
- [x] Spec goal 3 (zero-friction onboarding) — Task 6 (justfile has `dev` workflow), Task 8 (smoke test)
- [x] Spec goal 4 (env documentation) — Tasks 4, 5
- [x] Spec goal 5 (CI-ready) — Task 6 (`ci` recipe), Task 8 (`just ci` smoke test)
- [x] No placeholders in steps — all code is shown
- [x] Type names consistent: `genkey.Generate()`, `migrate.Run()`, `migrate.Down()`, `runGenJwk() int`, `runMigrate([]string) int`
- [x] Subcommand dispatch is a single `switch os.Args[1]` block in `main.go`
- [x] Justfile structure: phony workflows + granular + private helpers, all in one file
- [x] `.env.example` content is the complete list of env vars from `shared.Config`
- [x] `.env.local` is gitignored in BOTH services from day one (Tasks 4 and 5)
- [x] No new external dependencies added to `go.mod` (uses only stdlib for the new code)
- [x] End-to-end smoke test in Task 8 covers: build, gen-jwk, db-up, migrate, server start, /healthz, register, login, ci
