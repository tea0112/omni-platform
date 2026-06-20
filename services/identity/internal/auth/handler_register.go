package auth

import (
	"encoding/json"
	"net/http"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	user, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusCreated, user)
}
