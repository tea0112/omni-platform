package user

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	identityv1 "github.com/tea0112/omni-platform/services/identity/gen/identity/v1"
	"github.com/tea0112/omni-platform/services/identity/gen/identity/v1/identityv1connect"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

var _ identityv1connect.UserServiceHandler = (*UserGrpcHandler)(nil)

type UserGrpcHandler struct {
	svc *UserService
}

func NewUserGrpcHandler(svc *UserService) (string, http.Handler) {
	handler := &UserGrpcHandler{svc: svc}
	return identityv1connect.NewUserServiceHandler(handler)
}

func (h *UserGrpcHandler) GetUser(ctx context.Context, req *connect.Request[identityv1.GetUserRequest]) (*connect.Response[identityv1.GetUserResponse], error) {
	id, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"user_id": "invalid uuid"}})
	}
	u, err := h.svc.GetByID(ctx, id)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.GetUserResponse{
		UserId:      u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
	}), nil
}

func (h *UserGrpcHandler) UpdateUser(ctx context.Context, req *connect.Request[identityv1.UpdateUserRequest]) (*connect.Response[identityv1.UpdateUserResponse], error) {
	id, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"user_id": "invalid uuid"}})
	}
	u, err := h.svc.Update(ctx, id, UpdateUserRequest{DisplayName: req.Msg.DisplayName})
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.UpdateUserResponse{
		UserId:      u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
	}), nil
}

func (h *UserGrpcHandler) ListUsers(ctx context.Context, req *connect.Request[identityv1.ListUsersRequest]) (*connect.Response[identityv1.ListUsersResponse], error) {
	offset := int(req.Msg.Offset)
	limit := int(req.Msg.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	users, err := h.svc.List(ctx, offset, limit)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	resp := &identityv1.ListUsersResponse{}
	for i := range users {
		u := users[i]
		resp.Users = append(resp.Users, &identityv1.GetUserResponse{
			UserId:      u.ID.String(),
			Email:       u.Email,
			DisplayName: u.DisplayName,
		})
	}
	return connect.NewResponse(resp), nil
}
