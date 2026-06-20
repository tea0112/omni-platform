package role

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type Handler struct {
	svc *RoleService
}

func NewHandler(svc *RoleService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/roles", h.List)
	r.Post("/api/v1/roles", h.Create)
	r.Get("/api/v1/roles/{id}", h.GetByID)
	r.Put("/api/v1/roles/{id}", h.Update)
	r.Delete("/api/v1/roles/{id}", h.Delete)
	r.Get("/api/v1/roles/{id}/permissions", h.ListPermissions)
	r.Post("/api/v1/roles/{id}/permissions", h.AddPermission)
	r.Delete("/api/v1/roles/{id}/permissions/{permission}", h.RemovePermission)
	r.Post("/api/v1/roles/{roleID}/users", h.AssignToUser)
	r.Delete("/api/v1/roles/{roleID}/users/{userID}", h.RemoveFromUser)
	r.Get("/api/v1/users/{userID}/roles", h.GetUserRoles)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	role, err := h.svc.Create(r.Context(), req)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusCreated, role)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	role, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, role)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.List(r.Context())
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, roles)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	role, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, role)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "role deleted"})
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	permissions, err := h.svc.GetPermissions(r.Context(), id)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, permissions)
}

func (h *Handler) AddPermission(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	var req AddPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	if err := h.svc.AddPermission(r.Context(), id, req.Permission); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "permission added"})
}

func (h *Handler) RemovePermission(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	permission := chi.URLParam(r, "permission")
	if permission == "" {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"permission": "required"}})
		return
	}
	if err := h.svc.RemovePermission(r.Context(), id, permission); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "permission removed"})
}

func (h *Handler) AssignToUser(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"roleID": "invalid uuid"}})
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteErr(w, err)
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"user_id": "invalid uuid"}})
		return
	}
	if err := h.svc.AssignToUser(r.Context(), roleID, userID); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "role assigned"})
}

func (h *Handler) RemoveFromUser(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"roleID": "invalid uuid"}})
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"userID": "invalid uuid"}})
		return
	}
	if err := h.svc.RemoveFromUser(r.Context(), roleID, userID); err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "role removed from user"})
}

func (h *Handler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"userID": "invalid uuid"}})
		return
	}
	roles, err := h.svc.GetUserRoles(r.Context(), userID)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, roles)
}


