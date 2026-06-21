package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SessionContextRow struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	RefreshToken string
	DeviceInfo   []byte
	IPAddress    string
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}

func (r SessionContextRow) toSessionWithContext() (*SessionWithContext, error) {
	var device map[string]any
	if len(r.DeviceInfo) > 0 {
		if err := json.Unmarshal(r.DeviceInfo, &device); err != nil {
			return nil, fmt.Errorf("unmarshal device info: %w", err)
		}
	}
	return &SessionWithContext{
		ID:           r.ID,
		UserID:       r.UserID,
		RefreshToken: r.RefreshToken,
		DeviceInfo:   device,
		IPAddress:    r.IPAddress,
		ExpiresAt:    r.ExpiresAt,
		RevokedAt:    r.RevokedAt,
		CreatedAt:    r.CreatedAt,
	}, nil
}

func (r *AuthPGRepository) CreateSession(ctx context.Context, userID uuid.UUID, refreshToken string, deviceInfo map[string]any, ipAddress string, expiresAt time.Time) (*SessionContextRow, error) {
	deviceJSON, _ := json.Marshal(deviceInfo)
	id := uuid.Must(uuid.NewV7())
	_, err := r.q(ctx).Exec(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, device_info, ip_address, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, userID, refreshToken, deviceJSON, ipAddress, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return r.GetByRefreshToken(ctx, refreshToken)
}

func (r *AuthPGRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*SessionContextRow, error) {
	row := &SessionContextRow{}
	var deviceJSON []byte
	err := r.q(ctx).QueryRow(ctx,
		`SELECT id, user_id, refresh_token, device_info, ip_address, expires_at, revoked_at, created_at FROM sessions WHERE refresh_token = $1`,
		refreshToken,
	).Scan(&row.ID, &row.UserID, &row.RefreshToken, &deviceJSON, &row.IPAddress, &row.ExpiresAt, &row.RevokedAt, &row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get session by refresh token: %w", err)
	}
	row.DeviceInfo = deviceJSON
	return row, nil
}

func (r *AuthPGRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`,
		id,
	)
	return err
}

func (r *AuthPGRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	return err
}

func (r *AuthPGRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]SessionContextRow, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT id, user_id, refresh_token, device_info, ip_address, expires_at, revoked_at, created_at FROM sessions WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionContextRow
	for rows.Next() {
		var s SessionContextRow
		var deviceJSON []byte
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshToken, &deviceJSON, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.DeviceInfo = deviceJSON
		sessions = append(sessions, s)
	}
	return sessions, nil
}
