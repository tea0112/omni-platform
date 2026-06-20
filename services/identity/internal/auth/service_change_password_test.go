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

func TestChangePassword_Success(t *testing.T) {
	userRepo, _, hasher, svc := setupMocks(t)
	userID := uuid.Must(uuid.NewV7())
	currentHash, _ := hasher.Hash("oldpassword")

	userRepo.EXPECT().GetByID(gomock.Any(), userID).Return(&auth.User{
		ID: userID, Email: "a@b.com", PasswordHash: currentHash,
	}, nil)
	userRepo.EXPECT().UpdatePassword(gomock.Any(), userID, gomock.Any()).Return(nil)

	err := svc.ChangePassword(t.Context(), userID, "oldpassword", "newpassword")
	require.NoError(t, err)
}

func TestChangePassword_WrongCurrent(t *testing.T) {
	userRepo, _, hasher, svc := setupMocks(t)
	userID := uuid.Must(uuid.NewV7())
	currentHash, _ := hasher.Hash("correct")

	userRepo.EXPECT().GetByID(gomock.Any(), userID).Return(&auth.User{
		ID: userID, Email: "a@b.com", PasswordHash: currentHash,
	}, nil)

	err := svc.ChangePassword(t.Context(), userID, "wrong", "newpassword")
	assert.ErrorIs(t, err, shared.ErrUnauthenticated)
}
