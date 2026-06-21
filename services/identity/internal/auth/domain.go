package auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	DisplayName   string    `json:"display_name"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Session struct {
	ID           uuid.UUID      `json:"id"`
	UserID       uuid.UUID      `json:"user_id"`
	RefreshToken string         `json:"-"`
	DeviceInfo   map[string]any `json:"device_info,omitempty"`
	IPAddress    string         `json:"ip_address"`
	ExpiresAt    time.Time      `json:"expires_at"`
	RevokedAt    *time.Time     `json:"revoked_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResult struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
}

type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

type ChangeEmailInput struct {
	CurrentPassword string
	NewEmail        string
}

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
