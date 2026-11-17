//go:build integration

package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// Public journals through the real router, services and MongoDB. A gets the handle writer-a and
// B gets writer-b; each writes one entry. Every public request below is sent with no token.

const (
	handleA = "writer-a"
	handleB = "writer-b"
)

// newPublicEntryJSON is a minimal valid entry, the same shape the web app sends.
func newPublicEntryJSON(title string) string {
	return fmt.Sprintf(`{"title":%q,"entryDate":"2026-11-03","tags":[],"body":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}}`, title)
}

// setUpPublicUsers creates both users with handles and one entry each, and returns the entry IDs.
// Both journals are still private, as every new journal is.
func setUpPublicUsers(t *testing.T, h http.Handler) (idA, idB string) {
	t.Helper()
	for tok, handle := range map[string]string{tokenA: handleA, tokenB: handleB} {
		call(t, h, http.MethodGet, "/api/me", tok, "", nil) // first call creates the user and journal
		if code := call(t, h, http.MethodPatch, "/api/me", tok, fmt.Sprintf(`{"handle":%q}`, handle), nil); code != http.StatusOK {
			t.Fatalf("PATCH /api/me handle %s = %d", handle, code)
		}
	}
	ids := map[string]string{}
	for tok, title := range map[string]string{tokenA: "A's entry", tokenB: "B's entry"} {
		var created struct {
			ID string `json:"id"`
		}
		if code := call(t, h, http.MethodPost, "/api/entries", tok, newPublicEntryJSON(title), &created); code != http.StatusCreated || created.ID == "" {
			t.Fatalf("POST /api/entries for %s = %d, id %q", tok, code, created.ID)
		}
		ids[tok] = created.ID
	}
	return ids[tokenA], ids[tokenB]
}

func setVisibility(t *testing.T, h http.Handler, tok, v string) {
	t.Helper()
	if code := call(t, h, http.MethodPatch, "/api/journal", tok, fmt.Sprintf(`{"visibility":%q}`, v), nil); code != http.StatusOK {
		t.Fatalf("PATCH /api/journal visibility %s = %d", v, code)
	}
}

func TestNewJournalsAreNotPublic(t *testing.T) {
	h, _ := newIsolationServer(t)
	idA, _ := setUpPublicUsers(t, h)

	for _, path := range []string{"/api/public/" + handleA, "/api/public/" + handleA + "/entries/" + idA} {
		if code := call(t, h, http.MethodGet, path, "", "", nil); code != http.StatusNotFound {
			t.Errorf("GET %s before going public = %d, want 404", path, code)
		}
	}
}

func TestPublicJournalShowsOnlyItsOwnersEntries(t *testing.T) {
	h, _ := newIsolationServer(t)
	idA, idB := setUpPublicUsers(t, h)
	setVisibility(t, h, tokenA, "public")

	var pj journal.PublicJournal
	if code := call(t, h, http.MethodGet, "/api/public/"+handleA, "", "", &pj); code != http.StatusOK {
		t.Fatalf("GET A's public journal = %d", code)
	}
	if len(pj.Entries) != 1 || pj.Entries[0].ID != idA {
		t.Errorf("A's public journal lists %+v, want only entry %s", pj.Entries, idA)
	}

	// Handles are case-insensitive, like sign-up.
	if code := call(t, h, http.MethodGet, "/api/public/WRITER-A", "", "", nil); code != http.StatusOK {
		t.Errorf("GET /api/public/WRITER-A = %d, want 200", code)
	}

	cases := map[string]int{
		"/api/public/" + handleA + "/entries/" + idA: http.StatusOK,
		"/api/public/" + handleA + "/entries/" + idB: http.StatusNotFound, // B's entry through A's handle
		"/api/public/" + handleB:                     http.StatusNotFound, // B is still private
		"/api/public/" + handleB + "/entries/" + idB: http.StatusNotFound,
		"/api/public/nobody-here":                    http.StatusNotFound,
	}
	for path, want := range cases {
		if code := call(t, h, http.MethodGet, path, "", "", nil); code != want {
			t.Errorf("GET %s = %d, want %d", path, code, want)
		}
	}
}

func TestPublicResponsesLeakNothingPrivate(t *testing.T) {
	h, _ := newIsolationServer(t)
	idA, _ := setUpPublicUsers(t, h)
	setVisibility(t, h, tokenA, "public")

	for _, path := range []string{"/api/public/" + handleA, "/api/public/" + handleA + "/entries/" + idA} {
		var raw json.RawMessage
		if code := call(t, h, http.MethodGet, path, "", "", &raw); code != http.StatusOK {
			t.Fatalf("GET %s = %d", path, code)
		}
		for _, private := range []string{userA, "@example.com", "ownerId", "journalId", "bodyKey", "coverKey", "users/", "createdAt", "updatedAt"} {
			if strings.Contains(string(raw), private) {
				t.Errorf("GET %s contains %q: %s", path, private, raw)
			}
		}
	}
}

func TestPublicMediaLinksStayInTheOwnersFolder(t *testing.T) {
	h, _ := newIsolationServer(t)
	setUpPublicUsers(t, h)

	var up journal.Upload
	if code := call(t, h, http.MethodPost, "/api/media/uploads", tokenA, `{"contentType":"image/jpeg"}`, &up); code != http.StatusCreated {
		t.Fatalf("A's upload request = %d", code)
	}
	path := "/api/public/" + handleA + "/media/" + up.MediaID

	if code := call(t, h, http.MethodGet, path, "", "", nil); code != http.StatusNotFound {
		t.Errorf("GET %s while private = %d, want 404", path, code)
	}

	setVisibility(t, h, tokenA, "public")
	var link journal.MediaURL
	if code := call(t, h, http.MethodGet, path, "", "", &link); code != http.StatusOK {
		t.Fatalf("GET %s = %d", path, code)
	}
	if !strings.Contains(link.URL, "users/"+userA+"/") {
		t.Errorf("public link = %q, want it inside users/%s/", link.URL, userA)
	}

	// A's media ID through B's handle: B is private, so 404.
	if code := call(t, h, http.MethodGet, "/api/public/"+handleB+"/media/"+up.MediaID, "", "", nil); code != http.StatusNotFound {
		t.Errorf("A's media through B's handle = %d, want 404", code)
	}
}

func TestMakingAJournalPrivateHidesItAtOnce(t *testing.T) {
	h, _ := newIsolationServer(t)
	idA, _ := setUpPublicUsers(t, h)

	setVisibility(t, h, tokenA, "public")
	if code := call(t, h, http.MethodGet, "/api/public/"+handleA, "", "", nil); code != http.StatusOK {
		t.Fatalf("GET while public = %d, want 200", code)
	}

	setVisibility(t, h, tokenA, "private")
	for _, path := range []string{"/api/public/" + handleA, "/api/public/" + handleA + "/entries/" + idA} {
		if code := call(t, h, http.MethodGet, path, "", "", nil); code != http.StatusNotFound {
			t.Errorf("GET %s after going private = %d, want 404", path, code)
		}
	}
}
