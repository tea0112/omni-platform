package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type UserService struct {
	repo UserRepository
	rbac *shared.RBAC
}

func NewUserService(repo UserRepository, rbac *shared.RBAC) *UserService {
	return &UserService{repo: repo, rbac: rbac}
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if err := s.rbac.Can(ctx, "users.read"); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*User, error) {
	if err := s.rbac.Can(ctx, "users.write", id.String()); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, req)
}

func (s *UserService) List(ctx context.Context, offset, limit int) ([]User, error) {
	if err := s.rbac.Can(ctx, "users.read"); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, offset, limit)
}
