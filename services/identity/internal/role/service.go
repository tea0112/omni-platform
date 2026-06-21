package role

import (
	"context"

	"github.com/google/uuid"

	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

type RoleService struct {
	repo RoleRepository
	rbac *shared.RBAC
}

func NewRoleService(repo RoleRepository, rbac *shared.RBAC) *RoleService {
	return &RoleService{repo: repo, rbac: rbac}
}

func (s *RoleService) Create(ctx context.Context, req CreateRoleRequest) (*Role, error) {
	if err := s.rbac.Can(ctx, "roles.write"); err != nil {
		return nil, err
	}
	row, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	r := row.toDomain()
	return &r, nil
}

func (s *RoleService) GetByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	if err := s.rbac.Can(ctx, "roles.read"); err != nil {
		return nil, err
	}
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r := row.toDomain()
	return &r, nil
}

func (s *RoleService) List(ctx context.Context) ([]Role, error) {
	if err := s.rbac.Can(ctx, "roles.read"); err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]Role, len(rows))
	for i, r := range rows {
		roles[i] = r.toDomain()
	}
	return roles, nil
}

func (s *RoleService) Update(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*Role, error) {
	if err := s.rbac.Can(ctx, "roles.write"); err != nil {
		return nil, err
	}
	row, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	r := row.toDomain()
	return &r, nil
}

func (s *RoleService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.rbac.Can(ctx, "roles.write"); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *RoleService) AddPermission(ctx context.Context, roleID uuid.UUID, permission string) error {
	if err := s.rbac.Can(ctx, "roles.write"); err != nil {
		return err
	}
	return s.repo.AddPermission(ctx, roleID, permission)
}

func (s *RoleService) RemovePermission(ctx context.Context, roleID uuid.UUID, permission string) error {
	if err := s.rbac.Can(ctx, "roles.write"); err != nil {
		return err
	}
	return s.repo.RemovePermission(ctx, roleID, permission)
}

func (s *RoleService) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	if err := s.rbac.Can(ctx, "roles.read"); err != nil {
		return nil, err
	}
	return s.repo.GetPermissions(ctx, roleID)
}

func (s *RoleService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	if err := s.rbac.Can(ctx, "roles.read"); err != nil {
		return nil, err
	}
	rows, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	roles := make([]Role, len(rows))
	for i, r := range rows {
		roles[i] = r.toDomain()
	}
	return roles, nil
}

func (s *RoleService) AssignToUser(ctx context.Context, roleID, userID uuid.UUID) error {
	if err := s.rbac.Can(ctx, "roles.write"); err != nil {
		return err
	}
	return s.repo.AssignToUser(ctx, roleID, userID)
}

func (s *RoleService) RemoveFromUser(ctx context.Context, roleID, userID uuid.UUID) error {
	if err := s.rbac.Can(ctx, "roles.write"); err != nil {
		return err
	}
	return s.repo.RemoveFromUser(ctx, roleID, userID)
}
