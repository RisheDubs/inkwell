package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// fakeJournals records what it was asked and returns canned results.
type fakeJournals struct {
	gotOwnerID string
	getErr     error
	updateErr  error
}

func (f *fakeJournals) Journal(_ context.Context, ownerID string) (journal.Journal, error) {
	f.gotOwnerID = ownerID
	if f.getErr != nil {
		return journal.Journal{}, f.getErr
	}
	return journal.Journal{ID: "j1", OwnerID: ownerID, Title: "rishe's Journal", Visibility: "private",
		Theme: journal.Theme{Paper: "dotted", Font: "serif", Accent: "#2f5d50"}}, nil
}

func (f *fakeJournals) UpdateJournal(_ context.Context, ownerID string, in journal.UpdateJournalInput) (journal.Journal, error) {
	f.gotOwnerID = ownerID
	if f.updateErr != nil {
		return journal.Journal{}, f.updateErr
	}
	return journal.Journal{ID: "j1", OwnerID: ownerID, Title: "Morning pages"}, nil
}

func TestGetJournal(t *testing.T) {
	journals := &fakeJournals{}
	rec := do(t, newTestRouter(t, testDeps{journals: journals}), http.MethodGet, "/api/journal", goodToken, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if journals.gotOwnerID != "user-123" {
		t.Errorf("service got ownerID %q, want user-123 (from the token)", journals.gotOwnerID)
	}

	var j journal.Journal
	if err := json.NewDecoder(rec.Body).Decode(&j); err != nil {
		t.Fatal(err)
	}
	if j.ID != "j1" || j.Theme.Paper != "dotted" {
		t.Errorf("journal = %+v", j)
	}
}

func TestJournalNeedsAuth(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		rec := do(t, newTestRouter(t, testDeps{}), method, "/api/journal", "", `{"title":"x"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s status = %d, want %d", method, rec.Code, http.StatusUnauthorized)
		}
	}
}

func TestGetJournalNotFound(t *testing.T) {
	journals := &fakeJournals{getErr: journal.ErrNotFound}
	rec := do(t, newTestRouter(t, testDeps{journals: journals}), http.MethodGet, "/api/journal", goodToken, "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPatchJournal(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		updateErr error
		wantCode  int
		wantError string
	}{
		{name: "valid", body: `{"title":"Morning pages","theme":{"paper":"grid"}}`, wantCode: http.StatusOK},
		{name: "unknown top-level field", body: `{"ownerId":"someone-else"}`, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "unknown theme field", body: `{"theme":{"coverKey":"users/other/media/x.jpg"}}`, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "validation error", body: `{"visibility":"friends"}`, updateErr: &journal.ValidationError{Field: "visibility", Message: "bad"}, wantCode: http.StatusBadRequest, wantError: "invalid_request"},
		{name: "no journal yet", body: `{"title":"x"}`, updateErr: journal.ErrNotFound, wantCode: http.StatusNotFound, wantError: "not_found"},
		{name: "database failure", body: `{"title":"x"}`, updateErr: errors.New("boom"), wantCode: http.StatusInternalServerError, wantError: "internal"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			journals := &fakeJournals{updateErr: tc.updateErr}
			rec := do(t, newTestRouter(t, testDeps{journals: journals}), http.MethodPatch, "/api/journal", goodToken, tc.body)

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.wantCode, rec.Body.String())
			}
			if tc.wantError != "" {
				if code := decodeError(t, rec); code != tc.wantError {
					t.Errorf("error code = %q, want %q", code, tc.wantError)
				}
			}
		})
	}
}
