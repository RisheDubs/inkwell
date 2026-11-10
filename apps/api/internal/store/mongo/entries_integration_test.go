//go:build integration

package mongo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// seedJournal creates ownerID's journal and returns its ID.
func seedJournal(t *testing.T, s *Store, ownerID string) string {
	t.Helper()
	ctx := context.Background()
	if err := s.EnsureJournal(ctx, journal.NewJournal(ownerID, ownerID, time.Now().UTC())); err != nil {
		t.Fatalf("EnsureJournal(%s) error = %v", ownerID, err)
	}
	j, err := s.FindJournal(ctx, ownerID)
	if err != nil {
		t.Fatalf("FindJournal(%s) error = %v", ownerID, err)
	}
	return j.ID
}

func newEntry(ownerID, journalID, title, date string, tags ...string) journal.Entry {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return journal.Entry{
		OwnerID: ownerID, JournalID: journalID, Title: title, EntryDate: date,
		Tags: tags, CreatedAt: now, UpdatedAt: now,
	}
}

func mustCreate(t *testing.T, s *Store, e journal.Entry) journal.Entry {
	t.Helper()
	created, err := s.CreateEntry(context.Background(), e)
	if err != nil {
		t.Fatalf("CreateEntry(%q) error = %v", e.Title, err)
	}
	return created
}

func TestJournalFindAndUpdate(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedJournal(t, s, "user-a")

	paper := "grid"
	j, err := s.UpdateJournal(ctx, "user-a", journal.UpdateJournalInput{Theme: &journal.ThemeInput{Paper: &paper}})
	if err != nil {
		t.Fatalf("UpdateJournal() error = %v", err)
	}
	// Only theme.paper changed; the rest of the theme is untouched.
	if j.Theme.Paper != "grid" || j.Theme.Font != "serif" || j.Theme.Accent != "#2f5d50" {
		t.Errorf("theme = %+v", j.Theme)
	}
	if j.ID == "" {
		t.Error("journal ID is empty")
	}

	if _, err := s.FindJournal(ctx, "user-b"); !errors.Is(err, journal.ErrNotFound) {
		t.Errorf("FindJournal(user-b) error = %v, want ErrNotFound", err)
	}
}

func TestListEntriesPaginatesNewestFirst(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	jid := seedJournal(t, s, "user-a")

	// Two entries share a date, so the _id tie-breaker is exercised.
	for _, e := range []journal.Entry{
		newEntry("user-a", jid, "one", "2026-11-01"),
		newEntry("user-a", jid, "two", "2026-11-02"),
		newEntry("user-a", jid, "three-a", "2026-11-03"),
		newEntry("user-a", jid, "three-b", "2026-11-03"),
		newEntry("user-a", jid, "four", "2026-11-04"),
	} {
		mustCreate(t, s, e)
	}

	var titles []string
	var cursor *journal.Cursor
	pages := 0
	for {
		page, err := s.ListEntries(ctx, "user-a", journal.ListQuery{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatalf("ListEntries() page %d error = %v", pages+1, err)
		}
		pages++
		for _, e := range page.Entries {
			titles = append(titles, e.Title)
		}
		if page.NextCursor == "" {
			break
		}
		c, err := journal.DecodeCursor(page.NextCursor)
		if err != nil {
			t.Fatalf("DecodeCursor() error = %v", err)
		}
		cursor = &c
		if pages > 5 {
			t.Fatal("pagination did not terminate")
		}
	}

	if pages != 3 || len(titles) != 5 {
		t.Fatalf("got %d pages and %d entries (%v), want 3 pages and 5 entries", pages, len(titles), titles)
	}
	// Newest first; the two same-day entries in reverse creation order (larger _id first).
	want := []string{"four", "three-b", "three-a", "two", "one"}
	for i := range want {
		if titles[i] != want[i] {
			t.Errorf("titles = %v, want %v", titles, want)
			break
		}
	}
}

func TestListEntriesFiltersByTag(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	jid := seedJournal(t, s, "user-a")
	mustCreate(t, s, newEntry("user-a", jid, "work day", "2026-11-01", "work"))
	mustCreate(t, s, newEntry("user-a", jid, "beach", "2026-11-02", "holiday"))

	page, err := s.ListEntries(ctx, "user-a", journal.ListQuery{Tag: "work"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 1 || page.Entries[0].Title != "work day" {
		t.Errorf("entries = %+v, want only \"work day\"", page.Entries)
	}
}

// TestEntryIsolation is the Phase 3 "done when" at the repository level:
// user B can't read, list, edit or delete user A's entries.
func TestEntryIsolation(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	jidA := seedJournal(t, s, "user-a")
	jidB := seedJournal(t, s, "user-b")
	aEntry := mustCreate(t, s, newEntry("user-a", jidA, "A's secret", "2026-11-05", "private"))
	mustCreate(t, s, newEntry("user-b", jidB, "B's entry", "2026-11-05", "private"))

	t.Run("B can't read A's entry", func(t *testing.T) {
		if _, err := s.FindEntry(ctx, "user-b", aEntry.ID); !errors.Is(err, journal.ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})

	t.Run("B's list never contains A's entry", func(t *testing.T) {
		for _, q := range []journal.ListQuery{{}, {Tag: "private"}} {
			page, err := s.ListEntries(ctx, "user-b", q)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range page.Entries {
				if e.ID == aEntry.ID || e.OwnerID != "user-b" {
					t.Errorf("B's list (tag %q) contains %+v", q.Tag, e)
				}
			}
		}
	})

	t.Run("B can't edit A's entry", func(t *testing.T) {
		title := "hacked"
		if _, err := s.UpdateEntry(ctx, "user-b", aEntry.ID, journal.EntryUpdate{Title: &title}, now); !errors.Is(err, journal.ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})

	t.Run("B can't delete A's entry", func(t *testing.T) {
		if err := s.DeleteEntry(ctx, "user-b", aEntry.ID, now); !errors.Is(err, journal.ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})

	t.Run("A's entry is untouched", func(t *testing.T) {
		got, err := s.FindEntry(ctx, "user-a", aEntry.ID)
		if err != nil {
			t.Fatalf("FindEntry(user-a) error = %v", err)
		}
		if got.Title != "A's secret" {
			t.Errorf("title = %q, want %q", got.Title, "A's secret")
		}
	})

	t.Run("a malformed ID looks like any missing entry", func(t *testing.T) {
		if _, err := s.FindEntry(ctx, "user-a", "not-an-objectid"); !errors.Is(err, journal.ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
}

func TestDeletedEntriesDisappear(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	jid := seedJournal(t, s, "user-a")
	e := mustCreate(t, s, newEntry("user-a", jid, "regret", "2026-11-06"))

	if err := s.DeleteEntry(ctx, "user-a", e.ID, time.Now().UTC()); err != nil {
		t.Fatalf("DeleteEntry() error = %v", err)
	}
	if _, err := s.FindEntry(ctx, "user-a", e.ID); !errors.Is(err, journal.ErrNotFound) {
		t.Errorf("FindEntry() after delete error = %v, want ErrNotFound", err)
	}
	page, err := s.ListEntries(ctx, "user-a", journal.ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 0 {
		t.Errorf("list after delete = %+v, want empty", page.Entries)
	}
	// Deleting twice is "not found": there is no live entry left to delete.
	if err := s.DeleteEntry(ctx, "user-a", e.ID, time.Now().UTC()); !errors.Is(err, journal.ErrNotFound) {
		t.Errorf("second DeleteEntry() error = %v, want ErrNotFound", err)
	}
}
