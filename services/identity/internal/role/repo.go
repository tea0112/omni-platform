package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

//go:generate mockgen -destination=mocks/repo_mock.go -package=mocks . RoleRepository

type RoleRepository interface {
	Create(ctx context.Context, req CreateRoleRequest) (*Role, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	List(ctx context.Context) ([]Role, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*Role, error)
	Delete(ctx context.Context, id uuid.UUID) error
	AddPermission(ctx context.Context, roleID uuid.UUID, permission string) error
	RemovePermission(ctx context.Context, roleID uuid.UUID, permission string) error
	GetPermissions(ctx context.Context, roleID uuid.UUID) ([]string, error)
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error)
	AssignToUser(ctx context.Context, roleID, userID uuid.UUID) error
	RemoveFromUser(ctx context.Context, roleID, userID uuid.UUID) error
}

type RolePGRepository struct {
	pool *pgxpool.Pool
}

var _ RoleRepository = (*RolePGRepository)(nil)

func NewRoleRepository(pool *pgxpool.Pool) *RolePGRepository {
	return &RolePGRepository{pool: pool}
}

func (r *RolePGRepository) Create(ctx context.Context, req CreateRoleRequest) (*Role, error) {
	role := &Role{
		ID:   uuid.Must(uuid.NewV7()),
		Name: req.Name,
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO roles (id, name, description) VALUES ($1, $2, $3)`,
		role.ID, role.Name, role.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}
	return role, nil
}

func (r *RolePGRepository) GetByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	role := &Role{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, created_at FROM roles WHERE id = $1`, id,
	).Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("get role: %w", err)
	}
	return role, nil
}

func (r *RolePGRepository) List(ctx context.Context) ([]Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, created_at FROM roles ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *RolePGRepository) Update(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*Role, error) {
	if req.Name != nil {
		_, err := r.pool.Exec(ctx, `UPDATE roles SET name = $1 WHERE id = $2`, *req.Name, id)
		if err != nil {
			return nil, fmt.Errorf("update role name: %w", err)
		}
	}
	if req.Description != nil {
		_, err := r.pool.Exec(ctx, `UPDATE roles SET description = $1 WHERE id = $2`, *req.Description, id)
		if err != nil {
			return nil, fmt.Errorf("update role description: %w", err)
		}
	}
	return r.GetByID(ctx, id)
}

func (r *RolePGRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	return err
}

func (r *RolePGRepository) AddPermission(ctx context.Context, roleID uuid.UUID, permission string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO role_permissions (role_id, permission) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		roleID, permission,
	)
	return err
}

func (r *RolePGRepository) RemovePermission(ctx context.Context, roleID uuid.UUID, permission string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM role_permissions WHERE role_id = $1 AND permission = $2`,
		roleID, permission,
	)
	return err
}

func (r *RolePGRepository) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT permission FROM role_permissions WHERE role_id = $1 ORDER BY permission`,
		roleID,
	)
	if err != nil {
		return nil, fmt.Errorf("get permissions: %w", err)
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

func (r *RolePGRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.name, r.description, r.created_at FROM roles r
		 JOIN user_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id = $1 ORDER BY r.name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *RolePGRepository) AssignToUser(ctx context.Context, roleID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_roles (role_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		roleID, userID,
	)
	return err
}

func (r *RolePGRepository) RemoveFromUser(ctx context.Context, roleID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_roles WHERE role_id = $1 AND user_id = $2`,
		roleID, userID,
	)
	return err
}
