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
	_, err = s.pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.Must(uuid.NewV7()), user.ID, token, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}
	return s.emailSender.SendPasswordReset(ctx, email, token)
}
