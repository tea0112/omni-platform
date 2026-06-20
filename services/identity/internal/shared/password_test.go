package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func TestPasswordHasher_HashAndCompare(t *testing.T) {
	h := shared.NewPasswordHasher(4)
	hash, err := h.Hash("mypassword")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotContains(t, hash, "mypassword")
	err = h.Compare(hash, "mypassword")
	assert.NoError(t, err)
}

func TestPasswordHasher_WrongPassword(t *testing.T) {
	h := shared.NewPasswordHasher(4)
	hash, _ := h.Hash("correct")
	err := h.Compare(hash, "wrong")
	assert.Error(t, err)
}
