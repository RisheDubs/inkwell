//go:build integration

// Integration tests talk to a real MongoDB. Run them with:
//
//	MONGODB_URI=... go test -tags integration ./internal/store/mongo
package mongo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// newTestStore connects to a fresh, randomly named database and drops it afterwards.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI not set")
	}

	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	dbName := "inkwell_it_" + hex.EncodeToString(suffix)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := Connect(ctx, uri, dbName)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := s.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.db.Drop(ctx)
		_ = s.Close(ctx)
	})
	return s
}

func strPtr(s string) *string { return &s }

func TestEnsureUserIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	first, err := s.EnsureUser(ctx, journal.User{ID: "user-1", Email: "a@example.com", DisplayName: "A", CreatedAt: now})
	if err != nil {
		t.Fatalf("EnsureUser() error = %v", err)
	}
	// A second call with different values must not overwrite the stored user.
	second, err := s.EnsureUser(ctx, journal.User{ID: "user-1", Email: "changed@example.com", DisplayName: "B", CreatedAt: now})
	if err != nil {
		t.Fatalf("EnsureUser() second call error = %v", err)
	}
	if second.Email != first.Email || second.DisplayName != "A" {
		t.Errorf("second EnsureUser() = %+v, want the original %+v", second, first)
	}
}

func TestEnsureJournalCreatesOnlyOne(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	j := journal.NewJournal("user-1", "A", time.Now().UTC())

	for i := 0; i < 3; i++ {
		if err := s.EnsureJournal(ctx, j); err != nil {
			t.Fatalf("EnsureJournal() call %d error = %v", i+1, err)
		}
	}
	n, err := s.db.Collection(journalsCollection).CountDocuments(ctx, bson.D{{Key: "ownerId", Value: "user-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("journals for user-1 = %d, want 1", n)
	}
}

func TestHandlesAreUnique(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Two users without handles must be allowed (the partial index ignores them).
	for _, id := range []string{"user-1", "user-2"} {
		if _, err := s.EnsureUser(ctx, journal.User{ID: id, Email: id + "@example.com", DisplayName: id, CreatedAt: now}); err != nil {
			t.Fatalf("EnsureUser(%s) error = %v", id, err)
		}
	}

	if _, err := s.UpdateUser(ctx, "user-1", journal.UpdateMeInput{Handle: strPtr("rishe")}); err != nil {
		t.Fatalf("UpdateUser(user-1) error = %v", err)
	}
	_, err := s.UpdateUser(ctx, "user-2", journal.UpdateMeInput{Handle: strPtr("rishe")})
	if !errors.Is(err, journal.ErrHandleTaken) {
		t.Errorf("UpdateUser(user-2) error = %v, want ErrHandleTaken", err)
	}
}

func TestUpdateUserNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.UpdateUser(context.Background(), "nobody", journal.UpdateMeInput{DisplayName: strPtr("X")})
	if !errors.Is(err, journal.ErrNotFound) {
		t.Errorf("UpdateUser() error = %v, want ErrNotFound", err)
	}
}
