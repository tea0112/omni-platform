package auth

import (
	"context"

	"github.com/google/uuid"
)

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	return s.sessionRepo.RevokeAllForUser(ctx, userID)
}
