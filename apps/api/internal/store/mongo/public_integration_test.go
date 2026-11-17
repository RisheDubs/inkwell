//go:build integration

package mongo

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// insertTestUser writes a user straight into the collection, so the test doesn't depend on how
// handles are set through UpdateUser.
func insertTestUser(t *testing.T, s *Store, u journal.User) {
	t.Helper()
	if _, err := s.db.Collection(usersCollection).InsertOne(context.Background(), u); err != nil {
		t.Fatalf("insert user %s: %v", u.ID, err)
	}
}

// uniqueSuffix keeps IDs and handles apart between runs, since handles must be unique.
func uniqueSuffix() string { return fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000) }

func TestFindUserByHandleFindsTheOwner(t *testing.T) {
	s := newTestStore(t)
	sfx := uniqueSuffix()
	handle := "alice-" + sfx
	insertTestUser(t, s, journal.User{ID: "u-alice-" + sfx, Email: "alice-" + sfx + "@example.com", Handle: &handle})

	got, err := s.FindUserByHandle(context.Background(), handle)
	if err != nil {
		t.Fatalf("FindUserByHandle(%q) error = %v", handle, err)
	}
	if got.ID != "u-alice-"+sfx {
		t.Errorf("FindUserByHandle(%q) ID = %q, want u-alice-%s", handle, got.ID, sfx)
	}
}

func TestFindUserByHandleUnknownIsNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.FindUserByHandle(context.Background(), "nobody-"+uniqueSuffix())
	if !errors.Is(err, journal.ErrNotFound) {
		t.Fatalf("error = %v, want journal.ErrNotFound", err)
	}
}

func TestFindUserByHandleIgnoresUsersWithoutAHandle(t *testing.T) {
	s := newTestStore(t)
	sfx := uniqueSuffix()
	insertTestUser(t, s, journal.User{ID: "u-nohandle-" + sfx, Email: "nohandle-" + sfx + "@example.com"})

	// A user with no handle must never be found, not even by an empty handle.
	if _, err := s.FindUserByHandle(context.Background(), ""); !errors.Is(err, journal.ErrNotFound) {
		t.Fatalf("FindUserByHandle(\"\") error = %v, want journal.ErrNotFound", err)
	}
}
