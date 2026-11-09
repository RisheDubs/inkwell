package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer   = "https://cognito-idp.ap-southeast-2.amazonaws.com/ap-southeast-2_Test123"
	testClientID = "test-client"
	testKID      = "test-key-1"
)

// testKeys holds our "Cognito" signing key and an attacker's key.
type testKeys struct {
	pool     *rsa.PrivateKey
	attacker *rsa.PrivateKey
}

func newTestKeys(t *testing.T) testKeys {
	t.Helper()
	pool, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	attacker, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return testKeys{pool: pool, attacker: attacker}
}

// keyfunc plays the role of the JWKS: it only knows the pool's public key.
func (k testKeys) keyfunc(token *jwt.Token) (any, error) {
	if token.Header["kid"] != testKID {
		return nil, errors.New("unknown kid")
	}
	return &k.pool.PublicKey, nil
}

func validClaims() Claims {
	now := time.Now()
	return Claims{
		TokenUse: "access",
		ClientID: testClientID,
		Username: "rishe",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			Issuer:    testIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
}

func sign(t *testing.T, key *rsa.PrivateKey, claims Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKID
	s, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestVerifyAcceptsValidToken(t *testing.T) {
	keys := newTestKeys(t)
	v := NewVerifier(keys.keyfunc, testIssuer, testClientID)

	claims, err := v.Verify(sign(t, keys.pool, validClaims()))
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
	}
}

func TestVerifyRejectsBadTokens(t *testing.T) {
	keys := newTestKeys(t)
	v := NewVerifier(keys.keyfunc, testIssuer, testClientID)

	tests := []struct {
		name  string
		token func(t *testing.T) string
	}{
		{name: "empty", token: func(t *testing.T) string { return "" }},
		{name: "not a JWT", token: func(t *testing.T) string { return "hello" }},
		{name: "expired", token: func(t *testing.T) string {
			c := validClaims()
			c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
			return sign(t, keys.pool, c)
		}},
		{name: "no expiry", token: func(t *testing.T) string {
			c := validClaims()
			c.ExpiresAt = nil
			return sign(t, keys.pool, c)
		}},
		{name: "tampered payload", token: func(t *testing.T) string {
			good := sign(t, keys.pool, validClaims())
			other := validClaims()
			other.Subject = "someone-else"
			forged := sign(t, keys.pool, other)
			// Keep the original signature but swap in a different payload.
			g, f := strings.Split(good, "."), strings.Split(forged, ".")
			return g[0] + "." + f[1] + "." + g[2]
		}},
		{name: "signed by another key", token: func(t *testing.T) string {
			return sign(t, keys.attacker, validClaims())
		}},
		{name: "wrong issuer", token: func(t *testing.T) string {
			c := validClaims()
			c.Issuer = "https://evil.example.com"
			return sign(t, keys.pool, c)
		}},
		{name: "wrong app client", token: func(t *testing.T) string {
			c := validClaims()
			c.ClientID = "another-app"
			return sign(t, keys.pool, c)
		}},
		{name: "ID token instead of access token", token: func(t *testing.T) string {
			c := validClaims()
			c.TokenUse = "id"
			return sign(t, keys.pool, c)
		}},
		{name: "missing subject", token: func(t *testing.T) string {
			c := validClaims()
			c.Subject = ""
			return sign(t, keys.pool, c)
		}},
		{name: "alg none", token: func(t *testing.T) string {
			token := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims())
			token.Header["kid"] = testKID
			s, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
			if err != nil {
				t.Fatal(err)
			}
			return s
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := v.Verify(tc.token(t))
			if !errors.Is(err, ErrInvalidToken) {
				t.Errorf("Verify() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}
