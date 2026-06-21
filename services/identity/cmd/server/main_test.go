//go:build integration

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunGenJwk_WritesEnvLocal(t *testing.T) {
	tmp := t.TempDir()
	oldCwd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(oldCwd) })

	code := runGenJwk()
	require.Equal(t, 0, code, "runGenJwk should exit 0")

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
	t.Setenv("IDENTITY_DB_HOST", "")
	code := runMigrate([]string{})
	require.NotEqual(t, 0, code, "runMigrate with no DB env should fail")
}
