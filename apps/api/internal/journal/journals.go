package journal

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// UpdateJournalInput is what PATCH /api/journal accepts. A nil field means "leave unchanged".
type UpdateJournalInput struct {
	Title      *string     `json:"title"`
	Visibility *string     `json:"visibility"`
	Theme      *ThemeInput `json:"theme"`
}

// ThemeInput changes some or all of a journal's theme.
type ThemeInput struct {
	Paper  *string `json:"paper"`
	Font   *string `json:"font"`
	Accent *string `json:"accent"`
	// CoverMediaID is the media ID of an uploaded image, or "" to remove the cover.
	CoverMediaID *string `json:"coverMediaId"`
	// CoverKey is set by the service, never by the client (json:"-"): it is built from the
	// caller's own user ID, so a cover can only point into their own folder.
	CoverKey *string `json:"-"`
}

const maxTitleRunes = 100

var (
	allowedVisibility = map[string]bool{"private": true, "public": true}
	allowedPapers     = map[string]bool{"plain": true, "lined": true, "dotted": true, "grid": true}
	allowedFonts      = map[string]bool{"serif": true, "sans": true, "mono": true, "handwriting": true}
	accentPattern     = regexp.MustCompile(`^#[0-9a-f]{6}$`)
)

// Journal returns the caller's journal.
func (s *Service) Journal(ctx context.Context, ownerID string) (Journal, error) {
	return s.repo.FindJournal(ctx, ownerID)
}

// UpdateJournal validates in and applies it to the caller's journal.
func (s *Service) UpdateJournal(ctx context.Context, ownerID string, in UpdateJournalInput) (Journal, error) {
	clean, err := ValidateJournalUpdate(in)
	if err != nil {
		return Journal{}, err
	}
	return s.repo.UpdateJournal(ctx, ownerID, withCoverKey(ownerID, clean))
}

// withCoverKey turns a validated cover media ID into the S3 key the store saves ("" removes the
// cover). The key always comes from MediaKey and the caller's own user ID, never from the client.
func withCoverKey(ownerID string, in UpdateJournalInput) UpdateJournalInput {
	if in.Theme == nil || in.Theme.CoverMediaID == nil {
		return in
	}
	theme := *in.Theme // copy, so the caller's input is left untouched
	key := ""
	if id := *theme.CoverMediaID; id != "" {
		key = MediaKey(ownerID, id)
	}
	theme.CoverKey = &key
	in.Theme = &theme
	return in
}

// ValidateJournalUpdate checks and normalises PATCH /api/journal input.
// Every value is checked against an allow-list: the client can't store arbitrary styles.
func ValidateJournalUpdate(in UpdateJournalInput) (UpdateJournalInput, error) {
	themeEmpty := in.Theme == nil ||
		(in.Theme.Paper == nil && in.Theme.Font == nil && in.Theme.Accent == nil && in.Theme.CoverMediaID == nil)
	if in.Title == nil && in.Visibility == nil && themeEmpty {
		return UpdateJournalInput{}, &ValidationError{Message: "provide at least one of title, visibility or theme"}
	}

	var out UpdateJournalInput

	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		switch {
		case title == "":
			return UpdateJournalInput{}, &ValidationError{Field: "title", Message: "must not be empty"}
		case utf8.RuneCountInString(title) > maxTitleRunes:
			return UpdateJournalInput{}, &ValidationError{Field: "title", Message: fmt.Sprintf("must be at most %d characters", maxTitleRunes)}
		case strings.IndexFunc(title, unicode.IsControl) >= 0:
			return UpdateJournalInput{}, &ValidationError{Field: "title", Message: "must not contain control characters"}
		}
		out.Title = &title
	}

	if in.Visibility != nil {
		v := strings.ToLower(strings.TrimSpace(*in.Visibility))
		if !allowedVisibility[v] {
			return UpdateJournalInput{}, &ValidationError{Field: "visibility", Message: `must be "private" or "public"`}
		}
		out.Visibility = &v
	}

	if !themeEmpty {
		t := ThemeInput{} // starts empty, so a CoverKey can never come from the client
		if in.Theme.Paper != nil {
			p := strings.ToLower(strings.TrimSpace(*in.Theme.Paper))
			if !allowedPapers[p] {
				return UpdateJournalInput{}, &ValidationError{Field: "theme.paper", Message: "must be one of plain, lined, dotted, grid"}
			}
			t.Paper = &p
		}
		if in.Theme.Font != nil {
			f := strings.ToLower(strings.TrimSpace(*in.Theme.Font))
			if !allowedFonts[f] {
				return UpdateJournalInput{}, &ValidationError{Field: "theme.font", Message: "must be one of serif, sans, mono, handwriting"}
			}
			t.Font = &f
		}
		if in.Theme.Accent != nil {
			a := strings.ToLower(strings.TrimSpace(*in.Theme.Accent))
			if !accentPattern.MatchString(a) {
				return UpdateJournalInput{}, &ValidationError{Field: "theme.accent", Message: `must be a hex colour like "#2f5d50"`}
			}
			t.Accent = &a
		}
		if in.Theme.CoverMediaID != nil {
			id := strings.TrimSpace(*in.Theme.CoverMediaID)
			// "" removes the cover; anything else must be exactly a media ID from an upload.
			if id != "" && !mediaIDPattern.MatchString(id) {
				return UpdateJournalInput{}, &ValidationError{Field: "theme.coverMediaId", Message: "must be the media ID of an uploaded image, or empty to remove the cover"}
			}
			t.CoverMediaID = &id
		}
		out.Theme = &t
	}

	return out, nil
}
