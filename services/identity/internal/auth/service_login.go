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

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, shared.ErrUnauthenticated
	}

	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, shared.ErrUnauthenticated
	}

	roles, perms := []string{"user"}, []string{"profile.read", "profile.write"}

	accessToken, expiresAt, err := s.tokenSvc.GenerateAccessToken(user.ID.String(), roles, perms)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken := uuid.Must(uuid.NewV7()).String()
	_, err = s.sessionRepo.CreateSession(ctx, user.ID, refreshToken, deviceInfo, ipAddress, time.Now().Add(30*24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         *user,
	}, nil
}
