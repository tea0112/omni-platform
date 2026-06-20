package shared_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func TestLogEmailSender_SendPasswordReset(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	sender := shared.NewLogEmailSender(logger)

	err := sender.SendPasswordReset(context.Background(), "test@example.com", "abc123")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "password_reset")
	assert.Contains(t, output, "test@example.com")
	assert.Contains(t, output, "abc123")
}
