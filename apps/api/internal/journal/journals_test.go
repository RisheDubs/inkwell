package journal

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateJournalUpdate(t *testing.T) {
	tests := []struct {
		name    string
		in      UpdateJournalInput
		wantErr string // the field named in the error; "" means valid
	}{
		{name: "nothing to update", in: UpdateJournalInput{}, wantErr: "-"},
		{name: "empty theme object", in: UpdateJournalInput{Theme: &ThemeInput{}}, wantErr: "-"},
		{name: "title", in: UpdateJournalInput{Title: ptr("  Morning pages  ")}},
		{name: "empty title", in: UpdateJournalInput{Title: ptr("  ")}, wantErr: "title"},
		{name: "title too long", in: UpdateJournalInput{Title: ptr(strings.Repeat("a", 101))}, wantErr: "title"},
		{name: "title with newline", in: UpdateJournalInput{Title: ptr("a\nb")}, wantErr: "title"},
		{name: "public", in: UpdateJournalInput{Visibility: ptr("Public")}},
		{name: "unknown visibility", in: UpdateJournalInput{Visibility: ptr("friends")}, wantErr: "visibility"},
		{name: "paper", in: UpdateJournalInput{Theme: &ThemeInput{Paper: ptr("grid")}}},
		{name: "unknown paper", in: UpdateJournalInput{Theme: &ThemeInput{Paper: ptr("papyrus")}}, wantErr: "theme.paper"},
		{name: "unknown font", in: UpdateJournalInput{Theme: &ThemeInput{Font: ptr("comic")}}, wantErr: "theme.font"},
		{name: "accent uppercase is normalised", in: UpdateJournalInput{Theme: &ThemeInput{Accent: ptr("#2F5D50")}}},
		{name: "accent without #", in: UpdateJournalInput{Theme: &ThemeInput{Accent: ptr("2f5d50")}}, wantErr: "theme.accent"},
		{name: "accent CSS injection", in: UpdateJournalInput{Theme: &ThemeInput{Accent: ptr("red;background:url(x)")}}, wantErr: "theme.accent"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := ValidateJournalUpdate(tc.in)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				return
			}
			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("error = %v, want a *ValidationError", err)
			}
			if tc.wantErr != "-" && verr.Field != tc.wantErr {
				t.Errorf("error field = %q, want %q", verr.Field, tc.wantErr)
			}
			_ = out
		})
	}
}

func TestValidateJournalUpdateNormalises(t *testing.T) {
	out, err := ValidateJournalUpdate(UpdateJournalInput{
		Title:      ptr("  Morning pages "),
		Visibility: ptr(" PUBLIC "),
		Theme:      &ThemeInput{Accent: ptr("#2F5D50")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if *out.Title != "Morning pages" || *out.Visibility != "public" || *out.Theme.Accent != "#2f5d50" {
		t.Errorf("got title=%q visibility=%q accent=%q", *out.Title, *out.Visibility, *out.Theme.Accent)
	}
	if out.Theme.Paper != nil || out.Theme.Font != nil {
		t.Error("unset theme fields should stay nil")
	}
}
