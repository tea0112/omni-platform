package auth

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	identityv1 "github.com/tea0112/omni-platform/services/identity/gen/identity/v1"
	"github.com/tea0112/omni-platform/services/identity/gen/identity/v1/identityv1connect"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

var _ identityv1connect.AuthServiceHandler = (*AuthGrpcHandler)(nil)

type AuthGrpcHandler struct {
	svc *AuthService
}

func NewAuthGrpcHandler(svc *AuthService) (string, http.Handler) {
	return identityv1connect.NewAuthServiceHandler(&AuthGrpcHandler{svc: svc})
}

func (h *AuthGrpcHandler) Register(ctx context.Context, req *connect.Request[identityv1.RegisterRequest]) (*connect.Response[identityv1.RegisterResponse], error) {
	creds, err := h.svc.Register(ctx, req.Msg.Email, req.Msg.Password)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.RegisterResponse{
		UserId: creds.User().ID.String(),
		Email:  creds.User().Email,
	}), nil
}

func (h *AuthGrpcHandler) Login(ctx context.Context, req *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error) {
	result, err := h.svc.Login(ctx, req.Msg.Email, req.Msg.Password, "", nil)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt.Unix(),
	}), nil
}

func (h *AuthGrpcHandler) Refresh(ctx context.Context, req *connect.Request[identityv1.RefreshRequest]) (*connect.Response[identityv1.RefreshResponse], error) {
	result, err := h.svc.Refresh(ctx, req.Msg.RefreshToken, "", nil)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.RefreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    result.ExpiresAt.Unix(),
	}), nil
}

func (h *AuthGrpcHandler) Logout(ctx context.Context, req *connect.Request[identityv1.LogoutRequest]) (*connect.Response[identityv1.LogoutResponse], error) {
	p, ok := shared.GetPrincipal(ctx)
	if !ok {
		return nil, shared.AsConnectError(shared.ErrUnauthenticated)
	}
	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"user_id": "invalid"}})
	}
	if err := h.svc.Logout(ctx, userID); err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.LogoutResponse{}), nil
}
