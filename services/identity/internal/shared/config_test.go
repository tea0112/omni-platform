package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func TestMustLoad_Defaults(t *testing.T) {
	t.Setenv("IDENTITY_DB_HOST", "localhost")
	t.Setenv("IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK", `{"kty":"OKP","crv":"Ed25519","x":"oCBAv07QPyAbbX-NLVPIeMX6e2tlJonlyw8Cb0Ufz3w","d":"nWGxne_9WmC6hEr0kuwsxERJxWl7MmkZcDusAxzOf2A"}`)

	cfg := shared.MustLoad()

	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.NotNil(t, cfg.Auth.JWTPrivateKey)
	assert.NotNil(t, cfg.Auth.JWTPublicKey)
	assert.Equal(t, "log", cfg.Email.Provider)
}

func TestMustLoad_EnvOverride(t *testing.T) {
	t.Setenv("IDENTITY_DB_PORT", "6432")
	t.Setenv("IDENTITY_SERVER_PORT", "9090")
	t.Setenv("IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK", `{"kty":"OKP","crv":"Ed25519","x":"oCBAv07QPyAbbX-NLVPIeMX6e2tlJonlyw8Cb0Ufz3w","d":"nWGxne_9WmC6hEr0kuwsxERJxWl7MmkZcDusAxzOf2A"}`)

	cfg := shared.MustLoad()
	assert.Equal(t, 6432, cfg.DB.Port)
	assert.Equal(t, 9090, cfg.Server.Port)
}
