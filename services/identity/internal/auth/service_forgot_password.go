package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	row, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil // don't leak whether email exists
	}
	creds := row.toDomain()
	token := uuid.Must(uuid.NewV7()).String()
	expiresAt := time.Now().Add(1 * time.Hour)
	if err := s.userRepo.CreatePasswordResetToken(ctx, creds.User().ID, token, expiresAt); err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}
	return s.emailSender.SendPasswordReset(ctx, email, token)
}
