package auth

import (
	"time"

	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/identityuser"
)

type UserCredentials struct {
	user         identityuser.User
	passwordHash string
}

func NewUserCredentials(u identityuser.User, hash string) *UserCredentials {
	return &UserCredentials{user: u, passwordHash: hash}
}

func (c *UserCredentials) User() identityuser.User { return c.user }
func (c *UserCredentials) PasswordHash() string    { return c.passwordHash }

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	User         identityuser.User
}

type Credentials struct {
	Email    string
	Password string
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

type SessionContext struct {
	RefreshToken string
	DeviceInfo   map[string]any
	IPAddress    string
}

type SessionWithContext struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	RefreshToken string
	DeviceInfo   map[string]any
	IPAddress    string
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}
