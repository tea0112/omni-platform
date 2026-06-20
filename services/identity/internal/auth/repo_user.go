package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *AuthPGRepository) Create(ctx context.Context, email, passwordHash string) (*User, error) {
	user := &User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        email,
		PasswordHash: passwordHash,
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		user.ID, user.Email, user.PasswordHash,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return user, nil
}

func (r *AuthPGRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, email_verified, created_at, updated_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *AuthPGRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, email_verified, created_at, updated_at FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}
