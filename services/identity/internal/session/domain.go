package session

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID           uuid.UUID      `json:"id"`
	UserID       uuid.UUID      `json:"user_id"`
	RefreshToken string         `json:"-"`
	DeviceInfo   map[string]any `json:"device_info"`
	IPAddress    string         `json:"ip_address"`
	ExpiresAt    time.Time      `json:"expires_at"`
	RevokedAt    *time.Time     `json:"revoked_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}
