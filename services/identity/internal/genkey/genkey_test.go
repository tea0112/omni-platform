package genkey

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestGenerate_ReturnsValidJWK(t *testing.T) {
	s, err := Generate()
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	var jwk map[string]string
	if err := json.Unmarshal([]byte(s), &jwk); err != nil {
		t.Fatalf("Generate output is not valid JSON: %v\noutput: %s", err, s)
	}
	if jwk["kty"] != "OKP" {
		t.Errorf("kty = %q, want %q", jwk["kty"], "OKP")
	}
	if jwk["crv"] != "Ed25519" {
		t.Errorf("crv = %q, want %q", jwk["crv"], "Ed25519")
	}
	if _, err := base64.RawURLEncoding.DecodeString(jwk["d"]); err != nil {
		t.Errorf("d is not valid base64url: %v", err)
	}
	if _, err := base64.RawURLEncoding.DecodeString(jwk["x"]); err != nil {
		t.Errorf("x is not valid base64url: %v", err)
	}
	if jwk["d"] == "" || jwk["x"] == "" {
		t.Error("d and x must be non-empty")
	}
}

func TestGenerate_UniqueEachCall(t *testing.T) {
	s1, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if s1 == s2 {
		t.Error("two successive Generate calls produced the same JWK; expected unique keys")
	}
}
