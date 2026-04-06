// Package github provides GitHub API integration with JWT transport for App authentication.
package github

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTTransport implements http.RoundTripper for GitHub App JWT authentication.
// It signs each request with a JWT token generated from the App's private key.
type JWTTransport struct {
	appID      int64
	privateKey *rsa.PrivateKey
	underlying http.RoundTripper
}

// NewJWTTransport creates a new JWT transport for GitHub App authentication.
func NewJWTTransport(appID int64, privateKey *rsa.PrivateKey) *JWTTransport {
	return &JWTTransport{
		appID:      appID,
		privateKey: privateKey,
		underlying: http.DefaultTransport,
	}
}

// RoundTrip implements http.RoundTripper.
func (t *JWTTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Generate a new JWT for each request (they're short-lived)
	token, err := t.generateJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Clone the request and add Authorization header
	clonedReq := cloneRequest(req)
	clonedReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	clonedReq.Header.Set("Accept", "application/vnd.github+json")

	return t.underlying.RoundTrip(clonedReq)
}

// generateJWT creates a signed JWT for GitHub App authentication.
// The token is valid for 10 minutes (GitHub's maximum).
func (t *JWTTransport) generateJWT() (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"iat": now.Unix(),                        // Issued at
		"exp": now.Add(10 * time.Minute).Unix(), // Expires at (10 min max)
		"iss": t.appID,                          // Issuer (App ID)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(t.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, nil
}

// cloneRequest creates a shallow copy of the request.
func cloneRequest(req *http.Request) *http.Request {
	r := new(http.Request)
	*r = *req

	// Deep copy the header
	r.Header = make(http.Header, len(req.Header))
	for k, v := range req.Header {
		r.Header[k] = append([]string(nil), v...)
	}

	return r
}