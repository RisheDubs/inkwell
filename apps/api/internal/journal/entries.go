package journal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"
)

// Entry is one dated journal entry's metadata. Its body lives in S3 (Part 3B) at BodyKey.
type Entry struct {
	ID        string     `bson:"-" json:"id"`
	JournalID string     `bson:"-" json:"journalId"`
	OwnerID   string     `bson:"ownerId" json:"-"`
	Title     string     `bson:"title" json:"title"`
	EntryDate string     `bson:"entryDate" json:"entryDate"` // "YYYY-MM-DD"
	Mood      string     `bson:"mood,omitempty" json:"mood,omitempty"`
	Tags      []string   `bson:"tags" json:"tags"`
	BodyKey   string     `bson:"bodyKey" json:"-"`
	Excerpt   string     `bson:"excerpt" json:"excerpt"`
	WordCount int        `bson:"wordCount" json:"wordCount"`
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"-"`
}

// EntryUpdate changes an entry's metadata. A nil field means "leave unchanged".
type EntryUpdate struct {
	Title     *string
	EntryDate *string
	Mood      *string
	Tags      *[]string
	Excerpt   *string
	WordCount *int
}

// ListQuery asks for one page of an owner's entries, newest first.
type ListQuery struct {
	Limit  int
	Tag    string
	Cursor *Cursor // nil for the first page
}

// EntryPage is one page of entries plus the cursor for the next page ("" when there is none).
type EntryPage struct {
	Entries    []Entry `json:"entries"`
	NextCursor string  `json:"nextCursor,omitempty"`
}

// EntryRepository stores entry metadata. Every method takes the owner's ID and only ever
// touches that owner's entries: another user's entry looks exactly like a missing one.
type EntryRepository interface {
	CreateEntry(ctx context.Context, e Entry) (Entry, error)
	FindEntry(ctx context.Context, ownerID, id string) (Entry, error)
	ListEntries(ctx context.Context, ownerID string, q ListQuery) (EntryPage, error)
	UpdateEntry(ctx context.Context, ownerID, id string, upd EntryUpdate, now time.Time) (Entry, error)
	DeleteEntry(ctx context.Context, ownerID, id string, now time.Time) error
}

// Cursor marks a position in the newest-first list: the last entry already returned.
// Sorting by (entryDate, id) gives every entry a unique, stable position.
type Cursor struct {
	EntryDate string `json:"d"`
	ID        string `json:"i"`
}

// Encode turns the cursor into an opaque string for clients to send back unchanged.
func (c Cursor) Encode() string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor reverses Encode. A cursor the client has altered is a ValidationError.
func DecodeCursor(s string) (Cursor, error) {
	bad := &ValidationError{Field: "cursor", Message: "is not valid; use the nextCursor value from the previous page"}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, bad
	}
	var c Cursor
	if err := json.Unmarshal(raw, &c); err != nil || c.EntryDate == "" || c.ID == "" {
		return Cursor{}, bad
	}
	if _, err := time.Parse(time.DateOnly, c.EntryDate); err != nil {
		return Cursor{}, bad
	}
	return c, nil
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// NormaliseLimit applies the default page size and caps oversized requests.
func NormaliseLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultPageSize
	case limit > maxPageSize:
		return maxPageSize
	default:
		return limit
	}
}
