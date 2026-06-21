package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/fx"

	"github.com/tea0112/omni-platform/services/identity/internal/auth"
	"github.com/tea0112/omni-platform/services/identity/internal/role"
	"github.com/tea0112/omni-platform/services/identity/internal/session"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
	"github.com/tea0112/omni-platform/services/identity/internal/user"
)

type GrpcHandlerPair struct {
	Path     string
	Handler  http.Handler
	SkipAuth bool
}

func main() {
	fx.New(
		fx.Provide(
			shared.MustLoad,
			NewLogger,
			NewPasswordHasherFromConfig,
			shared.NewRBAC,
			NewTokenServiceFromConfig,
			NewRefreshTokenTTL,
			NewEmailSenderFromConfig,
			NewDBPoolFromConfig,
			NewTracerProviderFromConfig,
			fx.Annotate(auth.NewAuthRepository, fx.As(new(auth.UserRepository)), fx.As(new(auth.SessionRepository))),
			fx.Annotate(user.NewUserRepository, fx.As(new(user.UserRepository))),
			fx.Annotate(session.NewSessionRepository, fx.As(new(session.SessionRepository))),
			fx.Annotate(role.NewRoleRepository, fx.As(new(role.RoleRepository))),
			auth.NewAuthService,
			user.NewUserService,
			session.NewSessionService,
			role.NewRoleService,
			auth.NewHandler,
			user.NewHandler,
			session.NewHandler,
			role.NewHandler,
			fx.Annotated{Group: "grpc-handlers", Target: func(svc *auth.AuthService) GrpcHandlerPair {
				p, h := auth.NewAuthGrpcHandler(svc)
				return GrpcHandlerPair{Path: p, Handler: h, SkipAuth: true}
			}},
			fx.Annotated{Group: "grpc-handlers", Target: func(svc *user.UserService) GrpcHandlerPair {
				p, h := user.NewUserGrpcHandler(svc)
				return GrpcHandlerPair{Path: p, Handler: h}
			}},
			fx.Annotated{Group: "grpc-handlers", Target: func(svc *session.SessionService) GrpcHandlerPair {
				p, h := session.NewSessionGrpcHandler(svc)
				return GrpcHandlerPair{Path: p, Handler: h}
			}},
			fx.Annotated{Group: "grpc-handlers", Target: func(svc *role.RoleService) GrpcHandlerPair {
				p, h := role.NewRoleGrpcHandler(svc)
				return GrpcHandlerPair{Path: p, Handler: h}
			}},
		),
		fx.Invoke(
			RunMigrations,
			fx.Annotate(Serve, fx.ParamTags(
				``, ``,
				``, ``,
				``, ``,
				`group:"grpc-handlers"`,
				``, ``,
			)),
		),
	).Run()
}

func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func NewPasswordHasherFromConfig(cfg shared.Config) *shared.PasswordHasher {
	return shared.NewPasswordHasher(cfg.Auth.BcryptCost)
}

func NewTokenServiceFromConfig(cfg shared.Config) *shared.TokenService {
	return shared.NewTokenService(cfg.Auth.JWTPrivateKey, cfg.Auth.JWTPublicKey, cfg.Auth.AccessTokenTTL)
}

func NewRefreshTokenTTL(cfg shared.Config) time.Duration {
	return cfg.Auth.RefreshTokenTTL
}

func NewEmailSenderFromConfig(cfg shared.Config, logger *slog.Logger) shared.EmailSender {
	if cfg.Email.Provider == "smtp" {
		return shared.NewSMTPEmailSender(cfg.Email.SMTP)
	}
	return shared.NewLogEmailSender(logger)
}

func NewDBPoolFromConfig(lc fx.Lifecycle, cfg shared.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DB.DSN())
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return pool.Ping(ctx)
		},
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return nil
		},
	})
	return pool, nil
}

func NewTracerProviderFromConfig(cfg shared.Config) (*sdktrace.TracerProvider, error) {
	return shared.NewTracerProvider(context.Background(), cfg.OTEL.Endpoint)
}

func RunMigrations(cfg shared.Config) error {
	return shared.RunMigrations(cfg.DB.DSN())
}

func Serve(lc fx.Lifecycle, cfg shared.Config, authHandler *auth.Handler, userHandler *user.Handler, sessionHandler *session.Handler, roleHandler *role.Handler, grpcHandlers []GrpcHandlerPair, tokenSvc *shared.TokenService, tp *sdktrace.TracerProvider) {
	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Recoverer)
	mux.Use(otelhttp.NewMiddleware("identity-service"))
	mux.Use(middleware.Timeout(30 * time.Second))

	mux.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	authHandler.RegisterRoutes(mux)

	mux.Group(func(r chi.Router) {
		r.Use(shared.Authenticate(tokenSvc))
		userHandler.RegisterRoutes(r)
		sessionHandler.RegisterRoutes(r)
		roleHandler.RegisterRoutes(r)
	})

	authMiddleware := shared.Authenticate(tokenSvc)
	for _, gh := range grpcHandlers {
		h := gh.Handler
		if !gh.SkipAuth {
			h = authMiddleware(h)
		}
		mux.Handle(gh.Path, h)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: mux,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go srv.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			tp.Shutdown(ctx)
			return srv.Shutdown(ctx)
		},
	})
}
