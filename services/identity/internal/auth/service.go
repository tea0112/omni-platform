package auth

import (
	"time"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type AuthService struct {
	userRepo        UserRepository
	sessionRepo     SessionRepository
	txr             shared.TxRunner
	hasher          *shared.PasswordHasher
	tokenSvc        *shared.TokenService
	rbac            *shared.RBAC
	emailSender     shared.EmailSender
	refreshTokenTTL time.Duration
}

func NewAuthService(
	userRepo UserRepository,
	sessionRepo SessionRepository,
	txr shared.TxRunner,
	hasher *shared.PasswordHasher,
	tokenSvc *shared.TokenService,
	rbac *shared.RBAC,
	emailSender shared.EmailSender,
	refreshTokenTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		sessionRepo:     sessionRepo,
		txr:             txr,
		hasher:          hasher,
		tokenSvc:        tokenSvc,
		rbac:            rbac,
		emailSender:     emailSender,
		refreshTokenTTL: refreshTokenTTL,
	}
}
