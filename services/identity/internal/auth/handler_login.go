package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type loginRequestDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponseDTO struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         userResponse `json:"user"`
}

type userResponse struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	DisplayName   string    `json:"display_name"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	ip := r.RemoteAddr
	result, err := h.svc.Login(r.Context(), req.Email, req.Password, ip, nil)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, authResponseDTO{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt,
		User: userResponse{
			ID:            result.User.ID,
			Email:         result.User.Email,
			DisplayName:   result.User.DisplayName,
			EmailVerified: result.User.EmailVerified,
			CreatedAt:     result.User.CreatedAt,
			UpdatedAt:     result.User.UpdatedAt,
		},
	})
}
