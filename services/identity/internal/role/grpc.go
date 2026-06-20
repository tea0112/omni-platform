package role

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	identityv1 "github.com/tea0112/omni-platform/services/identity/gen/identity/v1"
	"github.com/tea0112/omni-platform/services/identity/gen/identity/v1/identityv1connect"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

var _ identityv1connect.RoleServiceHandler = (*RoleGrpcHandler)(nil)

type RoleGrpcHandler struct {
	svc *RoleService
}

func NewRoleGrpcHandler(svc *RoleService) (string, http.Handler) {
	handler := &RoleGrpcHandler{svc: svc}
	return identityv1connect.NewRoleServiceHandler(handler)
}

func (h *RoleGrpcHandler) CreateRole(ctx context.Context, req *connect.Request[identityv1.CreateRoleRequest]) (*connect.Response[identityv1.RoleResponse], error) {
	role, err := h.svc.Create(ctx, CreateRoleRequest{
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
	})
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.RoleResponse{
		RoleId:      role.ID.String(),
		Name:        role.Name,
		Description: role.Description,
	}), nil
}

func (h *RoleGrpcHandler) ListRoles(ctx context.Context, req *connect.Request[identityv1.ListRolesRequest]) (*connect.Response[identityv1.ListRolesResponse], error) {
	roles, err := h.svc.List(ctx)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	resp := &identityv1.ListRolesResponse{}
	for _, r := range roles {
		resp.Roles = append(resp.Roles, &identityv1.RoleResponse{
			RoleId:      r.ID.String(),
			Name:        r.Name,
			Description: r.Description,
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *RoleGrpcHandler) DeleteRole(ctx context.Context, req *connect.Request[identityv1.DeleteRoleRequest]) (*connect.Response[identityv1.DeleteRoleResponse], error) {
	roleID, err := uuid.Parse(req.Msg.RoleId)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"role_id": "invalid uuid"}})
	}
	if err := h.svc.Delete(ctx, roleID); err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.DeleteRoleResponse{}), nil
}

func (h *RoleGrpcHandler) AssignRole(ctx context.Context, req *connect.Request[identityv1.AssignRoleRequest]) (*connect.Response[identityv1.AssignRoleResponse], error) {
	userID, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"user_id": "invalid uuid"}})
	}
	roleID, err := uuid.Parse(req.Msg.RoleId)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"role_id": "invalid uuid"}})
	}
	if err := h.svc.AssignToUser(ctx, roleID, userID); err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.AssignRoleResponse{}), nil
}
