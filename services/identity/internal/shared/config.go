package shared

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

type ServerConfig struct {
	Port int
}

type AuthConfig struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	BcryptCost      int
	JWTPrivateKey   ed25519.PrivateKey
	JWTPublicKey    ed25519.PublicKey
}

type OTELConfig struct {
	Endpoint string
}

type EmailConfig struct {
	Provider string
	SMTP     SMTPConfig
}

type Config struct {
	DB     DBConfig
	Server ServerConfig
	Auth   AuthConfig
	OTEL   OTELConfig
	Email  EmailConfig
}

func MustLoad() Config {
	v := viper.New()
	v.SetEnvPrefix("IDENTITY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("db.host", "localhost")
	v.SetDefault("db.port", 5432)
	v.SetDefault("db.user", "identity")
	v.SetDefault("db.password", "identity")
	v.SetDefault("db.name", "identity")
	v.SetDefault("db.sslmode", "disable")
	v.SetDefault("server.port", 8080)
	v.SetDefault("auth.access_token_ttl", "15m")
	v.SetDefault("auth.refresh_token_ttl", "672h")
	v.SetDefault("auth.bcrypt_cost", 12)
	v.SetDefault("otel.endpoint", "localhost:4317")
	v.SetDefault("email.provider", "log")
	v.SetDefault("email.smtp.host", "localhost")
	v.SetDefault("email.smtp.port", 587)

	cfg := Config{
		DB: DBConfig{
			Host:     v.GetString("db.host"),
			Port:     v.GetInt("db.port"),
			User:     v.GetString("db.user"),
			Password: v.GetString("db.password"),
			Name:     v.GetString("db.name"),
			SSLMode:  v.GetString("db.sslmode"),
		},
		Server: ServerConfig{Port: v.GetInt("server.port")},
		Auth: AuthConfig{
			AccessTokenTTL:  v.GetDuration("auth.access_token_ttl"),
			RefreshTokenTTL: v.GetDuration("auth.refresh_token_ttl"),
			BcryptCost:      v.GetInt("auth.bcrypt_cost"),
		},
		OTEL: OTELConfig{Endpoint: v.GetString("otel.endpoint")},
		Email: EmailConfig{
			Provider: v.GetString("email.provider"),
			SMTP: SMTPConfig{
				Host:     v.GetString("email.smtp.host"),
				Port:     v.GetInt("email.smtp.port"),
				Username: v.GetString("email.smtp.username"),
				Password: v.GetString("email.smtp.password"),
				From:     v.GetString("email.smtp.from"),
			},
		},
	}

	jwkJSON := v.GetString("auth.jwt_private_key_jwk")
	if jwkJSON == "" {
		slog.Error("IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK is required")
		panic("IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK is required")
	}
	priv, pub, err := parseEd25519JWK(jwkJSON)
	if err != nil {
		slog.Error("invalid Ed25519 JWK", "error", err)
		panic("invalid Ed25519 JWK")
	}
	cfg.Auth.JWTPrivateKey = priv
	cfg.Auth.JWTPublicKey = pub

	return cfg
}

func parseEd25519JWK(jwkJSON string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	var jwk struct {
		D string `json:"d"`
		X string `json:"x"`
	}
	if err := json.Unmarshal([]byte(jwkJSON), &jwk); err != nil {
		return nil, nil, fmt.Errorf("parse jwk: %w", err)
	}
	dBytes, err := base64.RawURLEncoding.DecodeString(jwk.D)
	if err != nil {
		return nil, nil, fmt.Errorf("decode d: %w", err)
	}
	if len(dBytes) != ed25519.SeedSize {
		return nil, nil, fmt.Errorf("seed must be %d bytes, got %d", ed25519.SeedSize, len(dBytes))
	}
	priv := ed25519.NewKeyFromSeed(dBytes)
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, nil, fmt.Errorf("decode x: %w", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	if !bytes.Equal(xBytes, pub) {
		return nil, nil, fmt.Errorf("public key in JWK does not match seed-derived public key")
	}
	return priv, pub, nil
}
