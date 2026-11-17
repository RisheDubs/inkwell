package httpapi

import (
	"net/http"
	"testing"
)

// A wrong method on a private route must not reveal that the route exists: token first.
func TestWrongMethodOnAPrivateRouteNeedsATokenFirst(t *testing.T) {
	for _, path := range []string{"/api/me", "/api/journal", "/api/entries/e1"} {
		rec := publicRequest(t, &publicFake{}, http.MethodPut, path)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("PUT %s without a token: status = %d, want 401", path, rec.Code)
		}
	}
}

// A mistyped public link is the same plain 404 as a private journal, and never asks for a token.
func TestUnknownPublicPathIsThePublicNotFound(t *testing.T) {
	for _, path := range []string{"/api/public/alice/nope", "/api/public/alice/entries/e1/extra"} {
		t.Run(path, func(t *testing.T) {
			rec := publicRequest(t, &publicFake{}, http.MethodGet, path)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rec.Code)
			}
			if got := publicErrorBody(t, rec); got.Code != "not_found" || got.Message != "not found" {
				t.Errorf("error = %+v, want code not_found and message \"not found\"", got)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store", got)
			}
		})
	}
}
