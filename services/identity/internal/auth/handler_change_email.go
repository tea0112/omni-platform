package auth

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type changeEmailRequestDTO struct {
	CurrentPassword string `json:"current_password"`
	NewEmail        string `json:"new_email"`
}

func (h *Handler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
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

	var req changeEmailRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}

	user, err := h.svc.ChangeEmail(r.Context(), userID, req.CurrentPassword, req.NewEmail)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}

	shared.WriteJSON(w, http.StatusOK, map[string]any{
		"message": "email changed",
		"user": userResponse{
			ID:            user.ID,
			Email:         user.Email,
			DisplayName:   user.DisplayName,
			EmailVerified: user.EmailVerified,
			CreatedAt:     user.CreatedAt,
			UpdatedAt:     user.UpdatedAt,
		},
	})
}
