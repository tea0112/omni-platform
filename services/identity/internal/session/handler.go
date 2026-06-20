package session

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type Handler struct {
	svc *SessionService
}

func NewHandler(svc *SessionService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/users/{userID}/sessions", h.List)
	r.Delete("/api/v1/users/{userID}/sessions/{sessionID}", h.Revoke)
	r.Delete("/api/v1/users/{userID}/sessions", h.RevokeAll)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"userID": "invalid uuid"}})
		return
	}
	sessions, err := h.svc.List(r.Context(), userID)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, sessions)
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"sessionID": "invalid uuid"}})
		return
	}
	if err := h.svc.Revoke(r.Context(), sessionID); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "session revoked"})
}

func (h *Handler) RevokeAll(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"userID": "invalid uuid"}})
		return
	}
	if err := h.svc.RevokeAll(r.Context(), userID); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "all sessions revoked"})
}


