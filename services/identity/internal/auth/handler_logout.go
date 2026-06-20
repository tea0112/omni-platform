package auth

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.Logout(r.Context(), userID); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}
