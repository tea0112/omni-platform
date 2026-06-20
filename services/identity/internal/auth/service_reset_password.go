package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	userID, expiresAt, usedAt, err := s.userRepo.GetPasswordResetToken(ctx, token)
	if err != nil {
		return shared.ErrNotFound
	}
	if usedAt != nil || time.Now().After(expiresAt) {
		return shared.ErrTokenExpired
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return s.userRepo.MarkPasswordResetTokenUsed(ctx, token)
}
