package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// publicFake serves one public journal, "alice", with entry e1 and image c.png.
type publicFake struct {
	err      error // when set, every call fails with it
	calls    int
	gotList  journal.ListInput
	gotEntry string
	gotMedia string
}

func (f *publicFake) Journal(_ context.Context, handle string, in journal.ListInput) (journal.PublicJournal, error) {
	f.calls++
	f.gotList = in
	if f.err != nil {
		return journal.PublicJournal{}, f.err
	}
	if handle != "alice" {
		return journal.PublicJournal{}, journal.ErrNotFound
	}
	return journal.PublicJournal{Handle: "alice", Title: "Alice's Journal", Entries: []journal.PublicEntrySummary{}}, nil
}

func (f *publicFake) Entry(_ context.Context, handle, id string) (journal.PublicEntry, error) {
	f.calls++
	f.gotEntry = id
	if f.err != nil {
		return journal.PublicEntry{}, f.err
	}
	if handle != "alice" || id != "e1" {
		return journal.PublicEntry{}, journal.ErrNotFound
	}
	return journal.PublicEntry{PublicEntrySummary: journal.PublicEntrySummary{ID: "e1", Title: "Hello world"}}, nil
}

func (f *publicFake) MediaURL(_ context.Context, handle, mediaID string) (journal.MediaURL, error) {
	f.calls++
	f.gotMedia = mediaID
	if f.err != nil {
		return journal.MediaURL{}, f.err
	}
	if handle != "alice" || mediaID != "c.png" {
		return journal.MediaURL{}, journal.ErrNotFound
	}
	return journal.MediaURL{MediaID: mediaID, URL: "https://s3.example/" + mediaID}, nil
}

// publicRequest sends a request with no Authorization header at all.
func publicRequest(t *testing.T, f *publicFake, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewRouter(Deps{Public: f})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func publicErrorBody(t *testing.T, rec *httptest.ResponseRecorder) errorDetail {
	t.Helper()
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not JSON: %v: %s", err, rec.Body)
	}
	return body.Error
}

func TestPublicJournalNeedsNoToken(t *testing.T) {
	rec := publicRequest(t, &publicFake{}, http.MethodGet, "/api/public/alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "Alice's Journal") {
		t.Errorf("body = %s, want the journal title", rec.Body)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestPublicJournalPassesTheListQuery(t *testing.T) {
	f := &publicFake{}
	rec := publicRequest(t, f, http.MethodGet, "/api/public/alice?tag=travel&cursor=abc&limit=5")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if f.gotList.Tag != "travel" || f.gotList.Cursor != "abc" || f.gotList.Limit != 5 {
		t.Errorf("service got %+v, want tag travel, cursor abc, limit 5", f.gotList)
	}
}

func TestPublicJournalRejectsABadLimit(t *testing.T) {
	f := &publicFake{}
	rec := publicRequest(t, f, http.MethodGet, "/api/public/alice?limit=zero")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if f.calls != 0 {
		t.Errorf("service called %d times for a bad limit", f.calls)
	}
}

func TestPublicEntryAndMediaReachTheService(t *testing.T) {
	f := &publicFake{}
	if rec := publicRequest(t, f, http.MethodGet, "/api/public/alice/entries/e1"); rec.Code != http.StatusOK {
		t.Errorf("entry status = %d, want 200: %s", rec.Code, rec.Body)
	}
	rec := publicRequest(t, f, http.MethodGet, "/api/public/alice/media/c.png")
	if rec.Code != http.StatusOK {
		t.Errorf("media status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("media Cache-Control = %q, want no-store (it holds a signed link)", got)
	}
	if f.gotEntry != "e1" || f.gotMedia != "c.png" {
		t.Errorf("service got entry %q and media %q", f.gotEntry, f.gotMedia)
	}
}

// Every "you can't see this" must look exactly alike, or the differences tell an attacker which
// handles exist and which journals are private.
func TestPublicNotFoundLooksTheSameEverywhere(t *testing.T) {
	paths := []string{
		"/api/public/bob",
		"/api/public/a",
		"/api/public/bob/entries/e1",
		"/api/public/alice/entries/e2",
		"/api/public/bob/media/c.png",
		"/api/public/alice/media/x.png",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec := publicRequest(t, &publicFake{}, http.MethodGet, path)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rec.Code)
			}
			if got := publicErrorBody(t, rec); got.Code != "not_found" || got.Message != "not found" {
				t.Errorf("error = %+v, want code not_found and message \"not found\"", got)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store on errors too", got)
			}
		})
	}
}

func TestPublicInternalErrorsHideTheCause(t *testing.T) {
	f := &publicFake{err: errors.New("mongo: connection refused at 10.0.0.5")}
	rec := publicRequest(t, f, http.MethodGet, "/api/public/alice")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, "mongo") || strings.Contains(body, "10.0.0.5") {
		t.Errorf("body leaks the internal error: %s", body)
	}
}

func TestPublicRoutesAreReadOnly(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
		f := &publicFake{}
		rec := publicRequest(t, f, method, "/api/public/alice")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s status = %d, want 405", method, rec.Code)
		}
		if f.calls != 0 {
			t.Errorf("%s reached the service", method)
		}
	}
}

// The router split must not have made any private route public.
func TestPrivateRoutesStillNeedATokenAfterTheSplit(t *testing.T) {
	for _, path := range []string{"/api/me", "/api/journal", "/api/entries", "/api/entries/e1", "/api/media/c.png"} {
		rec := publicRequest(t, &publicFake{}, http.MethodGet, path)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s without a token: status = %d, want 401", path, rec.Code)
		}
	}
}
