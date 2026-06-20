package session

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	identityv1 "github.com/tea0112/omni-platform/services/identity/gen/identity/v1"
	"github.com/tea0112/omni-platform/services/identity/gen/identity/v1/identityv1connect"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

var _ identityv1connect.SessionServiceHandler = (*SessionGrpcHandler)(nil)

type SessionGrpcHandler struct {
	svc *SessionService
}

func NewSessionGrpcHandler(svc *SessionService) (string, http.Handler) {
	handler := &SessionGrpcHandler{svc: svc}
	return identityv1connect.NewSessionServiceHandler(handler)
}

func (h *SessionGrpcHandler) ListSessions(ctx context.Context, req *connect.Request[identityv1.ListSessionsRequest]) (*connect.Response[identityv1.ListSessionsResponse], error) {
	userID, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"user_id": "invalid uuid"}})
	}
	sessions, err := h.svc.List(ctx, userID)
	if err != nil {
		return nil, shared.AsConnectError(err)
	}
	resp := &identityv1.ListSessionsResponse{}
	for _, s := range sessions {
		resp.Sessions = append(resp.Sessions, &identityv1.SessionInfo{
			SessionId: s.ID.String(),
			UserId:    s.UserID.String(),
			IpAddress: s.IPAddress,
			ExpiresAt: s.ExpiresAt.Unix(),
			CreatedAt: s.CreatedAt.Unix(),
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *SessionGrpcHandler) RevokeSession(ctx context.Context, req *connect.Request[identityv1.RevokeSessionRequest]) (*connect.Response[identityv1.RevokeSessionResponse], error) {
	sessionID, err := uuid.Parse(req.Msg.SessionId)
	if err != nil {
		return nil, shared.AsConnectError(&shared.ValidationError{Fields: map[string]string{"session_id": "invalid uuid"}})
	}
	if err := h.svc.Revoke(ctx, sessionID); err != nil {
		return nil, shared.AsConnectError(err)
	}
	return connect.NewResponse(&identityv1.RevokeSessionResponse{}), nil
}
