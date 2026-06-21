//go:build integration

package auth_test

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tea0112/omni-platform/services/identity/internal/auth"
	"github.com/tea0112/omni-platform/services/identity/internal/migrate"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func TestAuthRepository_CreateAndGetByEmail(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:18.4-alpine",
		postgres.WithDatabase("identity"),
		postgres.WithUsername("identity"),
		postgres.WithPassword("identity"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { container.Terminate(ctx) })

	dsn, _ := container.ConnectionString(ctx, "sslmode=disable")
	pool, err := shared.NewDBPool(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, migrate.Run(dsn))

	repo := auth.NewAuthUserPGRepository(pool)

	user, err := repo.Create(ctx, "test@example.com", "hashed-password")
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)

	found, err := repo.GetByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
}
