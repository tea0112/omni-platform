package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . UserRepository

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*User, error)
	List(ctx context.Context, offset, limit int) ([]User, error)
}

type UserPGRepository struct {
	pool *pgxpool.Pool
}

var _ UserRepository = (*UserPGRepository)(nil)

func NewUserRepository(pool *pgxpool.Pool) *UserPGRepository {
	return &UserPGRepository{pool: pool}
}

func (r *UserPGRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, display_name, email_verified, created_at, updated_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (r *UserPGRepository) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*User, error) {
	if req.DisplayName != nil {
		_, err := r.pool.Exec(ctx, `UPDATE users SET display_name = $1, updated_at = now() WHERE id = $2`, *req.DisplayName, id)
		if err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
	}
	return r.GetByID(ctx, id)
}

func (r *UserPGRepository) List(ctx context.Context, offset, limit int) ([]User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, display_name, email_verified, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}
