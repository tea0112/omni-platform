package auth

import (
	"encoding/json"
	"net/http"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "password reset successful"})
}
