package shared_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func TestMapError_NotFound(t *testing.T) {
	status, code, body := shared.MapError(shared.ErrNotFound)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, codes.NotFound, code)
	assert.Equal(t, "not_found", body["code"])
}

func TestMapError_Validation(t *testing.T) {
	vErr := &shared.ValidationError{Fields: map[string]string{"email": "required"}}
	status, code, body := shared.MapError(vErr)
	assert.Equal(t, http.StatusUnprocessableEntity, status)
	assert.Equal(t, codes.InvalidArgument, code)
	assert.Equal(t, "validation_failed", body["code"])
	details := body["details"].(map[string]any)
	assert.Equal(t, map[string]any{"email": "required"}, details["fields"])
}

func TestMapError_Unknown(t *testing.T) {
	status, code, _ := shared.MapError(errors.New("unknown error"))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, codes.Internal, code)
}
