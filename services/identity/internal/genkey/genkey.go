package genkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrKeyGen = errors.New("genkey: key generation failed")

// JWK is the JSON Web Key representation of an Ed25519 key.
// Only the fields needed by the identity service are populated.
type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	D   string `json:"d"`
	X   string `json:"x"`
}

// Generate returns a freshly-generated Ed25519 key encoded as a JSON JWK string.
// The private seed (d) and the public key (x) are base64url-encoded without padding
// per RFC 7518 §6.2.1.1.
func Generate() (string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKeyGen, err)
	}
	j := JWK{
		Kty: "OKP",
		Crv: "Ed25519",
		D:   base64.RawURLEncoding.EncodeToString(priv.Seed()),
		X:   base64.RawURLEncoding.EncodeToString(pub),
	}
	b, err := json.Marshal(j)
	if err != nil {
		return "", fmt.Errorf("%w: marshal: %v", ErrKeyGen, err)
	}
	return string(b), nil
}
