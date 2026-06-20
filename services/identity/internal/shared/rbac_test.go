package shared_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func TestRBAC_Can_HasPermission(t *testing.T) {
	rbac := shared.NewRBAC()
	ctx := shared.WithPrincipal(context.Background(), shared.Principal{
		UserID:      "u1",
		Roles:       []string{"admin"},
		Permissions: []string{"users.write", "users.read"},
	})

	err := rbac.Can(ctx, "users.write")
	assert.NoError(t, err)
}

func TestRBAC_Can_MissingPermission(t *testing.T) {
	rbac := shared.NewRBAC()
	ctx := shared.WithPrincipal(context.Background(), shared.Principal{
		UserID:      "u1",
		Roles:       []string{"user"},
		Permissions: []string{"users.read"},
	})

	err := rbac.Can(ctx, "users.write")
	assert.ErrorIs(t, err, shared.ErrForbidden)
}

func TestRBAC_Can_OwnResource(t *testing.T) {
	rbac := shared.NewRBAC()
	ctx := shared.WithPrincipal(context.Background(), shared.Principal{
		UserID:      "u1",
		Roles:       []string{"user"},
		Permissions: []string{"profile.write"},
	})

	err := rbac.Can(ctx, "users.write", "u1")
	assert.NoError(t, err)
}

func TestRBAC_Can_NoPrincipal(t *testing.T) {
	rbac := shared.NewRBAC()
	err := rbac.Can(context.Background(), "users.write")
	assert.ErrorIs(t, err, shared.ErrUnauthenticated)
}
