package auth

import (
	"context"
	"fmt"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (s *AuthService) Register(ctx context.Context, email, password string) (*UserCredentials, error) {
	if email == "" || password == "" {
		return nil, &shared.ValidationError{Fields: map[string]string{
			"email":    "required",
			"password": "required",
		}}
	}

	_, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return nil, shared.ErrDuplicate
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	row, err := s.userRepo.Create(ctx, email, hash)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return row.toDomain(), nil
}
