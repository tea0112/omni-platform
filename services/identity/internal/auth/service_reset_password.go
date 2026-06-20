package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	var userID uuid.UUID
	var expiresAt time.Time
	var usedAt *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, expires_at, used_at FROM password_reset_tokens WHERE token = $1`,
		token,
	).Scan(&userID, &expiresAt, &usedAt)
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
	_, err = s.pool.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`, hash, userID)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	_, err = s.pool.Exec(ctx, `UPDATE password_reset_tokens SET used_at = now() WHERE token = $1`, token)
	return err
}
