package journal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// publicFakes stands in for the store, EntryService and MediaService, and records who asked.
type publicFakes struct {
	users      map[string]User    // by handle
	journals   map[string]Journal // by owner ID
	entries    map[string]Entry   // by owner ID + "/" + entry ID
	asked      []string           // owner IDs passed to the entry and media services
	storeCalls int
}

func (f *publicFakes) FindUserByHandle(_ context.Context, handle string) (User, error) {
	f.storeCalls++
	u, ok := f.users[handle]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (f *publicFakes) FindJournal(_ context.Context, ownerID string) (Journal, error) {
	j, ok := f.journals[ownerID]
	if !ok {
		return Journal{}, ErrNotFound
	}
	return j, nil
}

func (f *publicFakes) ListEntries(_ context.Context, ownerID string, _ ListInput) (EntryPage, error) {
	f.asked = append(f.asked, ownerID)
	var page EntryPage
	for key, e := range f.entries {
		if strings.HasPrefix(key, ownerID+"/") {
			page.Entries = append(page.Entries, e)
		}
	}
	return page, nil
}

func (f *publicFakes) Entry(_ context.Context, ownerID, id string) (EntryWithBody, error) {
	f.asked = append(f.asked, ownerID)
	e, ok := f.entries[ownerID+"/"+id]
	if !ok {
		return EntryWithBody{}, ErrNotFound
	}
	return EntryWithBody{Entry: e, Body: json.RawMessage(`{"type":"doc"}`)}, nil
}

func (f *publicFakes) DownloadURL(_ context.Context, ownerID, mediaID string) (MediaURL, error) {
	f.asked = append(f.asked, ownerID)
	return MediaURL{MediaID: mediaID, URL: "https://s3.example/" + mediaID}, nil
}

// Two users: alice has a public journal, bob a private one.
func newPublicFakes() *publicFakes {
	alice, bob := "alice", "bob"
	now := time.Now().UTC()
	return &publicFakes{
		users: map[string]User{
			"alice": {ID: "u-alice", Email: "alice@example.com", Handle: &alice},
			"bob":   {ID: "u-bob", Email: "bob@example.com", Handle: &bob},
		},
		journals: map[string]Journal{
			"u-alice": {ID: "j-alice", OwnerID: "u-alice", Title: "Alice's Journal", Visibility: "public", Theme: Theme{Paper: "grid", CoverKey: "users/u-alice/media/c.png"}},
			"u-bob":   {ID: "j-bob", OwnerID: "u-bob", Title: "Bob's Journal", Visibility: "private"},
		},
		entries: map[string]Entry{
			"u-alice/e1": {ID: "e1", JournalID: "j-alice", OwnerID: "u-alice", Title: "Hello world", EntryDate: "2026-11-03", BodyKey: "users/u-alice/entries/e1.json", CreatedAt: now},
			"u-bob/e2":   {ID: "e2", JournalID: "j-bob", OwnerID: "u-bob", Title: "Bob's secret", EntryDate: "2026-11-04"},
		},
	}
}

func newTestPublicService(f *publicFakes) *PublicService { return NewPublicService(f, f, f) }

func TestPublicJournalShowsAPublicJournal(t *testing.T) {
	f := newPublicFakes()
	got, err := newTestPublicService(f).Journal(context.Background(), "Alice", ListInput{})
	if err != nil {
		t.Fatalf("Journal(alice) error = %v", err)
	}
	if got.Handle != "alice" || got.Title != "Alice's Journal" || len(got.Entries) != 1 || got.Entries[0].Title != "Hello world" {
		t.Fatalf("Journal(alice) = %+v", got)
	}
	if len(f.asked) != 1 || f.asked[0] != "u-alice" {
		t.Errorf("entry service asked for owners %v, want only u-alice", f.asked)
	}
}

func TestPublicJournalHidesPrivateUnknownAndMalformedAlike(t *testing.T) {
	for _, handle := range []string{"bob", "nobody-here", "../alice", "a", "alice/../bob"} {
		t.Run(handle, func(t *testing.T) {
			f := newPublicFakes()
			_, err := newTestPublicService(f).Journal(context.Background(), handle, ListInput{})
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("Journal(%q) error = %v, want ErrNotFound", handle, err)
			}
			if len(f.asked) != 0 {
				t.Errorf("entries were listed for %v although the journal is not public", f.asked)
			}
		})
	}
}

func TestPublicJournalNeverAsksTheDatabaseAboutImpossibleHandles(t *testing.T) {
	f := newPublicFakes()
	_, _ = newTestPublicService(f).Journal(context.Background(), "../../etc/passwd", ListInput{})
	if f.storeCalls != 0 {
		t.Errorf("FindUserByHandle called %d times for a malformed handle", f.storeCalls)
	}
}

func TestPublicEntryIsScopedToTheHandlesOwner(t *testing.T) {
	f := newPublicFakes()
	svc := newTestPublicService(f)

	if _, err := svc.Entry(context.Background(), "alice", "e1"); err != nil {
		t.Fatalf("Entry(alice, e1) error = %v", err)
	}
	// Bob's entry through Alice's public handle: looked up under Alice, so it isn't found.
	if _, err := svc.Entry(context.Background(), "alice", "e2"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Entry(alice, bob's e2) error = %v, want ErrNotFound", err)
	}
	// Bob's entry through Bob's handle: his journal is private.
	if _, err := svc.Entry(context.Background(), "bob", "e2"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Entry(bob, e2) error = %v, want ErrNotFound", err)
	}
	for _, owner := range f.asked {
		if owner != "u-alice" {
			t.Errorf("entry service asked for owner %q; only u-alice is public", owner)
		}
	}
}

func TestPublicMediaURLUsesTheHandlesOwnerAndOnlyWhenPublic(t *testing.T) {
	f := newPublicFakes()
	svc := newTestPublicService(f)

	if _, err := svc.MediaURL(context.Background(), "alice", "c.png"); err != nil {
		t.Fatalf("MediaURL(alice) error = %v", err)
	}
	if _, err := svc.MediaURL(context.Background(), "bob", "c.png"); !errors.Is(err, ErrNotFound) {
		t.Errorf("MediaURL(bob) error = %v, want ErrNotFound", err)
	}
	if len(f.asked) != 1 || f.asked[0] != "u-alice" {
		t.Errorf("media service asked for owners %v, want only u-alice", f.asked)
	}
}

func TestPublicResponsesNeverIncludePrivateFields(t *testing.T) {
	f := newPublicFakes()
	svc := newTestPublicService(f)

	j, err := svc.Journal(context.Background(), "alice", ListInput{})
	if err != nil {
		t.Fatalf("Journal error = %v", err)
	}
	e, err := svc.Entry(context.Background(), "alice", "e1")
	if err != nil {
		t.Fatalf("Entry error = %v", err)
	}
	for name, v := range map[string]any{"journal": j, "entry": e} {
		b, _ := json.Marshal(v)
		for _, private := range []string{"u-alice", "alice@example.com", "ownerId", "journalId", "j-alice", "bodyKey", "coverKey", "users/", "createdAt", "updatedAt"} {
			if strings.Contains(string(b), private) {
				t.Errorf("public %s JSON contains %q: %s", name, private, b)
			}
		}
	}
}

func TestPublicJournalListsEmptyEntriesAsAnArray(t *testing.T) {
	f := newPublicFakes()
	f.entries = map[string]Entry{}
	j, err := newTestPublicService(f).Journal(context.Background(), "alice", ListInput{})
	if err != nil {
		t.Fatalf("Journal error = %v", err)
	}
	b, _ := json.Marshal(j)
	if !strings.Contains(string(b), `"entries":[]`) {
		t.Errorf("JSON = %s, want \"entries\":[]", b)
	}
}
