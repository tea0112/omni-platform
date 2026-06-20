package session

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . SessionRepository

type SessionRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type SessionPGRepository struct {
	pool *pgxpool.Pool
}

var _ SessionRepository = (*SessionPGRepository)(nil)

func NewSessionRepository(pool *pgxpool.Pool) *SessionPGRepository {
	return &SessionPGRepository{pool: pool}
}

func (r *SessionPGRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, refresh_token, device_info, ip_address, expires_at, revoked_at, created_at FROM sessions WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get sessions: %w", err)
	}
	defer rows.Close()
	var sessions []Session
	for rows.Next() {
		var s Session
		var devJSON []byte
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshToken, &devJSON, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		json.Unmarshal(devJSON, &s.DeviceInfo)
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *SessionPGRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE id = $1`, id)
	return err
}

func (r *SessionPGRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}
