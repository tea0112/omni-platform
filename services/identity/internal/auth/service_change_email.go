package auth

import (
	"context"
	"fmt"
	"net/mail"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/identityuser"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (s *AuthService) ChangeEmail(ctx context.Context, userID uuid.UUID, currentPassword, newEmail string) (*identityuser.User, error) {
	if currentPassword == "" || newEmail == "" {
		return nil, &shared.ValidationError{Fields: map[string]string{
			"current_password": "required",
			"new_email":        "required",
		}}
	}

	if _, err := mail.ParseAddress(newEmail); err != nil {
		return nil, &shared.ValidationError{Fields: map[string]string{
			"new_email": "invalid email format",
		}}
	}

	row, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	creds := row.toDomain()

	if err := s.hasher.Compare(creds.PasswordHash(), currentPassword); err != nil {
		return nil, shared.ErrUnauthenticated
	}

	if err := s.userRepo.UpdateEmail(ctx, userID, newEmail); err != nil {
		return nil, fmt.Errorf("update email: %w", err)
	}

	user := creds.User()
	user.Email = newEmail

	return &user, nil
}
