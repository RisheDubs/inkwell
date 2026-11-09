// Package auth verifies Amazon Cognito access tokens and looks up signed-in users.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken wraps every reason a token is rejected.
var ErrInvalidToken = errors.New("invalid token")

// Claims are the parts of a Cognito access token the API relies on.
type Claims struct {
	TokenUse string `json:"token_use"`
	ClientID string `json:"client_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Verifier checks that a token was issued by our user pool, for our app client, and is still valid.
type Verifier struct {
	keyfunc  jwt.Keyfunc
	issuer   string
	clientID string
}

// NewVerifier builds a Verifier. keyfunc supplies the public key for each token's "kid".
func NewVerifier(keyfunc jwt.Keyfunc, issuer, clientID string) *Verifier {
	return &Verifier{keyfunc: keyfunc, issuer: issuer, clientID: clientID}
}

// NewCognitoVerifier downloads the user pool's public keys (its JWKS) and keeps them
// refreshed in the background until ctx is cancelled.
func NewCognitoVerifier(ctx context.Context, jwksURL, issuer, clientID string) (*Verifier, error) {
	k, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("load JWKS: %w", err)
	}
	return NewVerifier(k.Keyfunc, issuer, clientID), nil
}

// Verify parses tokenString and returns its claims only if every check passes.
func (v *Verifier) Verify(tokenString string) (Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(tokenString, &claims, v.keyfunc,
		jwt.WithValidMethods([]string{"RS256"}), // reject "none" and algorithm-swap tricks
		jwt.WithIssuer(v.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second), // tolerate small clock differences between servers
	)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	// Cognito also issues ID tokens; only access tokens may call the API.
	if claims.TokenUse != "access" {
		return Claims{}, fmt.Errorf("%w: token_use is %q, want \"access\"", ErrInvalidToken, claims.TokenUse)
	}
	// Access tokens carry client_id (not aud). Reject tokens minted for other apps.
	if claims.ClientID != v.clientID {
		return Claims{}, fmt.Errorf("%w: token was issued for a different app client", ErrInvalidToken)
	}
	if claims.Subject == "" {
		return Claims{}, fmt.Errorf("%w: missing sub", ErrInvalidToken)
	}
	return claims, nil
}
