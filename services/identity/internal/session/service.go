package session

import (
	"context"

	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type SessionService struct {
	repo SessionRepository
	rbac *shared.RBAC
}

func NewSessionService(repo SessionRepository, rbac *shared.RBAC) *SessionService {
	return &SessionService{repo: repo, rbac: rbac}
}

func (s *SessionService) ListByUser(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	p, _ := shared.GetPrincipal(ctx)
	if p.UserID != userID.String() {
		if err := s.rbac.Can(ctx, "sessions.read"); err != nil {
			return nil, err
		}
	}
	rows, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	sessions := make([]Session, len(rows))
	for i, r := range rows {
		sessions[i] = r.toDomain()
	}
	return sessions, nil
}

func (s *SessionService) Revoke(ctx context.Context, id uuid.UUID) error {
	if err := s.rbac.Can(ctx, "sessions.write"); err != nil {
		return err
	}
	return s.repo.Revoke(ctx, id)
}

func (s *SessionService) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	p, _ := shared.GetPrincipal(ctx)
	if p.UserID != userID.String() {
		if err := s.rbac.Can(ctx, "sessions.write"); err != nil {
			return err
		}
	}
	return s.repo.RevokeAllForUser(ctx, userID)
}
