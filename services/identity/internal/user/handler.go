package user

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type Handler struct {
	svc *UserService
}

func NewHandler(svc *UserService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/users/{id}", h.GetByID)
	r.Patch("/api/v1/users/{id}", h.Update)
	r.Get("/api/v1/users", h.List)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	user, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	users, err := h.svc.List(r.Context(), offset, limit)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, users)
}


