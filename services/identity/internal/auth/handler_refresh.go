package auth

import (
	"encoding/json"
	"net/http"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type refreshRequestDTO struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	ip := r.RemoteAddr
	result, err := h.svc.Refresh(r.Context(), req.RefreshToken, ip, nil)
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
