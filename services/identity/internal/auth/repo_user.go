package auth

import (
	"context"
	"fmt"
	"time"

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

func (r *AuthPGRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.Must(uuid.NewV7()), userID, token, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

func (r *AuthPGRepository) GetPasswordResetToken(ctx context.Context, token string) (uuid.UUID, time.Time, *time.Time, error) {
	var userID uuid.UUID
	var expiresAt time.Time
	var usedAt *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, expires_at, used_at FROM password_reset_tokens WHERE token = $1`,
		token,
	).Scan(&userID, &expiresAt, &usedAt)
	if err != nil {
		return uuid.UUID{}, time.Time{}, nil, fmt.Errorf("get password reset token: %w", err)
	}
	return userID, expiresAt, usedAt, nil
}

func (r *AuthPGRepository) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = now() WHERE token = $1`,
		token,
	)
	if err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}

func (r *AuthPGRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`,
		passwordHash, userID,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (r *AuthPGRepository) UpdateEmail(ctx context.Context, userID uuid.UUID, email string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email = $1, email_verified = false, updated_at = now() WHERE id = $2`,
		email, userID,
	)
	if err != nil {
		return fmt.Errorf("update email: %w", err)
	}
	return nil
}

func (r *AuthPGRepository) GetUserRolesAndPermissions(ctx context.Context, userID uuid.UUID) ([]string, []string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.name, rp.permission
		 FROM user_roles ur
		 JOIN roles r ON r.id = ur.role_id
		 JOIN role_permissions rp ON rp.role_id = r.id
		 WHERE ur.user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("get user roles and permissions: %w", err)
	}
	defer rows.Close()

	roleSet := make(map[string]struct{})
	permSet := make(map[string]struct{})
	for rows.Next() {
		var role, perm string
		if err := rows.Scan(&role, &perm); err != nil {
			return nil, nil, fmt.Errorf("scan role/permission: %w", err)
		}
		roleSet[role] = struct{}{}
		permSet[perm] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration: %w", err)
	}

	roles := make([]string, 0, len(roleSet))
	for r := range roleSet {
		roles = append(roles, r)
	}
	perms := make([]string, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}
	return roles, perms, nil
}
