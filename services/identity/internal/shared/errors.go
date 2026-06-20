package shared

import (
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrDuplicate       = errors.New("already exists")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrTokenExpired    = errors.New("token expired")
	ErrTokenRevoked    = errors.New("token revoked")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for k, v := range e.Fields {
		parts = append(parts, k+": "+v)
	}
	return "validation failed: " + strings.Join(parts, ", ")
}

func MapError(err error) (int, codes.Code, map[string]any) {
	var vErr *ValidationError
	switch {
	case errors.As(err, &vErr):
		return http.StatusUnprocessableEntity, codes.InvalidArgument, map[string]any{
			"code":    "validation_failed",
			"message": "validation failed",
			"details": map[string]any{"fields": fieldsToAny(vErr.Fields)},
		}
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, codes.NotFound, errBody("not_found", "not found")
	case errors.Is(err, ErrDuplicate):
		return http.StatusConflict, codes.AlreadyExists, errBody("already_exists", "already exists")
	case errors.Is(err, ErrUnauthenticated):
		return http.StatusUnauthorized, codes.Unauthenticated, errBody("unauthenticated", "unauthenticated")
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, codes.PermissionDenied, errBody("forbidden", "forbidden")
	case errors.Is(err, ErrTokenExpired):
		return http.StatusUnauthorized, codes.Unauthenticated, errBody("token_expired", "token expired")
	case errors.Is(err, ErrTokenRevoked):
		return http.StatusUnauthorized, codes.Unauthenticated, errBody("token_revoked", "token revoked")
	default:
		return http.StatusInternalServerError, codes.Internal, errBody("internal", "internal server error")
	}
}

func fieldsToAny(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func errBody(code, message string) map[string]any {
	return map[string]any{"code": code, "message": message}
}

func AsConnectError(err error) *connect.Error {
	_, grpcCode, _ := MapError(err)
	return connect.NewError(connect.Code(grpcCode), err)
}
