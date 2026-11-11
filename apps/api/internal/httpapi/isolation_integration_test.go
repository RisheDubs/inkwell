//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/RisheDubs/inkwell/apps/api/internal/auth"
	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
	mongostore "github.com/RisheDubs/inkwell/apps/api/internal/store/mongo"
)

// These tests run the real router, real services and real MongoDB with two users, A and B,
// and check that B can't read, change or delete anything of A's through any route.
// Only Cognito and S3 are replaced: tokens map straight to users, and bodies live in memory.

const (
	tokenA = "token-user-a"
	tokenB = "token-user-b"
	userA  = "user-a"
	userB  = "user-b"
)

// twoUsers accepts tokenA as user A and tokenB as user B.
type twoUsers struct{}

func (twoUsers) Verify(token string) (auth.Claims, error) {
	subs := map[string]string{tokenA: userA, tokenB: userB}
	sub, ok := subs[token]
	if !ok {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	c := auth.Claims{TokenUse: "access"}
	c.Subject = sub
	return c, nil
}

// emailsByToken plays Cognito's GetUser.
type emailsByToken struct{}

func (emailsByToken) Email(_ context.Context, token string) (string, error) {
	return strings.TrimPrefix(token, "token-") + "@example.com", nil
}

// memBodies is an in-memory S3 for entry bodies.
type memBodies struct {
	mu sync.Mutex
	m  map[string][]byte
}

func (b *memBodies) PutBody(_ context.Context, key string, body []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.m[key] = append([]byte(nil), body...)
	return nil
}

func (b *memBodies) GetBody(_ context.Context, key string) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	body, ok := b.m[key]
	if !ok {
		return nil, journal.ErrNotFound
	}
	return body, nil
}

// keyPresigner returns the S3 key inside each "link", so tests can see which folder it points at.
type keyPresigner struct{}

func (keyPresigner) PresignUpload(_ context.Context, key, contentType string, _ int64, _ time.Duration) (string, map[string]string, error) {
	return "https://bucket.example/", map[string]string{"key": key, "Content-Type": contentType}, nil
}

func (keyPresigner) PresignDownload(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://bucket.example/" + key, nil
}

// newIsolationServer wires the real router to a fresh, randomly named database.
func newIsolationServer(t *testing.T) (http.Handler, *memBodies) {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI not set")
	}
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	dbName := "inkwell_it_http_" + hex.EncodeToString(suffix)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store, err := mongostore.Connect(ctx, uri, dbName)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := store.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if client, err := mongodriver.Connect(options.Client().ApplyURI(uri)); err == nil {
			_ = client.Database(dbName).Drop(ctx)
			_ = client.Disconnect(ctx)
		}
		_ = store.Close(ctx)
	})

	bodies := &memBodies{m: map[string][]byte{}}
	svc := journal.NewService(store, emailsByToken{})
	return NewRouter(Deps{
		DB:       store,
		Verifier: twoUsers{},
		Me:       svc,
		Journals: svc,
		Entries:  journal.NewEntryService(store, store, bodies, mongostore.NewID),
		Media:    journal.NewMediaService(keyPresigner{}, mongostore.NewID),
	}), bodies
}

// call sends one request and decodes a JSON response into out (if out isn't nil).
func call(t *testing.T, h http.Handler, method, path, token, body string, out any) int {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if out != nil && rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("%s %s: response is not JSON: %v (%s)", method, path, err, rec.Body)
		}
	}
	return rec.Code
}

const bodyA = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"A's private thoughts"}]}]}`

func TestUsersCannotTouchEachOthersEntries(t *testing.T) {
	h, bodies := newIsolationServer(t)

	// Both users sign in once, which creates their user and journal.
	for _, tok := range []string{tokenA, tokenB} {
		if code := call(t, h, http.MethodGet, "/api/me", tok, "", nil); code != http.StatusOK {
			t.Fatalf("GET /api/me (%s) = %d", tok, code)
		}
	}

	// A writes an entry.
	var created journal.EntryWithBody
	code := call(t, h, http.MethodPost, "/api/entries", tokenA,
		`{"title":"Private","entryDate":"2026-11-10","tags":["secret"],"body":`+bodyA+`}`, &created)
	if code != http.StatusCreated || created.ID == "" {
		t.Fatalf("A's POST /api/entries = %d, id %q", code, created.ID)
	}
	entryPath := "/api/entries/" + created.ID
	keyA := journal.BodyKey(userA, created.ID)

	// B tries every route on A's entry. Each must look exactly like "no such entry".
	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"read", http.MethodGet, entryPath, "", http.StatusNotFound},
		{"edit title", http.MethodPatch, entryPath, `{"title":"Hacked"}`, http.StatusNotFound},
		{"overwrite body", http.MethodPatch, entryPath, `{"body":{"type":"doc","content":[]}}`, http.StatusNotFound},
		{"delete", http.MethodDelete, entryPath, "", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run("B cannot "+tt.name, func(t *testing.T) {
			if code := call(t, h, tt.method, tt.path, tokenB, tt.body, nil); code != tt.want {
				t.Errorf("%s %s as B = %d, want %d", tt.method, tt.path, code, tt.want)
			}
		})
	}

	t.Run("B's list and tag filter show nothing of A's", func(t *testing.T) {
		for _, path := range []string{"/api/entries", "/api/entries?tag=secret"} {
			var page journal.EntryPage
			if code := call(t, h, http.MethodGet, path, tokenB, "", &page); code != http.StatusOK || len(page.Entries) != 0 {
				t.Errorf("GET %s as B = %d with %d entries, want 200 and 0", path, code, len(page.Entries))
			}
		}
	})

	t.Run("A's entry and body are unchanged", func(t *testing.T) {
		var got journal.EntryWithBody
		if code := call(t, h, http.MethodGet, entryPath, tokenA, "", &got); code != http.StatusOK {
			t.Fatalf("A's GET = %d, want 200", code)
		}
		if got.Title != "Private" {
			t.Errorf("title = %q, want %q", got.Title, "Private")
		}
		stored, _ := bodies.GetBody(context.Background(), keyA)
		if !bytes.Contains(stored, []byte("A's private thoughts")) {
			t.Errorf("A's stored body changed: %s", stored)
		}
	})

	t.Run("no S3 object was written outside A's folder for A's entry", func(t *testing.T) {
		bodies.mu.Lock()
		defer bodies.mu.Unlock()
		for key := range bodies.m {
			if strings.Contains(key, created.ID) && key != keyA {
				t.Errorf("unexpected body key %q", key)
			}
		}
	})
}

func TestUsersHaveSeparateJournals(t *testing.T) {
	h, _ := newIsolationServer(t)
	for _, tok := range []string{tokenA, tokenB} {
		call(t, h, http.MethodGet, "/api/me", tok, "", nil)
	}

	if code := call(t, h, http.MethodPatch, "/api/journal", tokenB, `{"title":"B's notebook","theme":{"paper":"grid"}}`, nil); code != http.StatusOK {
		t.Fatalf("B's PATCH /api/journal = %d", code)
	}

	var a journal.Journal
	call(t, h, http.MethodGet, "/api/journal", tokenA, "", &a)
	if a.Title == "B's notebook" || a.Theme.Paper == "grid" {
		t.Errorf("B's change reached A's journal: %+v", a)
	}
}

func TestMediaLinksStayInTheCallersFolder(t *testing.T) {
	h, _ := newIsolationServer(t)

	var up journal.Upload
	if code := call(t, h, http.MethodPost, "/api/media/uploads", tokenA, `{"contentType":"image/jpeg"}`, &up); code != http.StatusCreated {
		t.Fatalf("A's upload request = %d", code)
	}
	if want := "users/" + userA + "/media/"; !strings.HasPrefix(up.Fields["key"], want) {
		t.Errorf("A's upload key = %q, want it under %s", up.Fields["key"], want)
	}

	// B asks for A's exact media ID: the link is built from B's own ID, so it points at B's folder.
	var link journal.MediaURL
	if code := call(t, h, http.MethodGet, "/api/media/"+up.MediaID, tokenB, "", &link); code != http.StatusOK {
		t.Fatalf("B's GET media = %d", code)
	}
	if strings.Contains(link.URL, "users/"+userA+"/") || !strings.Contains(link.URL, "users/"+userB+"/") {
		t.Errorf("B got a link into %q; it must only ever point inside users/%s/", link.URL, userB)
	}
}

func TestEveryRouteNeedsAToken(t *testing.T) {
	h, _ := newIsolationServer(t)
	for _, rt := range []struct{ method, path string }{
		{http.MethodGet, "/api/me"}, {http.MethodPatch, "/api/me"},
		{http.MethodGet, "/api/journal"}, {http.MethodPatch, "/api/journal"},
		{http.MethodGet, "/api/entries"}, {http.MethodPost, "/api/entries"},
		{http.MethodGet, "/api/entries/x"}, {http.MethodPatch, "/api/entries/x"}, {http.MethodDelete, "/api/entries/x"},
		{http.MethodPost, "/api/media/uploads"}, {http.MethodGet, "/api/media/x.png"},
		{http.MethodGet, "/api/not-a-route"},
	} {
		for _, tok := range []string{"", "forged-token"} {
			if code := call(t, h, rt.method, rt.path, tok, "{}", nil); code != http.StatusUnauthorized {
				t.Errorf("%s %s with token %q = %d, want 401", rt.method, rt.path, tok, code)
			}
		}
	}
}
