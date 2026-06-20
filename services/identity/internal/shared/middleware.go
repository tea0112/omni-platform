package shared

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const principalKey contextKey = "principal"

type Principal struct {
	UserID      string
	Roles       []string
	Permissions []string
}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func GetPrincipal(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

func Authenticate(tokenSvc *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"code": "unauthenticated", "message": "missing authorization header"})
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := tokenSvc.ValidateAccessToken(token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"code": "unauthenticated", "message": "invalid token"})
				return
			}
			p := Principal{
				UserID:      claims.Subject,
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
			}
			ctx := WithPrincipal(r.Context(), p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
