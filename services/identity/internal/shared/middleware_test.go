package shared_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tea0112/omni-platform/services/identity/internal/shared"
)

func TestAuthenticate_ValidToken(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	tokenSvc := shared.NewTokenService(priv, pub, 15*time.Minute)
	token, _, _ := tokenSvc.GenerateAccessToken("u1", []string{"admin"}, []string{"users.read"})

	mw := shared.Authenticate(tokenSvc)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := shared.GetPrincipal(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "u1", p.UserID)
		assert.Equal(t, []string{"admin"}, p.Roles)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthenticate_MissingHeader(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	tokenSvc := shared.NewTokenService(priv, pub, 15*time.Minute)

	mw := shared.Authenticate(tokenSvc)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	tokenSvc := shared.NewTokenService(priv, pub, 15*time.Minute)

	mw := shared.Authenticate(tokenSvc)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
