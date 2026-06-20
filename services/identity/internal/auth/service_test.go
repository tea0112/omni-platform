package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/tea0112/omni-platform/services/identity/internal/auth"
	"github.com/tea0112/omni-platform/services/identity/internal/auth/mocks"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func setupMocks(t *testing.T) (*mocks.MockUserRepository, *mocks.MockSessionRepository, *shared.PasswordHasher, *auth.AuthService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)
	hasher := shared.NewPasswordHasher(4)
	svc := auth.NewAuthService(userRepo, sessionRepo, hasher, nil, shared.NewRBAC(), nil, nil)
	return userRepo, sessionRepo, hasher, svc
}

func TestAuthService_Register_Success(t *testing.T) {
	userRepo, _, _, svc := setupMocks(t)

	userRepo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(nil, shared.ErrNotFound)
	userRepo.EXPECT().Create(gomock.Any(), "test@example.com", gomock.Any()).Return(&auth.User{ID: uuid.Must(uuid.NewV7())}, nil)

	user, err := svc.Register(context.Background(), "test@example.com", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
}

func TestAuthService_Register_Duplicate(t *testing.T) {
	userRepo, _, _, svc := setupMocks(t)
	userRepo.EXPECT().GetByEmail(gomock.Any(), "existing@test.com").Return(&auth.User{}, nil)

	_, err := svc.Register(context.Background(), "existing@test.com", "password")
	assert.ErrorIs(t, err, shared.ErrDuplicate)
}

func TestAuthService_Register_ValidationError(t *testing.T) {
	_, _, _, svc := setupMocks(t)

	_, err := svc.Register(context.Background(), "", "")
	assert.Error(t, err)
	var vErr *shared.ValidationError
	assert.ErrorAs(t, err, &vErr)
}

func TestAuthService_Logout_Success(t *testing.T) {
	_, sessionRepo, _, svc := setupMocks(t)
	userID := uuid.Must(uuid.NewV7())

	sessionRepo.EXPECT().RevokeAllForUser(gomock.Any(), userID).Return(nil)

	err := svc.Logout(context.Background(), userID)
	assert.NoError(t, err)
}
