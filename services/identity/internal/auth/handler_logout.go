package auth

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	p, ok := shared.GetPrincipal(r.Context())
	if !ok {
		writeErr(w, shared.ErrUnauthenticated)
		return
	}
	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		writeErr(w, &shared.ValidationError{Fields: map[string]string{"user_id": "invalid uuid"}})
		return
	}
	if err := h.svc.Logout(r.Context(), userID); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}
