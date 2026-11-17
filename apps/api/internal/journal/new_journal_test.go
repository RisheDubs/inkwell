package journal

import (
	"strings"
	"testing"
	"time"
)

// A new journal may be made public later, so its defaults must contain nothing private.
func TestNewJournalStartsPrivateWithANeutralTitle(t *testing.T) {
	j := NewJournal("u-1234-secret-sub", time.Now())

	if j.Title != "My Journal" {
		t.Errorf("Title = %q, want %q", j.Title, "My Journal")
	}
	if strings.Contains(j.Title, "u-1234") {
		t.Errorf("Title %q contains the owner's ID", j.Title)
	}
	if j.Visibility != "private" {
		t.Errorf("Visibility = %q, want private: journals must never start public", j.Visibility)
	}
}
