package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tea0112/omni-platform/services/identity/internal/identityuser"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . UserRepository

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*UserRow, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*UserRow, error)
	List(ctx context.Context, offset, limit int) ([]UserRow, error)
}

// UserRow is the database representation of a user.
type UserRow struct {
	ID            uuid.UUID
	Email         string
	DisplayName   string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (r UserRow) toDomain() identityuser.User {
	return identityuser.User{
		ID:            r.ID,
		Email:         r.Email,
		DisplayName:   r.DisplayName,
		EmailVerified: r.EmailVerified,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

type UserPGRepository struct {
	defaultQuerier shared.Querier
}

var _ UserRepository = (*UserPGRepository)(nil)

func NewUserPGRepository(pool *pgxpool.Pool) *UserPGRepository {
	return &UserPGRepository{defaultQuerier: pool}
}

func (r *UserPGRepository) q(ctx context.Context) shared.Querier {
	if txQ := shared.QuerierFromContext(ctx); txQ != nil {
		return txQ
	}
	return r.defaultQuerier
}

func (r *UserPGRepository) GetByID(ctx context.Context, id uuid.UUID) (*UserRow, error) {
	row := &UserRow{}
	err := r.q(ctx).QueryRow(ctx,
		`SELECT id, email, display_name, email_verified, created_at, updated_at FROM users WHERE id = $1`, id,
	).Scan(&row.ID, &row.Email, &row.DisplayName, &row.EmailVerified, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return row, nil
}

func (r *UserPGRepository) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*UserRow, error) {
	if req.DisplayName != nil {
		_, err := r.q(ctx).Exec(ctx,
			`UPDATE users SET display_name = $1, updated_at = now() WHERE id = $2`, *req.DisplayName, id)
		if err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
	}
	return r.GetByID(ctx, id)
}

func (r *UserPGRepository) List(ctx context.Context, offset, limit int) ([]UserRow, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT id, email, display_name, email_verified, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []UserRow
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}
