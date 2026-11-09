package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeCognito stands in for Cognito's GetUser endpoint.
func fakeCognito(t *testing.T, status int, response string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Amz-Target"); got != "AWSCognitoIdentityProviderService.GetUser" {
			t.Errorf("X-Amz-Target = %q", got)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["AccessToken"] != "token-abc" {
			t.Errorf("request body = %v (err %v), want AccessToken token-abc", body, err)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestEmailReturnsVerifiedEmail(t *testing.T) {
	srv := fakeCognito(t, http.StatusOK, `{"Username":"abc","UserAttributes":[
		{"Name":"sub","Value":"user-123"},
		{"Name":"email","Value":"rishe@example.com"},
		{"Name":"email_verified","Value":"true"}]}`)

	email, err := NewUserInfoClient(srv.URL).Email(context.Background(), "token-abc")
	if err != nil {
		t.Fatalf("Email() error = %v", err)
	}
	if email != "rishe@example.com" {
		t.Errorf("Email() = %q, want %q", email, "rishe@example.com")
	}
}

func TestEmailErrors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		response string
	}{
		{name: "cognito rejects the token", status: http.StatusBadRequest, response: `{"__type":"NotAuthorizedException"}`},
		{name: "no email attribute", status: http.StatusOK, response: `{"UserAttributes":[{"Name":"sub","Value":"x"}]}`},
		{name: "email not verified", status: http.StatusOK, response: `{"UserAttributes":[{"Name":"email","Value":"a@b.c"},{"Name":"email_verified","Value":"false"}]}`},
		{name: "invalid JSON", status: http.StatusOK, response: `not json`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := fakeCognito(t, tc.status, tc.response)
			if _, err := NewUserInfoClient(srv.URL).Email(context.Background(), "token-abc"); err == nil {
				t.Error("Email(): expected an error, got nil")
			}
		})
	}
}
