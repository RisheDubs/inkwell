package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// UserInfoClient looks up a signed-in user's email with Cognito's GetUser API.
// GetUser is authorised by the user's own access token, so it needs no AWS credentials.
type UserInfoClient struct {
	endpoint string
	http     *http.Client
}

// NewUserInfoClient creates a client for the Cognito endpoint in the pool's region.
func NewUserInfoClient(endpoint string) *UserInfoClient {
	return &UserInfoClient{
		endpoint: endpoint,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

// Email returns the verified email address of the user who owns accessToken.
func (c *UserInfoClient) Email(ctx context.Context, accessToken string) (string, error) {
	payload, err := json.Marshal(map[string]string{"AccessToken": accessToken})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService.GetUser")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("cognito GetUser: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body := io.LimitReader(resp.Body, 64<<10) // never read an unbounded response
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Type string `json:"__type"`
		}
		_ = json.NewDecoder(body).Decode(&apiErr)
		return "", fmt.Errorf("cognito GetUser: status %d %s", resp.StatusCode, apiErr.Type)
	}

	var out struct {
		UserAttributes []struct {
			Name  string `json:"Name"`
			Value string `json:"Value"`
		} `json:"UserAttributes"`
	}
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		return "", fmt.Errorf("cognito GetUser: decode response: %w", err)
	}

	attrs := make(map[string]string, len(out.UserAttributes))
	for _, a := range out.UserAttributes {
		attrs[a.Name] = a.Value
	}
	if attrs["email"] == "" {
		return "", errors.New("cognito GetUser: user has no email attribute")
	}
	if attrs["email_verified"] != "true" {
		return "", errors.New("cognito GetUser: email is not verified")
	}
	return attrs["email"], nil
}
