package user

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/identityuser"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type Handler struct {
	svc *UserService
}

func NewHandler(svc *UserService) *Handler {
	return &Handler{svc: svc}
}

type userResponse struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	DisplayName   string    `json:"display_name"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toUserResponse(u identityuser.User) userResponse {
	return userResponse{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

type updateUserRequestDTO struct {
	DisplayName *string `json:"display_name"`
}

func (d updateUserRequestDTO) toDomain() UpdateUserRequest {
	return UpdateUserRequest{DisplayName: d.DisplayName}
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
	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, toUserResponse(*u))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.WriteErr(w, &shared.ValidationError{Fields: map[string]string{"id": "invalid uuid"}})
		return
	}
	var dto updateUserRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		shared.WriteErr(w, err)
		return
	}
	u, err := h.svc.Update(r.Context(), id, dto.toDomain())
	if err != nil {
		shared.WriteErr(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, toUserResponse(*u))
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
	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = toUserResponse(u)
	}
	shared.WriteJSON(w, http.StatusOK, resp)
}
