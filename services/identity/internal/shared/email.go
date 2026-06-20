package shared

import "context"

type EmailSender interface {
	SendPasswordReset(ctx context.Context, to, token string) error
}
