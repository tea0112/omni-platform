package shared

import (
	"context"
	"fmt"
	"log/slog"
)

type LogEmailSender struct {
	logger *slog.Logger
}

func NewLogEmailSender(logger *slog.Logger) *LogEmailSender {
	return &LogEmailSender{logger: logger}
}

func (s *LogEmailSender) SendPasswordReset(ctx context.Context, to, token string) error {
	s.logger.InfoContext(ctx, "password_reset",
		"email", to,
		"reset_token", token,
		"reset_link", fmt.Sprintf("http://localhost:8080/reset-password?token=%s", token),
	)
	return nil
}
