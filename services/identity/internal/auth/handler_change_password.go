package auth

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type changePasswordRequestDTO struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	p, ok := shared.GetPrincipal(r.Context())
	if !ok {
		shared.WriteErr(w, shared.ErrUnauthenticated)
		return
	}
	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"user_id": "invalid uuid"}})
		return
	}

	var req changePasswordRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}

	if err := h.svc.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		shared.WriteErr(w, err)
		return
	}

	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "password changed"})
}
