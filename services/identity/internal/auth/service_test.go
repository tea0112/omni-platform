package auth_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/tea0112/omni-platform/services/identity/internal/auth"
	"github.com/tea0112/omni-platform/services/identity/internal/auth/mocks"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type mockEmailSender struct{}

func (m *mockEmailSender) SendPasswordReset(_ context.Context, _, _ string) error {
	return nil
}

func setupMocks(t *testing.T) (*mocks.MockUserRepository, *mocks.MockSessionRepository, *shared.PasswordHasher, *auth.AuthService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)
	hasher := shared.NewPasswordHasher(4)
	svc := auth.NewAuthService(userRepo, sessionRepo, hasher, nil, shared.NewRBAC(), &mockEmailSender{})
	return userRepo, sessionRepo, hasher, svc
}

func setupMocksWithToken(t *testing.T) (*mocks.MockUserRepository, *mocks.MockSessionRepository, *shared.PasswordHasher, *auth.AuthService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)
	hasher := shared.NewPasswordHasher(4)
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	tokenSvc := shared.NewTokenService(priv, pub, 15*time.Minute)
	svc := auth.NewAuthService(userRepo, sessionRepo, hasher, tokenSvc, shared.NewRBAC(), &mockEmailSender{})
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

func TestAuthService_ForgotPassword_Success(t *testing.T) {
	userRepo, _, _, svc := setupMocks(t)
	userID := uuid.Must(uuid.NewV7())

	userRepo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(&auth.User{ID: userID, Email: "test@example.com"}, nil)
	userRepo.EXPECT().CreatePasswordResetToken(gomock.Any(), userID, gomock.Any(), gomock.Any()).Return(nil)

	err := svc.ForgotPassword(context.Background(), "test@example.com")
	assert.NoError(t, err)
}

func TestAuthService_ForgotPassword_NoLeak(t *testing.T) {
	userRepo, _, _, svc := setupMocks(t)

	userRepo.EXPECT().GetByEmail(gomock.Any(), "missing@test.com").Return(nil, shared.ErrNotFound)

	err := svc.ForgotPassword(context.Background(), "missing@test.com")
	assert.NoError(t, err)
}

func TestAuthService_ResetPassword_Success(t *testing.T) {
	userRepo, _, _, svc := setupMocks(t)
	userID := uuid.Must(uuid.NewV7())
	token := "valid-token"
	expiresAt := time.Now().Add(1 * time.Hour)
	newPassword := "new-password-123"

	userRepo.EXPECT().GetPasswordResetToken(gomock.Any(), token).Return(userID, expiresAt, nil, nil)
	userRepo.EXPECT().UpdatePassword(gomock.Any(), userID, gomock.Any()).Return(nil)
	userRepo.EXPECT().MarkPasswordResetTokenUsed(gomock.Any(), token).Return(nil)

	err := svc.ResetPassword(context.Background(), token, newPassword)
	assert.NoError(t, err)
}

func TestAuthService_ResetPassword_TokenExpired(t *testing.T) {
	userRepo, _, _, svc := setupMocks(t)
	expiredAt := time.Now().Add(-1 * time.Hour)

	userRepo.EXPECT().GetPasswordResetToken(gomock.Any(), "expired-token").Return(uuid.UUID{}, expiredAt, nil, nil)

	err := svc.ResetPassword(context.Background(), "expired-token", "new-pass")
	assert.ErrorIs(t, err, shared.ErrTokenExpired)
}

func TestAuthService_ResetPassword_TokenAlreadyUsed(t *testing.T) {
	userRepo, _, _, svc := setupMocks(t)
	now := time.Now()
	usedAt := &now

	userRepo.EXPECT().GetPasswordResetToken(gomock.Any(), "used-token").Return(uuid.Must(uuid.NewV7()), time.Now().Add(1*time.Hour), usedAt, nil)

	err := svc.ResetPassword(context.Background(), "used-token", "new-pass")
	assert.ErrorIs(t, err, shared.ErrTokenExpired)
}

func TestAuthService_ResetPassword_TokenNotFound(t *testing.T) {
	userRepo, _, _, svc := setupMocks(t)

	userRepo.EXPECT().GetPasswordResetToken(gomock.Any(), "invalid-token").Return(uuid.UUID{}, time.Time{}, nil, shared.ErrNotFound)

	err := svc.ResetPassword(context.Background(), "invalid-token", "new-pass")
	assert.ErrorIs(t, err, shared.ErrNotFound)
}

func TestAuthService_Login_Success(t *testing.T) {
	userRepo, sessionRepo, hasher, svc := setupMocksWithToken(t)
	userID := uuid.Must(uuid.NewV7())
	password := "password123"
	hash, _ := hasher.Hash(password)

	userRepo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(&auth.User{ID: userID, Email: "test@example.com", PasswordHash: hash}, nil)
	userRepo.EXPECT().GetUserRolesAndPermissions(gomock.Any(), userID).Return([]string{"user"}, []string{"profile.read", "profile.write"}, nil)
	sessionRepo.EXPECT().CreateSession(gomock.Any(), userID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&auth.Session{}, nil)

	result, err := svc.Login(context.Background(), "test@example.com", password, "127.0.0.1", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	userRepo, _, hasher, svc := setupMocksWithToken(t)
	userID := uuid.Must(uuid.NewV7())
	hash, _ := hasher.Hash("correct-password")

	userRepo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(&auth.User{ID: userID, Email: "test@example.com", PasswordHash: hash}, nil)

	_, err := svc.Login(context.Background(), "test@example.com", "wrong-password", "127.0.0.1", nil)
	assert.ErrorIs(t, err, shared.ErrUnauthenticated)
}

func TestAuthService_Login_ValidationError(t *testing.T) {
	_, _, _, svc := setupMocksWithToken(t)

	_, err := svc.Login(context.Background(), "", "", "127.0.0.1", nil)
	assert.Error(t, err)
	var vErr *shared.ValidationError
	assert.ErrorAs(t, err, &vErr)
}

func TestAuthService_Refresh_Success(t *testing.T) {
	userRepo, sessionRepo, _, svc := setupMocksWithToken(t)
	userID := uuid.Must(uuid.NewV7())
	sessionID := uuid.Must(uuid.NewV7())
	refreshToken := "valid-refresh-token"

	sessionRepo.EXPECT().GetByRefreshToken(gomock.Any(), refreshToken).Return(&auth.Session{
		ID:           sessionID,
		UserID:       userID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}, nil)
	sessionRepo.EXPECT().Revoke(gomock.Any(), sessionID).Return(nil)
	userRepo.EXPECT().GetByID(gomock.Any(), userID).Return(&auth.User{ID: userID, Email: "test@example.com"}, nil)
	userRepo.EXPECT().GetUserRolesAndPermissions(gomock.Any(), userID).Return([]string{"user"}, []string{"profile.read", "profile.write"}, nil)
	sessionRepo.EXPECT().CreateSession(gomock.Any(), userID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&auth.Session{}, nil)

	result, err := svc.Refresh(context.Background(), refreshToken, "127.0.0.1", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
}

func TestAuthService_Refresh_Revoked(t *testing.T) {
	_, sessionRepo, _, svc := setupMocksWithToken(t)
	now := time.Now()
	refreshToken := "revoked-token"

	sessionRepo.EXPECT().GetByRefreshToken(gomock.Any(), refreshToken).Return(&auth.Session{
		RefreshToken: refreshToken,
		RevokedAt:    &now,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}, nil)
	sessionRepo.EXPECT().RevokeAllForUser(gomock.Any(), gomock.Any()).Return(nil)

	_, err := svc.Refresh(context.Background(), refreshToken, "127.0.0.1", nil)
	assert.ErrorIs(t, err, shared.ErrTokenRevoked)
}

func TestAuthService_Refresh_Expired(t *testing.T) {
	_, sessionRepo, _, svc := setupMocksWithToken(t)
	refreshToken := "expired-token"

	sessionRepo.EXPECT().GetByRefreshToken(gomock.Any(), refreshToken).Return(&auth.Session{
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	}, nil)

	_, err := svc.Refresh(context.Background(), refreshToken, "127.0.0.1", nil)
	assert.ErrorIs(t, err, shared.ErrTokenExpired)
}
