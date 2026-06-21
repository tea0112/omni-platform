package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . UserRepository,SessionRepository

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*UserCredentialsRow, error)
	GetByEmail(ctx context.Context, email string) (*UserCredentialsRow, error)
	GetByID(ctx context.Context, id uuid.UUID) (*UserCredentialsRow, error)
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	GetPasswordResetToken(ctx context.Context, token string) (userID uuid.UUID, expiresAt time.Time, usedAt *time.Time, err error)
	MarkPasswordResetTokenUsed(ctx context.Context, token string) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	UpdateEmail(ctx context.Context, userID uuid.UUID, email string) error
	GetUserRolesAndPermissions(ctx context.Context, userID uuid.UUID) (roles []string, perms []string, err error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, userID uuid.UUID, refreshToken string, deviceInfo map[string]any, ipAddress string, expiresAt time.Time) (*SessionContextRow, error)
	GetByRefreshToken(ctx context.Context, refreshToken string) (*SessionContextRow, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]SessionContextRow, error)
}

type AuthPGRepository struct {
	defaultQuerier shared.Querier
}

var _ UserRepository = (*AuthPGRepository)(nil)
var _ SessionRepository = (*AuthPGRepository)(nil)

func NewAuthUserPGRepository(pool *pgxpool.Pool) *AuthPGRepository {
	return &AuthPGRepository{defaultQuerier: pool}
}

func NewAuthSessionPGRepository(repo *AuthPGRepository) *AuthPGRepository {
	return repo
}

func (r *AuthPGRepository) q(ctx context.Context) shared.Querier {
	if txQ := shared.QuerierFromContext(ctx); txQ != nil {
		return txQ
	}
	return r.defaultQuerier
}
