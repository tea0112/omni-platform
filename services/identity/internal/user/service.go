package user

import (
	"context"

	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/identityuser"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type UserService struct {
	repo UserRepository
	rbac *shared.RBAC
}

func NewUserService(repo UserRepository, rbac *shared.RBAC) *UserService {
	return &UserService{repo: repo, rbac: rbac}
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*identityuser.User, error) {
	p, _ := shared.GetPrincipal(ctx)
	if p.UserID != id.String() {
		if err := s.rbac.Can(ctx, "users.read"); err != nil {
			return nil, err
		}
	}
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	u := row.toDomain()
	return &u, nil
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*identityuser.User, error) {
	p, _ := shared.GetPrincipal(ctx)
	if p.UserID != id.String() {
		if err := s.rbac.Can(ctx, "users.write", id.String()); err != nil {
			return nil, err
		}
	}
	row, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	u := row.toDomain()
	return &u, nil
}

func (s *UserService) List(ctx context.Context, offset, limit int) ([]identityuser.User, error) {
	if err := s.rbac.Can(ctx, "users.read"); err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	users := make([]identityuser.User, len(rows))
	for i, r := range rows {
		users[i] = r.toDomain()
	}
	return users, nil
}
