package auth_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/tea0112/omni-platform/services/identity/internal/auth"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func TestChangeEmail_Success(t *testing.T) {
	userRepo, _, hasher, svc := setupMocks(t)
	userID := uuid.Must(uuid.NewV7())
	currentHash, _ := hasher.Hash("password")

	userRepo.EXPECT().GetByID(gomock.Any(), userID).Return(&auth.UserCredentialsRow{
		ID: userID, Email: "old@test.com", PasswordHash: currentHash,
	}, nil)
	userRepo.EXPECT().UpdateEmail(gomock.Any(), userID, "new@test.com").Return(nil)

	user, err := svc.ChangeEmail(t.Context(), userID, "password", "new@test.com")
	require.NoError(t, err)
	assert.Equal(t, "new@test.com", user.Email)
}

func TestChangeEmail_WrongPassword(t *testing.T) {
	userRepo, _, hasher, svc := setupMocks(t)
	userID := uuid.Must(uuid.NewV7())
	currentHash, _ := hasher.Hash("correct")

	userRepo.EXPECT().GetByID(gomock.Any(), userID).Return(&auth.UserCredentialsRow{
		ID: userID, Email: "a@b.com", PasswordHash: currentHash,
	}, nil)

	_, err := svc.ChangeEmail(t.Context(), userID, "wrong", "new@test.com")
	assert.ErrorIs(t, err, shared.ErrUnauthenticated)
}
