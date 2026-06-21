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
