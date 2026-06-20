package shared

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	jwt.RegisteredClaims
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type TokenService struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	ttl        time.Duration
}

func NewTokenService(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey, ttl time.Duration) *TokenService {
	return &TokenService{privateKey: privateKey, publicKey: publicKey, ttl: ttl}
}

func (s *TokenService) GenerateAccessToken(subject string, roles, perms []string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.ttl)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.Must(uuid.NewV7()).String(),
		},
		Roles:       roles,
		Permissions: perms,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

func (s *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("parse token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenExpired
	}
	return claims, nil
}
