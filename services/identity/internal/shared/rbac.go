package shared

import "context"

type RBAC struct{}

func NewRBAC() *RBAC {
	return &RBAC{}
}

func (r *RBAC) Can(ctx context.Context, action string, resource ...string) error {
	p, ok := GetPrincipal(ctx)
	if !ok {
		return ErrUnauthenticated
	}

	if len(resource) > 0 && resource[0] == p.UserID {
		profileAction := "profile." + actionSuffix(action)
		for _, perm := range p.Permissions {
			if perm == profileAction || perm == action {
				return nil
			}
		}
	}

	for _, perm := range p.Permissions {
		if perm == action {
			return nil
		}
	}

	return ErrForbidden
}

func actionSuffix(action string) string {
	for i, c := range action {
		if c == '.' {
			return action[i+1:]
		}
	}
	return action
}
