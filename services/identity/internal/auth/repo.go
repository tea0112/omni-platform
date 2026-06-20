package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . UserRepository,SessionRepository

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, userID uuid.UUID, refreshToken string, deviceInfo map[string]any, ipAddress string, expiresAt time.Time) (*Session, error)
	GetByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Session, error)
}

type AuthPGRepository struct {
	pool *pgxpool.Pool
}

var _ UserRepository = (*AuthPGRepository)(nil)
var _ SessionRepository = (*AuthPGRepository)(nil)

func NewAuthRepository(pool *pgxpool.Pool) *AuthPGRepository {
	return &AuthPGRepository{pool: pool}
}
