package journal

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const coverID = "6af4937db325014a79b03c88.png"

// coverText returns a pointer to s (named to avoid clashing with helpers in other test files).
func coverText(s string) *string { return &s }

func TestValidateJournalUpdateAcceptsCover(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"a media ID", coverID, coverID},
		{"spaces around it are trimmed", "  " + coverID + " ", coverID},
		{"empty removes the cover", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateJournalUpdate(UpdateJournalInput{Theme: &ThemeInput{CoverMediaID: coverText(tt.in)}})
			if err != nil {
				t.Fatalf("ValidateJournalUpdate: %v", err)
			}
			if got.Theme == nil || got.Theme.CoverMediaID == nil || *got.Theme.CoverMediaID != tt.want {
				t.Fatalf("CoverMediaID = %v, want %q", got.Theme, tt.want)
			}
		})
	}
}

func TestValidateJournalUpdateRejectsBadCover(t *testing.T) {
	for _, bad := range []string{
		"../" + coverID,                       // path tricks
		"6af4937db325014a79b03c88.svg",        // a type we never accept
		"https://evil.example/" + coverID,     // a URL
		"users/someone-else/media/" + coverID, // someone else's key
		"6AF4937DB325014A79B03C88.png",        // not exactly how we mint IDs
	} {
		t.Run(bad, func(t *testing.T) {
			_, err := ValidateJournalUpdate(UpdateJournalInput{Theme: &ThemeInput{CoverMediaID: coverText(bad)}})
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Field != "theme.coverMediaId" {
				t.Fatalf("err = %v, want a ValidationError on theme.coverMediaId", err)
			}
		})
	}
}

func TestValidateJournalUpdateNeverTakesCoverKeyFromClient(t *testing.T) {
	got, err := ValidateJournalUpdate(UpdateJournalInput{Theme: &ThemeInput{
		Paper:    coverText("grid"),
		CoverKey: coverText("users/someone-else/media/" + coverID),
	}})
	if err != nil {
		t.Fatalf("ValidateJournalUpdate: %v", err)
	}
	if got.Theme.CoverKey != nil {
		t.Fatalf("CoverKey = %q, want nil: only the service may set it", *got.Theme.CoverKey)
	}
}

func TestWithCoverKeyBuildsTheKeyFromTheCaller(t *testing.T) {
	got := withCoverKey("u1", UpdateJournalInput{Theme: &ThemeInput{CoverMediaID: coverText(coverID)}})
	if want := MediaKey("u1", coverID); got.Theme.CoverKey == nil || *got.Theme.CoverKey != want {
		t.Fatalf("CoverKey = %v, want %q", got.Theme.CoverKey, want)
	}

	cleared := withCoverKey("u1", UpdateJournalInput{Theme: &ThemeInput{CoverMediaID: coverText("")}})
	if cleared.Theme.CoverKey == nil || *cleared.Theme.CoverKey != "" {
		t.Fatalf("CoverKey = %v, want a pointer to \"\" (remove the cover)", cleared.Theme.CoverKey)
	}

	untouched := withCoverKey("u1", UpdateJournalInput{Theme: &ThemeInput{Paper: coverText("grid")}})
	if untouched.Theme.CoverKey != nil {
		t.Fatal("CoverKey set although the cover was not being changed")
	}
}

func TestThemeJSONShowsTheMediaIDNeverTheKey(t *testing.T) {
	b, err := json.Marshal(Theme{Paper: "dotted", Font: "serif", Accent: "#2f5d50", CoverKey: MediaKey("u1", coverID)})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, `"coverMediaId":"`+coverID+`"`) {
		t.Errorf("JSON %s has no coverMediaId", got)
	}
	if strings.Contains(got, "coverKey") || strings.Contains(got, "users/") {
		t.Errorf("JSON %s leaks the S3 key", got)
	}

	noCover, _ := json.Marshal(Theme{Paper: "dotted", Font: "serif", Accent: "#2f5d50"})
	if strings.Contains(string(noCover), "coverMediaId") {
		t.Errorf("JSON %s has a coverMediaId although there is no cover", noCover)
	}
}
