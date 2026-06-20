package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *AuthPGRepository) CreateSession(ctx context.Context, userID uuid.UUID, refreshToken string, deviceInfo map[string]any, ipAddress string, expiresAt time.Time) (*Session, error) {
	deviceJSON, _ := json.Marshal(deviceInfo)
	session := &Session{
		ID:           uuid.Must(uuid.NewV7()),
		UserID:       userID,
		RefreshToken: refreshToken,
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		ExpiresAt:    expiresAt,
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, device_info, ip_address, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		session.ID, session.UserID, session.RefreshToken, deviceJSON, session.IPAddress, session.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return session, nil
}

func (r *AuthPGRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	session := &Session{}
	var deviceJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, refresh_token, device_info, ip_address, expires_at, revoked_at, created_at FROM sessions WHERE refresh_token = $1`,
		refreshToken,
	).Scan(&session.ID, &session.UserID, &session.RefreshToken, &deviceJSON, &session.IPAddress, &session.ExpiresAt, &session.RevokedAt, &session.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get session by refresh token: %w", err)
	}
	json.Unmarshal(deviceJSON, &session.DeviceInfo)
	return session, nil
}

func (r *AuthPGRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`,
		id,
	)
	return err
}

func (r *AuthPGRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	return err
}

func (r *AuthPGRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, refresh_token, device_info, ip_address, expires_at, revoked_at, created_at FROM sessions WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var deviceJSON []byte
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshToken, &deviceJSON, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		json.Unmarshal(deviceJSON, &s.DeviceInfo)
		sessions = append(sessions, s)
	}
	return sessions, nil
}
