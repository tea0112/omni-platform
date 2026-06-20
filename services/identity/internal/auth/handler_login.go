package auth

import (
	"encoding/json"
	"net/http"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
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
	shared.WriteJSON(w, http.StatusOK, result)
}
