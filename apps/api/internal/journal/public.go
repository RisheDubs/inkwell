package journal

import (
	"context"
	"encoding/json"
	"strings"
)

// The public side of Inkwell: what anyone may read of a journal its owner has made public, with
// no sign-in. Everything here is a gate in front of the owner-scoped services that already
// exist: it turns a handle into an owner ID only when the journal is public, then reuses them.

// PublicJournal is what anyone may see of a public journal. It is its own type, never Journal or
// User, so a field added to those later can't leak through the public routes.
type PublicJournal struct {
	Handle     string               `json:"handle"`
	Title      string               `json:"title"`
	Theme      Theme                `json:"theme"` // Theme.MarshalJSON already hides the cover's S3 key
	Entries    []PublicEntrySummary `json:"entries"`
	NextCursor string               `json:"nextCursor,omitempty"`
}

// PublicEntrySummary is one entry in a public journal's list. No owner, journal or timestamps.
type PublicEntrySummary struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	EntryDate string   `json:"entryDate"`
	Mood      string   `json:"mood,omitempty"`
	Tags      []string `json:"tags"`
	Excerpt   string   `json:"excerpt"`
	WordCount int      `json:"wordCount"`
}

// PublicEntry is one entry of a public journal, with its body.
type PublicEntry struct {
	PublicEntrySummary
	Body json.RawMessage `json:"body"`
}

// PublicUsers finds users and journals. *mongo.Store implements it.
type PublicUsers interface {
	FindUserByHandle(ctx context.Context, handle string) (User, error)
	FindJournal(ctx context.Context, ownerID string) (Journal, error)
}

// publicEntries and publicMedia are the parts of EntryService and MediaService this file uses.
type publicEntries interface {
	ListEntries(ctx context.Context, ownerID string, in ListInput) (EntryPage, error)
	Entry(ctx context.Context, ownerID, id string) (EntryWithBody, error)
}

type publicMedia interface {
	DownloadURL(ctx context.Context, ownerID, mediaID string) (MediaURL, error)
}

// PublicService serves public journals.
type PublicService struct {
	users   PublicUsers
	entries publicEntries
	media   publicMedia
}

// NewPublicService wires the public side to the existing owner-scoped services.
func NewPublicService(users PublicUsers, entries publicEntries, media publicMedia) *PublicService {
	return &PublicService{users: users, entries: entries, media: media}
}

// owner returns the owner ID and journal behind handle, but only if that journal is public.
// An unknown handle, a malformed one and a private journal all give the same ErrNotFound, so
// nobody can find out which handles exist or which journals are private.
func (s *PublicService) owner(ctx context.Context, handle string) (string, Journal, error) {
	handle = strings.ToLower(strings.TrimSpace(handle))
	if !handlePattern.MatchString(handle) {
		return "", Journal{}, ErrNotFound // can't be a real handle: don't even ask the database
	}
	u, err := s.users.FindUserByHandle(ctx, handle)
	if err != nil {
		return "", Journal{}, err
	}
	j, err := s.users.FindJournal(ctx, u.ID)
	if err != nil {
		return "", Journal{}, err
	}
	if j.Visibility != "public" {
		return "", Journal{}, ErrNotFound
	}
	return u.ID, j, nil
}

// Journal returns a public journal's title, theme and a page of its entries, newest first.
func (s *PublicService) Journal(ctx context.Context, handle string, in ListInput) (PublicJournal, error) {
	ownerID, j, err := s.owner(ctx, handle)
	if err != nil {
		return PublicJournal{}, err
	}
	page, err := s.entries.ListEntries(ctx, ownerID, in) // same cursor, limit and tag rules as the owner's own list
	if err != nil {
		return PublicJournal{}, err
	}
	out := PublicJournal{
		Handle:     strings.ToLower(strings.TrimSpace(handle)),
		Title:      j.Title,
		Theme:      j.Theme,
		Entries:    make([]PublicEntrySummary, 0, len(page.Entries)), // [] not null when empty
		NextCursor: page.NextCursor,
	}
	for _, e := range page.Entries {
		out.Entries = append(out.Entries, publicSummary(e))
	}
	return out, nil
}

// Entry returns one entry of a public journal. It is looked up by ID and the handle's owner, so
// another user's entry ID gives ErrNotFound, exactly like the private routes.
func (s *PublicService) Entry(ctx context.Context, handle, id string) (PublicEntry, error) {
	ownerID, _, err := s.owner(ctx, handle)
	if err != nil {
		return PublicEntry{}, err
	}
	e, err := s.entries.Entry(ctx, ownerID, id)
	if err != nil {
		return PublicEntry{}, err
	}
	return PublicEntry{PublicEntrySummary: publicSummary(e.Entry), Body: e.Body}, nil
}

// MediaURL returns a short-lived link to an image in a public journal's folder.
func (s *PublicService) MediaURL(ctx context.Context, handle, mediaID string) (MediaURL, error) {
	ownerID, _, err := s.owner(ctx, handle)
	if err != nil {
		return MediaURL{}, err
	}
	return s.media.DownloadURL(ctx, ownerID, mediaID)
}

func publicSummary(e Entry) PublicEntrySummary {
	tags := e.Tags
	if tags == nil {
		tags = []string{} // [] in JSON, never null
	}
	return PublicEntrySummary{
		ID: e.ID, Title: e.Title, EntryDate: e.EntryDate, Mood: e.Mood,
		Tags: tags, Excerpt: e.Excerpt, WordCount: e.WordCount,
	}
}
