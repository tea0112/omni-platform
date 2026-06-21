package auth

import (
	"encoding/json"
	"net/http"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type registerRequestDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	creds, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	u := creds.User()
	shared.WriteJSON(w, http.StatusCreated, userResponse{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		EmailVerified: u.EmailVerified,
	})
}
