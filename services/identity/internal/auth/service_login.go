package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (s *AuthService) Login(ctx context.Context, email, password, ipAddress string, deviceInfo map[string]any) (*AuthResult, error) {
	if email == "" || password == "" {
		return nil, &shared.ValidationError{Fields: map[string]string{
			"email":    "required",
			"password": "required",
		}}
	}

	row, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, shared.ErrUnauthenticated
	}
	creds := row.toDomain()

	if err := s.hasher.Compare(creds.PasswordHash(), password); err != nil {
		return nil, shared.ErrUnauthenticated
	}

	roles, perms, err := s.userRepo.GetUserRolesAndPermissions(ctx, creds.User().ID)
	if err != nil {
		return nil, fmt.Errorf("get roles and permissions: %w", err)
	}

	accessToken, expiresAt, err := s.tokenSvc.GenerateAccessToken(creds.User().ID.String(), roles, perms)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken := uuid.Must(uuid.NewV7()).String()
	_, err = s.sessionRepo.CreateSession(ctx, creds.User().ID, refreshToken, deviceInfo, ipAddress, time.Now().Add(s.refreshTokenTTL))
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         creds.User(),
	}, nil
}
