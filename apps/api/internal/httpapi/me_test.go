package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RisheDubs/inkwell/apps/api/internal/auth"
	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

const goodToken = "good-token"

// fakeVerifier accepts only goodToken, as user-123.
type fakeVerifier struct{}

func (fakeVerifier) Verify(token string) (auth.Claims, error) {
	if token != goodToken {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	c := auth.Claims{TokenUse: "access"}
	c.Subject = "user-123"
	return c, nil
}

// fakeMe records what it was asked and returns canned results.
type fakeMe struct {
	gotUserID   string
	gotToken    string
	gotUpdate   journal.UpdateMeInput
	meErr       error
	updateErr   error
	updateCalls int
}

func (f *fakeMe) Me(_ context.Context, userID, accessToken string) (journal.User, error) {
	f.gotUserID, f.gotToken = userID, accessToken
	if f.meErr != nil {
		return journal.User{}, f.meErr
	}
	return journal.User{ID: userID, Email: "rishe@example.com", DisplayName: "rishe"}, nil
}

func (f *fakeMe) UpdateMe(_ context.Context, userID string, in journal.UpdateMeInput) (journal.User, error) {
	f.updateCalls++
	f.gotUserID, f.gotUpdate = userID, in
	if f.updateErr != nil {
		return journal.User{}, f.updateErr
	}
	return journal.User{ID: userID, DisplayName: "Rishi"}, nil
}

type testDeps struct {
	ping     error
	me       *fakeMe
	journals *fakeJournals
	entries  *fakeEntries
}

func newTestRouter(t *testing.T, d testDeps) http.Handler {
	t.Helper()
	if d.me == nil {
		d.me = &fakeMe{}
	}
	if d.journals == nil {
		d.journals = &fakeJournals{}
	}
	if d.entries == nil {
		d.entries = &fakeEntries{}
	}
	return NewRouter(Deps{DB: fakePinger{err: d.ping}, Verifier: fakeVerifier{}, Me: d.me, Journals: d.journals, Entries: d.entries})
}

func do(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeError checks the standard error format and returns its code.
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.RequestID == "" {
		t.Error("error body has no requestId")
	}
	return body.Error.Code
}

func TestAPIRejectsMissingOrBadTokens(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{name: "no Authorization header", header: ""},
		{name: "wrong scheme", header: "Basic dXNlcjpwYXNz"},
		{name: "Bearer with no token", header: "Bearer "},
		{name: "token without scheme", header: goodToken},
		{name: "invalid token", header: "Bearer nope"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			me := &fakeMe{}
			router := newTestRouter(t, testDeps{me: me})

			req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if rec.Header().Get("WWW-Authenticate") == "" {
				t.Error("WWW-Authenticate header is missing")
			}
			if code := decodeError(t, rec); code != "unauthorized" {
				t.Errorf("error code = %q, want %q", code, "unauthorized")
			}
			if me.gotUserID != "" {
				t.Error("handler ran despite a bad token")
			}
		})
	}
}

func TestUnknownAPIRouteNeedsAuthFirst(t *testing.T) {
	// Without a token, unknown /api routes say 401 (not 404), so routes can't be probed.
	rec := do(t, newTestRouter(t, testDeps{}), http.MethodGet, "/api/secret", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetMe(t *testing.T) {
	me := &fakeMe{}
	rec := do(t, newTestRouter(t, testDeps{me: me}), http.MethodGet, "/api/me", goodToken, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	// The user ID must come from the verified token.
	if me.gotUserID != "user-123" || me.gotToken != goodToken {
		t.Errorf("service got userID=%q token=%q", me.gotUserID, me.gotToken)
	}

	var u journal.User
	if err := json.NewDecoder(rec.Body).Decode(&u); err != nil {
		t.Fatal(err)
	}
	if u.ID != "user-123" {
		t.Errorf("id = %q, want %q", u.ID, "user-123")
	}
}

func TestGetMeHidesInternalErrors(t *testing.T) {
	me := &fakeMe{meErr: errors.New("mongo: connection refused at 10.0.0.5")}
	rec := do(t, newTestRouter(t, testDeps{me: me}), http.MethodGet, "/api/me", goodToken, "")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Error("response leaks internal error details")
	}
}

func TestPatchMe(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		updateErr error
		wantCode  int
		wantError string
	}{
		{name: "valid update", body: `{"displayName":"Rishi"}`, wantCode: http.StatusOK},
		{name: "empty body", body: ``, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "not JSON", body: `hello`, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "unknown field", body: `{"displayName":"Rishi","isAdmin":true}`, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "trying to change the user ID", body: `{"id":"someone-else"}`, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "two JSON objects", body: `{"displayName":"A"}{"displayName":"B"}`, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "body too large", body: `{"displayName":"` + strings.Repeat("a", 5000) + `"}`, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "validation error", body: `{"handle":"ab"}`, updateErr: &journal.ValidationError{Field: "handle", Message: "too short"}, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "handle taken", body: `{"handle":"rishe"}`, updateErr: journal.ErrHandleTaken, wantCode: http.StatusConflict, wantError: "handle_taken"},
		{name: "user not created yet", body: `{"displayName":"Rishi"}`, updateErr: journal.ErrNotFound, wantCode: http.StatusNotFound, wantError: "not_found"},
		{name: "database failure", body: `{"displayName":"Rishi"}`, updateErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantError: "internal"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			me := &fakeMe{updateErr: tc.updateErr}
			rec := do(t, newTestRouter(t, testDeps{me: me}), http.MethodPatch, "/api/me", goodToken, tc.body)

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.wantCode, rec.Body.String())
			}
			if tc.wantError != "" {
				if code := decodeError(t, rec); code != tc.wantError {
					t.Errorf("error code = %q, want %q", code, tc.wantError)
				}
			}
			if tc.wantCode == http.StatusOK && me.gotUserID != "user-123" {
				t.Errorf("service got userID %q, want user-123", me.gotUserID)
			}
		})
	}
}
