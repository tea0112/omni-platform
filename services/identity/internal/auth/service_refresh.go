package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (s *AuthService) Refresh(ctx context.Context, refreshToken, ipAddress string, deviceInfo map[string]any) (*AuthResult, error) {
	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, shared.ErrNotFound
	}

	if session.RevokedAt != nil {
		s.sessionRepo.RevokeAllForUser(ctx, session.UserID)
		return nil, shared.ErrTokenRevoked
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, shared.ErrTokenExpired
	}

	s.sessionRepo.Revoke(ctx, session.ID)

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	roles, perms, err := s.userRepo.GetUserRolesAndPermissions(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get roles and permissions: %w", err)
	}

	accessToken, expiresAt, err := s.tokenSvc.GenerateAccessToken(user.ID.String(), roles, perms)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	newRefreshToken := uuid.Must(uuid.NewV7()).String()
	_, err = s.sessionRepo.CreateSession(ctx, user.ID, newRefreshToken, deviceInfo, ipAddress, time.Now().Add(s.refreshTokenTTL))
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		User:         *user,
	}, nil
}
