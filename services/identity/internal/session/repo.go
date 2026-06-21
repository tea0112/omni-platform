package session

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . SessionRepository

type SessionRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]SessionRow, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type SessionRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (r SessionRow) toDomain() Session {
	return Session{
		ID:        r.ID,
		UserID:    r.UserID,
		ExpiresAt: r.ExpiresAt,
		RevokedAt: r.RevokedAt,
		CreatedAt: r.CreatedAt,
	}
}

type SessionPGRepository struct {
	defaultQuerier shared.Querier
}

var _ SessionRepository = (*SessionPGRepository)(nil)

func NewSessionPGRepository(pool *pgxpool.Pool) *SessionPGRepository {
	return &SessionPGRepository{defaultQuerier: pool}
}

func (r *SessionPGRepository) q(ctx context.Context) shared.Querier {
	if txQ := shared.QuerierFromContext(ctx); txQ != nil {
		return txQ
	}
	return r.defaultQuerier
}

func (r *SessionPGRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]SessionRow, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT id, user_id, expires_at, revoked_at, created_at FROM sessions WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get sessions: %w", err)
	}
	defer rows.Close()
	var sessions []SessionRow
	for rows.Next() {
		var s SessionRow
		if err := rows.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return sessions, nil
}

func (r *SessionPGRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE id = $1`, id)
	return err
}

func (r *SessionPGRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}
