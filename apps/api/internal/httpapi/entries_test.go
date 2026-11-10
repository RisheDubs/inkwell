package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// fakeEntries records what it was asked and returns canned results.
type fakeEntries struct {
	gotOwnerID string
	gotID      string
	gotCreate  journal.CreateEntryInput
	gotList    journal.ListInput
	err        error
}

func cannedEntry() journal.EntryWithBody {
	return journal.EntryWithBody{
		Entry: journal.Entry{ID: "e1", OwnerID: "user-123", Title: "Hello", EntryDate: "2026-11-10",
			Tags: []string{}, BodyKey: "users/user-123/entries/e1.json"},
		Body: json.RawMessage(`{"type":"doc","content":[]}`),
	}
}

func (f *fakeEntries) CreateEntry(_ context.Context, ownerID string, in journal.CreateEntryInput) (journal.EntryWithBody, error) {
	f.gotOwnerID, f.gotCreate = ownerID, in
	if f.err != nil {
		return journal.EntryWithBody{}, f.err
	}
	return cannedEntry(), nil
}

func (f *fakeEntries) Entry(_ context.Context, ownerID, id string) (journal.EntryWithBody, error) {
	f.gotOwnerID, f.gotID = ownerID, id
	if f.err != nil {
		return journal.EntryWithBody{}, f.err
	}
	return cannedEntry(), nil
}

func (f *fakeEntries) ListEntries(_ context.Context, ownerID string, in journal.ListInput) (journal.EntryPage, error) {
	f.gotOwnerID, f.gotList = ownerID, in
	if f.err != nil {
		return journal.EntryPage{}, f.err
	}
	return journal.EntryPage{Entries: []journal.Entry{cannedEntry().Entry}}, nil
}

func (f *fakeEntries) UpdateEntry(_ context.Context, ownerID, id string, _ journal.UpdateEntryInput) (journal.EntryWithBody, error) {
	f.gotOwnerID, f.gotID = ownerID, id
	if f.err != nil {
		return journal.EntryWithBody{}, f.err
	}
	return cannedEntry(), nil
}

func (f *fakeEntries) DeleteEntry(_ context.Context, ownerID, id string) error {
	f.gotOwnerID, f.gotID = ownerID, id
	return f.err
}

const validEntryJSON = `{"title":"Hello","entryDate":"2026-11-10","tags":["work"],"body":{"type":"doc","content":[]}}`

// errorCode returns the "code" from an error response body.
func errorCode(t *testing.T, body string) string {
	t.Helper()
	var got struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("error body %q is not JSON: %v", body, err)
	}
	return got.Error.Code
}

func TestEntriesNeedAuth(t *testing.T) {
	router := newTestRouter(t, testDeps{})
	for _, rt := range []struct{ method, path string }{
		{http.MethodGet, "/api/entries"},
		{http.MethodPost, "/api/entries"},
		{http.MethodGet, "/api/entries/e1"},
		{http.MethodPatch, "/api/entries/e1"},
		{http.MethodDelete, "/api/entries/e1"},
	} {
		rec := do(t, router, rt.method, rt.path, "", validEntryJSON)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want %d", rt.method, rt.path, rec.Code, http.StatusUnauthorized)
		}
	}
}

func TestCreateEntry(t *testing.T) {
	entries := &fakeEntries{}
	rec := do(t, newTestRouter(t, testDeps{entries: entries}), http.MethodPost, "/api/entries", goodToken, validEntryJSON)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusCreated, rec.Body)
	}
	if loc := rec.Header().Get("Location"); loc != "/api/entries/e1" {
		t.Errorf("Location = %q, want /api/entries/e1", loc)
	}
	if entries.gotOwnerID != "user-123" {
		t.Errorf("service got ownerID %q, want user-123 (from the token)", entries.gotOwnerID)
	}
	if entries.gotCreate.Title != "Hello" || !strings.Contains(string(entries.gotCreate.Body), `"type":"doc"`) {
		t.Errorf("service got %+v", entries.gotCreate)
	}

	body := rec.Body.String()
	var got map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if _, ok := got["body"]; !ok {
		t.Errorf("response has no body field: %s", body)
	}
	for _, internal := range []string{"bodyKey", "ownerId", "users/"} {
		if strings.Contains(body, internal) {
			t.Errorf("response leaks %q: %s", internal, body)
		}
	}
}

func TestCreateEntryErrors(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		err      error
		wantCode int
		wantErr  string
	}{
		{name: "owner in the body", body: `{"ownerId":"someone-else","entryDate":"2026-11-10","body":{"type":"doc"}}`, wantCode: http.StatusBadRequest, wantErr: "invalid_request"},
		{name: "body key in the body", body: `{"bodyKey":"users/other/entries/x.json","entryDate":"2026-11-10"}`, wantCode: http.StatusBadRequest, wantErr: "invalid_request"},
		{name: "too large", body: `{"title":"` + strings.Repeat("a", 300<<10) + `"}`, wantCode: http.StatusBadRequest, wantErr: "invalid_request"},
		{name: "validation error", body: validEntryJSON, err: &journal.ValidationError{Field: "mood", Message: "bad"}, wantCode: http.StatusBadRequest, wantErr: "invalid_request"},
		{name: "no journal yet", body: validEntryJSON, err: journal.ErrNotFound, wantCode: http.StatusNotFound, wantErr: "not_found"},
		{name: "S3 failure", body: validEntryJSON, err: errors.New("AccessDenied: arn:aws:s3:::secret-bucket"), wantCode: http.StatusInternalServerError, wantErr: "internal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, newTestRouter(t, testDeps{entries: &fakeEntries{err: tt.err}}), http.MethodPost, "/api/entries", goodToken, tt.body)
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantCode, rec.Body)
			}
			if code := errorCode(t, rec.Body.String()); code != tt.wantErr {
				t.Errorf("error code = %q, want %q", code, tt.wantErr)
			}
			if strings.Contains(rec.Body.String(), "secret-bucket") {
				t.Errorf("response leaks the internal error: %s", rec.Body)
			}
		})
	}
}

func TestGetEntry(t *testing.T) {
	entries := &fakeEntries{}
	rec := do(t, newTestRouter(t, testDeps{entries: entries}), http.MethodGet, "/api/entries/abc123", goodToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if entries.gotID != "abc123" || entries.gotOwnerID != "user-123" {
		t.Errorf("service got id %q owner %q, want abc123 and user-123", entries.gotID, entries.gotOwnerID)
	}
}

func TestGetEntryNotFound(t *testing.T) {
	rec := do(t, newTestRouter(t, testDeps{entries: &fakeEntries{err: journal.ErrNotFound}}), http.MethodGet, "/api/entries/someone-elses", goodToken, "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (never 403)", rec.Code, http.StatusNotFound)
	}
}

func TestListEntriesQuery(t *testing.T) {
	entries := &fakeEntries{}
	rec := do(t, newTestRouter(t, testDeps{entries: entries}), http.MethodGet, "/api/entries?limit=10&tag=work&cursor=abc", goodToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	want := journal.ListInput{Limit: 10, Tag: "work", Cursor: "abc"}
	if entries.gotList != want {
		t.Errorf("service got %+v, want %+v", entries.gotList, want)
	}
	if strings.Contains(rec.Body.String(), `"body"`) {
		t.Errorf("list responses must not include bodies: %s", rec.Body)
	}
}

func TestListEntriesRejectsBadLimit(t *testing.T) {
	for _, limit := range []string{"abc", "0", "-5"} {
		entries := &fakeEntries{}
		rec := do(t, newTestRouter(t, testDeps{entries: entries}), http.MethodGet, "/api/entries?limit="+limit, goodToken, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("limit=%s: status = %d, want %d", limit, rec.Code, http.StatusBadRequest)
		}
		if entries.gotOwnerID != "" {
			t.Errorf("limit=%s: service was called; bad input must stop at the handler", limit)
		}
	}
}

func TestPatchEntry(t *testing.T) {
	entries := &fakeEntries{}
	rec := do(t, newTestRouter(t, testDeps{entries: entries}), http.MethodPatch, "/api/entries/e1", goodToken, `{"mood":"calm"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if entries.gotID != "e1" {
		t.Errorf("service got id %q, want e1", entries.gotID)
	}

	rec = do(t, newTestRouter(t, testDeps{entries: &fakeEntries{}}), http.MethodPatch, "/api/entries/e1", goodToken, `{"deletedAt":null}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("unknown field: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeleteEntry(t *testing.T) {
	entries := &fakeEntries{}
	rec := do(t, newTestRouter(t, testDeps{entries: entries}), http.MethodDelete, "/api/entries/e1", goodToken, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("204 must have no body, got %q", rec.Body)
	}
	if entries.gotID != "e1" || entries.gotOwnerID != "user-123" {
		t.Errorf("service got id %q owner %q", entries.gotID, entries.gotOwnerID)
	}

	rec = do(t, newTestRouter(t, testDeps{entries: &fakeEntries{err: journal.ErrNotFound}}), http.MethodDelete, "/api/entries/e1", goodToken, "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("missing entry: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
