package session

import (
	"net/http"
	"time"

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

type sessionResponse struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func toSessionResponse(s Session) sessionResponse {
	return sessionResponse{
		ID:        s.ID,
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
		RevokedAt: s.RevokedAt,
		CreatedAt: s.CreatedAt,
	}
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
	sessions, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	resp := make([]sessionResponse, len(sessions))
	for i, s := range sessions {
		resp[i] = toSessionResponse(s)
	}
	shared.WriteJSON(w, http.StatusOK, resp)
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
	if err := h.svc.RevokeAllForUser(r.Context(), userID); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "all sessions revoked"})
}
