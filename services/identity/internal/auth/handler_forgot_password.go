package auth

import (
	"encoding/json"
	"net/http"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	if err := h.svc.ForgotPassword(r.Context(), req.Email); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a reset link has been sent"})
}
