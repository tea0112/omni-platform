package auth

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type AuthService struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
	hasher      *shared.PasswordHasher
	tokenSvc    *shared.TokenService
	rbac        *shared.RBAC
	emailSender shared.EmailSender
	pool        *pgxpool.Pool
}

func NewAuthService(
	userRepo UserRepository,
	sessionRepo SessionRepository,
	hasher *shared.PasswordHasher,
	tokenSvc *shared.TokenService,
	rbac *shared.RBAC,
	emailSender shared.EmailSender,
	pool *pgxpool.Pool,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		hasher:      hasher,
		tokenSvc:    tokenSvc,
		rbac:        rbac,
		emailSender: emailSender,
		pool:        pool,
	}
}
