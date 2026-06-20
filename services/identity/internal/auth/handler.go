package auth

import (
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *AuthService
}

func NewHandler(svc *AuthService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/auth/register", h.Register)
	r.Post("/api/v1/auth/login", h.Login)
	r.Post("/api/v1/auth/refresh", h.Refresh)
	r.Post("/api/v1/auth/logout", h.Logout)
	r.Post("/api/v1/auth/change-password", h.ChangePassword)
	r.Post("/api/v1/auth/change-email", h.ChangeEmail)
	r.Post("/api/v1/auth/forgot-password", h.ForgotPassword)
	r.Post("/api/v1/auth/reset-password", h.ResetPassword)
}


