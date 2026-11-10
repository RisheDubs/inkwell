package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// entryDoc is how an entry is stored: the domain fields plus MongoDB's ObjectIDs.
type entryDoc struct {
	ID            bson.ObjectID `bson:"_id"`
	JournalID     bson.ObjectID `bson:"journalId"`
	journal.Entry `bson:",inline"`
}

func (d entryDoc) toDomain() journal.Entry {
	e := d.Entry
	e.ID = d.ID.Hex()
	e.JournalID = d.JournalID.Hex()
	return e
}

// ownedEntryFilter matches one live (not deleted) entry, and only if it belongs to ownerID.
// ok is false when id isn't a valid ObjectID; callers treat that exactly like "not found".
func ownedEntryFilter(ownerID, id string) (bson.D, bool) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, false
	}
	return bson.D{
		{Key: "_id", Value: oid},
		{Key: "ownerId", Value: ownerID},
		{Key: "deletedAt", Value: nil}, // matches a missing field too
	}, true
}

// CreateEntry stores e and returns it with its new ID. If e.ID is set (a valid ObjectID hex)
// it is used, so the caller can build the S3 body key before the entry exists.
func (s *Store) CreateEntry(ctx context.Context, e journal.Entry) (journal.Entry, error) {
	id := bson.NewObjectID()
	if e.ID != "" {
		parsed, err := bson.ObjectIDFromHex(e.ID)
		if err != nil {
			return journal.Entry{}, fmt.Errorf("create entry: invalid id: %w", err)
		}
		id = parsed
	}
	journalID, err := bson.ObjectIDFromHex(e.JournalID)
	if err != nil {
		return journal.Entry{}, fmt.Errorf("create entry: invalid journal id: %w", err)
	}
	if e.Tags == nil {
		e.Tags = []string{} // store [] not null, so tag queries and JSON stay consistent
	}

	d := entryDoc{ID: id, JournalID: journalID, Entry: e}
	if _, err := s.db.Collection(entriesCollection).InsertOne(ctx, d); err != nil {
		return journal.Entry{}, fmt.Errorf("create entry: %w", err)
	}
	return d.toDomain(), nil
}

// FindEntry returns the owner's entry, or journal.ErrNotFound if it doesn't exist,
// was deleted, or belongs to someone else.
func (s *Store) FindEntry(ctx context.Context, ownerID, id string) (journal.Entry, error) {
	filter, ok := ownedEntryFilter(ownerID, id)
	if !ok {
		return journal.Entry{}, journal.ErrNotFound
	}
	var d entryDoc
	err := s.db.Collection(entriesCollection).FindOne(ctx, filter).Decode(&d)
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return journal.Entry{}, journal.ErrNotFound
	}
	if err != nil {
		return journal.Entry{}, fmt.Errorf("find entry: %w", err)
	}
	return d.toDomain(), nil
}

// ListEntries returns one page of the owner's live entries, newest first.
func (s *Store) ListEntries(ctx context.Context, ownerID string, q journal.ListQuery) (journal.EntryPage, error) {
	limit := journal.NormaliseLimit(q.Limit)

	filter := bson.D{
		{Key: "ownerId", Value: ownerID},
		{Key: "deletedAt", Value: nil},
	}
	if q.Tag != "" {
		filter = append(filter, bson.E{Key: "tags", Value: q.Tag})
	}
	if q.Cursor != nil {
		cursorID, err := bson.ObjectIDFromHex(q.Cursor.ID)
		if err != nil {
			return journal.EntryPage{}, &journal.ValidationError{Field: "cursor", Message: "is not valid; use the nextCursor value from the previous page"}
		}
		// "Older than the last entry I saw": an earlier date, or the same date with a smaller _id.
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "entryDate", Value: bson.D{{Key: "$lt", Value: q.Cursor.EntryDate}}}},
			bson.D{
				{Key: "entryDate", Value: q.Cursor.EntryDate},
				{Key: "_id", Value: bson.D{{Key: "$lt", Value: cursorID}}},
			},
		}})
	}

	// Ask for one extra entry: if it comes back, there is another page.
	opts := options.Find().
		SetSort(bson.D{{Key: "entryDate", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit + 1))

	cur, err := s.db.Collection(entriesCollection).Find(ctx, filter, opts)
	if err != nil {
		return journal.EntryPage{}, fmt.Errorf("list entries: %w", err)
	}
	var docs []entryDoc
	if err := cur.All(ctx, &docs); err != nil {
		return journal.EntryPage{}, fmt.Errorf("list entries: %w", err)
	}

	page := journal.EntryPage{Entries: make([]journal.Entry, 0, limit)}
	for i, d := range docs {
		if i == limit {
			last := docs[limit-1]
			page.NextCursor = journal.Cursor{EntryDate: last.EntryDate, ID: last.ID.Hex()}.Encode()
			break
		}
		page.Entries = append(page.Entries, d.toDomain())
	}
	return page, nil
}

// UpdateEntry applies the non-nil fields of upd to the owner's entry.
func (s *Store) UpdateEntry(ctx context.Context, ownerID, id string, upd journal.EntryUpdate, now time.Time) (journal.Entry, error) {
	filter, ok := ownedEntryFilter(ownerID, id)
	if !ok {
		return journal.Entry{}, journal.ErrNotFound
	}

	set := bson.D{{Key: "updatedAt", Value: now}}
	if upd.Title != nil {
		set = append(set, bson.E{Key: "title", Value: *upd.Title})
	}
	if upd.EntryDate != nil {
		set = append(set, bson.E{Key: "entryDate", Value: *upd.EntryDate})
	}
	if upd.Mood != nil {
		set = append(set, bson.E{Key: "mood", Value: *upd.Mood})
	}
	if upd.Tags != nil {
		set = append(set, bson.E{Key: "tags", Value: *upd.Tags})
	}
	if upd.Excerpt != nil {
		set = append(set, bson.E{Key: "excerpt", Value: *upd.Excerpt})
	}
	if upd.WordCount != nil {
		set = append(set, bson.E{Key: "wordCount", Value: *upd.WordCount})
	}

	var d entryDoc
	err := s.db.Collection(entriesCollection).FindOneAndUpdate(ctx, filter,
		bson.D{{Key: "$set", Value: set}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&d)
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return journal.Entry{}, journal.ErrNotFound
	}
	if err != nil {
		return journal.Entry{}, fmt.Errorf("update entry: %w", err)
	}
	return d.toDomain(), nil
}

// DeleteEntry soft-deletes the owner's entry by setting deletedAt. It disappears from
// FindEntry and ListEntries but stays in the database (and its body in S3) for recovery.
func (s *Store) DeleteEntry(ctx context.Context, ownerID, id string, now time.Time) error {
	filter, ok := ownedEntryFilter(ownerID, id)
	if !ok {
		return journal.ErrNotFound
	}
	res, err := s.db.Collection(entriesCollection).UpdateOne(ctx, filter,
		bson.D{{Key: "$set", Value: bson.D{{Key: "deletedAt", Value: now}, {Key: "updatedAt", Value: now}}}})
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	if res.MatchedCount == 0 {
		return journal.ErrNotFound
	}
	return nil
}
