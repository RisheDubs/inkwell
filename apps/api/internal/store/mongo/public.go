package mongo

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// FindUserByHandle returns the user with exactly this handle. Callers normalise and validate the
// handle first (journal.PublicService does); the store matches it as given.
//
// The $type condition repeats the handle_unique index's partial filter. MongoDB only uses a
// partial index when the query proves it falls inside that filter, so without it this lookup
// could scan every user instead of using the index.
func (s *Store) FindUserByHandle(ctx context.Context, handle string) (journal.User, error) {
	filter := bson.D{{Key: "handle", Value: bson.D{
		{Key: "$eq", Value: handle},
		{Key: "$type", Value: "string"},
	}}}
	var u journal.User
	err := s.db.Collection(usersCollection).FindOne(ctx, filter).Decode(&u)
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return journal.User{}, journal.ErrNotFound
	}
	if err != nil {
		return journal.User{}, fmt.Errorf("find user by handle: %w", err)
	}
	return u, nil
}
