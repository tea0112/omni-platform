package role

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
}

type CreateRoleRequest struct {
	Name        string
	Description string
}

type UpdateRoleRequest struct {
	Name        *string
	Description *string
}

type AddPermissionRequest struct {
	Permission string
}
