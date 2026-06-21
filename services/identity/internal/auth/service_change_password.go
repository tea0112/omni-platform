package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if currentPassword == "" || newPassword == "" {
		return &shared.ValidationError{Fields: map[string]string{
			"current_password": "required",
			"new_password":     "required",
		}}
	}

	if len(newPassword) < 8 {
		return &shared.ValidationError{Fields: map[string]string{
			"new_password": "must be at least 8 characters",
		}}
	}

	row, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	creds := row.toDomain()

	if err := s.hasher.Compare(creds.PasswordHash(), currentPassword); err != nil {
		return shared.ErrUnauthenticated
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}
