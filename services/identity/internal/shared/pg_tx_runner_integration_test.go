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
