package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil // don't leak whether email exists
	}
	token := uuid.Must(uuid.NewV7()).String()
	expiresAt := time.Now().Add(1 * time.Hour)
	if err := s.userRepo.CreatePasswordResetToken(ctx, user.ID, token, expiresAt); err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}
	return s.emailSender.SendPasswordReset(ctx, email, token)
}
