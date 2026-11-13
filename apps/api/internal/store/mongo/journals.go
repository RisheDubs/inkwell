package mongo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// journalDoc is how a journal is stored: the domain fields plus MongoDB's ObjectID.
type journalDoc struct {
	ID              bson.ObjectID `bson:"_id"`
	journal.Journal `bson:",inline"`
}

func (d journalDoc) toDomain() journal.Journal {
	j := d.Journal
	j.ID = d.ID.Hex()
	return j
}

// FindJournal returns the owner's journal, or journal.ErrNotFound.
func (s *Store) FindJournal(ctx context.Context, ownerID string) (journal.Journal, error) {
	var d journalDoc
	err := s.db.Collection(journalsCollection).FindOne(ctx, bson.D{{Key: "ownerId", Value: ownerID}}).Decode(&d)
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return journal.Journal{}, journal.ErrNotFound
	}
	if err != nil {
		return journal.Journal{}, fmt.Errorf("find journal: %w", err)
	}
	return d.toDomain(), nil
}

// UpdateJournal applies the non-nil fields of in to the owner's journal.
func (s *Store) UpdateJournal(ctx context.Context, ownerID string, in journal.UpdateJournalInput) (journal.Journal, error) {
	set := bson.D{}
	unset := bson.D{}
	if in.Title != nil {
		set = append(set, bson.E{Key: "title", Value: *in.Title})
	}
	if in.Visibility != nil {
		set = append(set, bson.E{Key: "visibility", Value: *in.Visibility})
	}
	if in.Theme != nil {
		// Dotted paths change one theme field without replacing the whole theme object.
		if in.Theme.Paper != nil {
			set = append(set, bson.E{Key: "theme.paper", Value: *in.Theme.Paper})
		}
		if in.Theme.Font != nil {
			set = append(set, bson.E{Key: "theme.font", Value: *in.Theme.Font})
		}
		if in.Theme.Accent != nil {
			set = append(set, bson.E{Key: "theme.accent", Value: *in.Theme.Accent})
		}
		if in.Theme.CoverKey != nil {
			if *in.Theme.CoverKey == "" {
				// Removing the cover deletes the field, matching coverKey's omitempty.
				unset = append(unset, bson.E{Key: "theme.coverKey", Value: ""})
			} else {
				set = append(set, bson.E{Key: "theme.coverKey", Value: *in.Theme.CoverKey})
			}
		}
	}
	if len(set) == 0 && len(unset) == 0 {
		return s.FindJournal(ctx, ownerID)
	}

	update := bson.D{}
	if len(set) > 0 {
		update = append(update, bson.E{Key: "$set", Value: set})
	}
	if len(unset) > 0 {
		update = append(update, bson.E{Key: "$unset", Value: unset})
	}

	var d journalDoc
	err := s.db.Collection(journalsCollection).FindOneAndUpdate(ctx,
		bson.D{{Key: "ownerId", Value: ownerID}},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&d)
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return journal.Journal{}, journal.ErrNotFound
	}
	if err != nil {
		return journal.Journal{}, fmt.Errorf("update journal: %w", err)
	}
	return d.toDomain(), nil
}
