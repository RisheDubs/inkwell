package journal

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxEntryTitleRunes = 200
	maxTags       = 20
)

var (
	tagPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	moods      = map[string]bool{
		"happy": true, "excited": true, "grateful": true, "calm": true, "neutral": true,
		"tired": true, "anxious": true, "sad": true, "angry": true,
	}
	earliestEntryDate = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
)

// CreateEntryInput is what POST /api/entries accepts.
type CreateEntryInput struct {
	Title     string          `json:"title"`
	EntryDate string          `json:"entryDate"`
	Mood      string          `json:"mood"`
	Tags      []string        `json:"tags"`
	Body      json.RawMessage `json:"body"`
}

// UpdateEntryInput is what PATCH /api/entries/{id} accepts. A nil field means "leave unchanged".
type UpdateEntryInput struct {
	Title     *string         `json:"title"`
	EntryDate *string         `json:"entryDate"`
	Mood      *string         `json:"mood"`
	Tags      *[]string       `json:"tags"`
	Body      json.RawMessage `json:"body"`
}

// ListInput is GET /api/entries' query string, before validation.
type ListInput struct {
	Limit  int
	Tag    string
	Cursor string
}

// EntryWithBody is one entry as the API returns it: its metadata plus its Tiptap body.
type EntryWithBody struct {
	Entry
	Body json.RawMessage `json:"body"`
}

// JournalFinder finds the caller's journal, which new entries belong to.
type JournalFinder interface {
	FindJournal(ctx context.Context, ownerID string) (Journal, error)
}

// EntryService applies Inkwell's rules for entries: metadata in MongoDB, bodies in S3.
type EntryService struct {
	entries  EntryRepository
	journals JournalFinder
	bodies   BodyStore
	newID    func() string
	now      func() time.Time
}

// NewEntryService creates an EntryService. newID makes entry IDs up front, so the body's
// key is known before the entry is saved.
func NewEntryService(entries EntryRepository, journals JournalFinder, bodies BodyStore, newID func() string) *EntryService {
	return &EntryService{entries: entries, journals: journals, bodies: bodies, newID: newID, now: time.Now}
}

// CreateEntry validates in, stores the body in S3, then the metadata in MongoDB.
func (s *EntryService) CreateEntry(ctx context.Context, ownerID string, in CreateEntryInput) (EntryWithBody, error) {
	now := s.now().UTC()

	title, err := cleanTitle(in.Title)
	if err != nil {
		return EntryWithBody{}, err
	}
	date, err := cleanEntryDate(in.EntryDate, now)
	if err != nil {
		return EntryWithBody{}, err
	}
	mood, err := cleanMood(in.Mood)
	if err != nil {
		return EntryWithBody{}, err
	}
	tags, err := cleanTags(in.Tags)
	if err != nil {
		return EntryWithBody{}, err
	}
	text, err := ParseBody(in.Body)
	if err != nil {
		return EntryWithBody{}, err
	}

	j, err := s.journals.FindJournal(ctx, ownerID)
	if err != nil {
		return EntryWithBody{}, fmt.Errorf("find journal: %w", err)
	}

	id := s.newID()
	key := BodyKey(ownerID, id)

	// Body first, then metadata. If MongoDB fails after this, the worst case is an orphaned
	// private object nobody can see, never an entry whose body is missing.
	if err := s.bodies.PutBody(ctx, key, in.Body); err != nil {
		return EntryWithBody{}, fmt.Errorf("store body: %w", err)
	}

	excerpt, words := Summarise(text)
	e, err := s.entries.CreateEntry(ctx, Entry{
		ID: id, JournalID: j.ID, OwnerID: ownerID,
		Title: title, EntryDate: date, Mood: mood, Tags: tags,
		BodyKey: key, Excerpt: excerpt, WordCount: words,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return EntryWithBody{}, fmt.Errorf("save entry: %w", err)
	}
	return EntryWithBody{Entry: e, Body: in.Body}, nil
}

// Entry returns one of the owner's entries with its body.
func (s *EntryService) Entry(ctx context.Context, ownerID, id string) (EntryWithBody, error) {
	e, err := s.entries.FindEntry(ctx, ownerID, id)
	if err != nil {
		return EntryWithBody{}, err
	}
	if err := checkBodyKey(ownerID, e.BodyKey); err != nil {
		return EntryWithBody{}, err
	}
	body, err := s.bodies.GetBody(ctx, e.BodyKey)
	if err != nil {
		return EntryWithBody{}, fmt.Errorf("load body: %w", err)
	}
	return EntryWithBody{Entry: e, Body: body}, nil
}

// ListEntries returns one page of the owner's entries, newest first. Bodies are not included:
// list views only need the excerpt.
func (s *EntryService) ListEntries(ctx context.Context, ownerID string, in ListInput) (EntryPage, error) {
	q := ListQuery{Limit: NormaliseLimit(in.Limit)}
	if in.Tag != "" {
		tag := strings.ToLower(strings.TrimSpace(in.Tag))
		if !tagPattern.MatchString(tag) {
			return EntryPage{}, &ValidationError{Field: "tag", Message: "must be 1-32 characters: lowercase letters, numbers or -"}
		}
		q.Tag = tag
	}
	if in.Cursor != "" {
		c, err := DecodeCursor(in.Cursor)
		if err != nil {
			return EntryPage{}, err
		}
		q.Cursor = &c
	}
	return s.entries.ListEntries(ctx, ownerID, q)
}

// UpdateEntry changes an entry's metadata and, if given, its body.
func (s *EntryService) UpdateEntry(ctx context.Context, ownerID, id string, in UpdateEntryInput) (EntryWithBody, error) {
	now := s.now().UTC()
	upd, text, err := validateEntryUpdate(in, now)
	if err != nil {
		return EntryWithBody{}, err
	}

	// Ownership first: a body is never written for an entry the caller doesn't own.
	current, err := s.entries.FindEntry(ctx, ownerID, id)
	if err != nil {
		return EntryWithBody{}, err
	}
	if err := checkBodyKey(ownerID, current.BodyKey); err != nil {
		return EntryWithBody{}, err
	}

	body := in.Body
	if body != nil {
		if err := s.bodies.PutBody(ctx, current.BodyKey, body); err != nil {
			return EntryWithBody{}, fmt.Errorf("store body: %w", err)
		}
		excerpt, words := Summarise(text)
		upd.Excerpt, upd.WordCount = &excerpt, &words
	}

	e, err := s.entries.UpdateEntry(ctx, ownerID, id, upd, now)
	if err != nil {
		return EntryWithBody{}, err
	}
	if body == nil {
		if body, err = s.bodies.GetBody(ctx, e.BodyKey); err != nil {
			return EntryWithBody{}, fmt.Errorf("load body: %w", err)
		}
	}
	return EntryWithBody{Entry: e, Body: body}, nil
}

// DeleteEntry soft-deletes the owner's entry. The body stays in S3 until a purge job (later phase).
func (s *EntryService) DeleteEntry(ctx context.Context, ownerID, id string) error {
	return s.entries.DeleteEntry(ctx, ownerID, id, s.now().UTC())
}

// validateEntryUpdate checks PATCH input and returns the metadata update plus the body's text.
func validateEntryUpdate(in UpdateEntryInput, now time.Time) (EntryUpdate, string, error) {
	if in.Title == nil && in.EntryDate == nil && in.Mood == nil && in.Tags == nil && in.Body == nil {
		return EntryUpdate{}, "", &ValidationError{Message: "provide at least one of title, entryDate, mood, tags or body"}
	}

	var upd EntryUpdate
	if in.Title != nil {
		title, err := cleanTitle(*in.Title)
		if err != nil {
			return EntryUpdate{}, "", err
		}
		upd.Title = &title
	}
	if in.EntryDate != nil {
		date, err := cleanEntryDate(*in.EntryDate, now)
		if err != nil {
			return EntryUpdate{}, "", err
		}
		upd.EntryDate = &date
	}
	if in.Mood != nil {
		mood, err := cleanMood(*in.Mood)
		if err != nil {
			return EntryUpdate{}, "", err
		}
		upd.Mood = &mood
	}
	if in.Tags != nil {
		tags, err := cleanTags(*in.Tags)
		if err != nil {
			return EntryUpdate{}, "", err
		}
		upd.Tags = &tags
	}

	var text string
	if in.Body != nil {
		var err error
		if text, err = ParseBody(in.Body); err != nil {
			return EntryUpdate{}, "", err
		}
	}
	return upd, text, nil
}

// checkBodyKey is defence in depth: even if a bad record reached MongoDB, the API never reads
// or writes outside the caller's own S3 prefix.
func checkBodyKey(ownerID, key string) error {
	if !strings.HasPrefix(key, "users/"+ownerID+"/entries/") {
		return fmt.Errorf("body key %q is outside owner %q's prefix", key, ownerID)
	}
	return nil
}

func cleanTitle(s string) (string, error) {
	title := strings.TrimSpace(s)
	switch {
	case utf8.RuneCountInString(title) > maxEntryTitleRunes:
		return "", &ValidationError{Field: "title", Message: fmt.Sprintf("must be at most %d characters", maxEntryTitleRunes)}
	case strings.IndexFunc(title, unicode.IsControl) >= 0:
		return "", &ValidationError{Field: "title", Message: "must not contain control characters"}
	}
	return title, nil
}

func cleanEntryDate(s string, now time.Time) (string, error) {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return "", &ValidationError{Field: "entryDate", Message: "must be a real date like 2026-11-10"}
	}
	// One day of slack: while it's still the 10th in UTC, it's already the 11th in Melbourne.
	if d.Before(earliestEntryDate) || d.After(now.AddDate(0, 0, 1)) {
		return "", &ValidationError{Field: "entryDate", Message: "must be between 1900-01-01 and tomorrow"}
	}
	return s, nil
}

func cleanMood(s string) (string, error) {
	mood := strings.ToLower(strings.TrimSpace(s))
	if mood != "" && !moods[mood] {
		return "", &ValidationError{Field: "mood", Message: "must be one of happy, excited, grateful, calm, neutral, tired, anxious, sad or angry"}
	}
	return mood, nil
}

func cleanTags(in []string) ([]string, error) {
	if len(in) > maxTags {
		return nil, &ValidationError{Field: "tags", Message: fmt.Sprintf("must be at most %d tags", maxTags)}
	}
	out := make([]string, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, raw := range in {
		tag := strings.ToLower(strings.TrimSpace(raw))
		if !tagPattern.MatchString(tag) {
			return nil, &ValidationError{Field: "tags", Message: "each tag must be 1-32 characters: lowercase letters, numbers or -"}
		}
		if !seen[tag] {
			seen[tag] = true
			out = append(out, tag)
		}
	}
	return out, nil
}
