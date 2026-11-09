package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

// fakePinger stands in for MongoDB in tests. Ping returns err (nil means healthy).
type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error {
	return f.err
}

func TestHealthz(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantCode   int
		wantStatus string
	}{
		{name: "database reachable", pingErr: nil, wantCode: http.StatusOK, wantStatus: "ok"},
		{name: "database unreachable", pingErr: errors.New("connection refused"), wantCode: http.StatusServiceUnavailable, wantStatus: "unavailable"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter(t, testDeps{ping: tc.pingErr})
			rec := do(t, router, http.MethodGet, "/healthz", "", "")

			if rec.Code != tc.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want %q", got, "application/json")
			}

			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["status"] != tc.wantStatus {
				t.Errorf(`body["status"] = %q, want %q`, body["status"], tc.wantStatus)
			}
		})
	}
}

func TestHealthzNeedsNoToken(t *testing.T) {
	router := newTestRouter(t, testDeps{})
	rec := do(t, router, http.MethodGet, "/healthz", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestUnknownRouteReturns404(t *testing.T) {
	router := newTestRouter(t, testDeps{})
	rec := do(t, router, http.MethodGet, "/nope", "", "")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestResponsesIncludeRequestID(t *testing.T) {
	router := newTestRouter(t, testDeps{})
	rec := do(t, router, http.MethodGet, "/healthz", "", "")

	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("X-Request-Id header is missing")
	}
}
