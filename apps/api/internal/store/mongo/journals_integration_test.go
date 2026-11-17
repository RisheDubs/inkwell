//go:build integration

package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

func TestJournalCoverIsSetKeptAndCleared(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedJournal(t, s, "user-a")

	key := journal.MediaKey("user-a", "6af4937db325014a79b03c88.png")
	j, err := s.UpdateJournal(ctx, "user-a", journal.UpdateJournalInput{Theme: &journal.ThemeInput{CoverKey: &key}})
	if err != nil {
		t.Fatalf("UpdateJournal(set cover) error = %v", err)
	}
	if j.Theme.CoverKey != key {
		t.Fatalf("CoverKey = %q, want %q", j.Theme.CoverKey, key)
	}

	// Changing another theme field leaves the cover alone.
	paper := "lined"
	j, err = s.UpdateJournal(ctx, "user-a", journal.UpdateJournalInput{Theme: &journal.ThemeInput{Paper: &paper}})
	if err != nil {
		t.Fatalf("UpdateJournal(paper) error = %v", err)
	}
	if j.Theme.CoverKey != key || j.Theme.Paper != "lined" {
		t.Errorf("theme = %+v, want paper lined and the cover kept", j.Theme)
	}

	// "" removes the cover, and the rest of the theme stays.
	empty := ""
	j, err = s.UpdateJournal(ctx, "user-a", journal.UpdateJournalInput{Theme: &journal.ThemeInput{CoverKey: &empty}})
	if err != nil {
		t.Fatalf("UpdateJournal(clear cover) error = %v", err)
	}
	if j.Theme.CoverKey != "" || j.Theme.Paper != "lined" {
		t.Errorf("theme = %+v, want no cover and paper still lined", j.Theme)
	}

	// The field is really gone from the document ($unset), not saved as an empty string.
	var raw struct {
		Theme bson.M `bson:"theme"`
	}
	if err := s.db.Collection(journalsCollection).FindOne(ctx, bson.D{{Key: "ownerId", Value: "user-a"}}).Decode(&raw); err != nil {
		t.Fatalf("FindOne(raw) error = %v", err)
	}
	if _, ok := raw.Theme["coverKey"]; ok {
		t.Errorf("stored theme still has coverKey: %v", raw.Theme)
	}
}
