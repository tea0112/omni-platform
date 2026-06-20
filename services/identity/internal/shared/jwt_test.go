package shared_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func TestJWT_Roundtrip(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	svc := shared.NewTokenService(priv, pub, 15*time.Minute)

	token, expiresAt, err := svc.GenerateAccessToken("user-123", []string{"admin"}, []string{"users.read", "users.write"})
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))

	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.Subject)
	assert.Equal(t, []string{"admin"}, claims.Roles)
	assert.Equal(t, []string{"users.read", "users.write"}, claims.Permissions)
}

func TestJWT_Expired(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	svc := shared.NewTokenService(priv, pub, -1*time.Second)
	token, _, _ := svc.GenerateAccessToken("user-123", nil, nil)
	_, err := svc.ValidateAccessToken(token)
	assert.ErrorIs(t, err, shared.ErrTokenExpired)
}
