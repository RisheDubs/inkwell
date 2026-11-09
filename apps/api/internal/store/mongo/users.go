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

// FindUser returns the user with this ID, or journal.ErrNotFound.
func (s *Store) FindUser(ctx context.Context, id string) (journal.User, error) {
	var u journal.User
	err := s.db.Collection(usersCollection).FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&u)
	if errors.Is(err, mongodriver.ErrNoDocuments) {
		return journal.User{}, journal.ErrNotFound
	}
	if err != nil {
		return journal.User{}, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}

// EnsureUser inserts u if no user with its ID exists, then returns the stored user.
// $setOnInsert only applies when the upsert inserts, so an existing user is never overwritten.
func (s *Store) EnsureUser(ctx context.Context, u journal.User) (journal.User, error) {
	update := bson.D{{Key: "$setOnInsert", Value: bson.D{
		{Key: "email", Value: u.Email},
		{Key: "displayName", Value: u.DisplayName},
		{Key: "aiOptIn", Value: u.AIOptIn},
		{Key: "createdAt", Value: u.CreatedAt},
	}}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var stored journal.User
	err := s.db.Collection(usersCollection).
		FindOneAndUpdate(ctx, bson.D{{Key: "_id", Value: u.ID}}, update, opts).
		Decode(&stored)
	if err != nil {
		return journal.User{}, fmt.Errorf("ensure user: %w", err)
	}
	return stored, nil
}

// EnsureJournal inserts j if its owner has no journal yet.
func (s *Store) EnsureJournal(ctx context.Context, j journal.Journal) error {
	update := bson.D{{Key: "$setOnInsert", Value: bson.D{
		{Key: "title", Value: j.Title},
		{Key: "visibility", Value: j.Visibility},
		{Key: "theme", Value: j.Theme},
		{Key: "createdAt", Value: j.CreatedAt},
	}}}
	_, err := s.db.Collection(journalsCollection).UpdateOne(ctx,
		bson.D{{Key: "ownerId", Value: j.OwnerID}}, update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("ensure journal: %w", err)
	}
	return nil
}

// UpdateUser applies the non-nil fields of in and returns the updated user.
func (s *Store) UpdateUser(ctx context.Context, id string, in journal.UpdateMeInput) (journal.User, error) {
	set := bson.D{}
	if in.DisplayName != nil {
		set = append(set, bson.E{Key: "displayName", Value: *in.DisplayName})
	}
	if in.Handle != nil {
		set = append(set, bson.E{Key: "handle", Value: *in.Handle})
	}
	if in.AIOptIn != nil {
		set = append(set, bson.E{Key: "aiOptIn", Value: *in.AIOptIn})
	}
	if len(set) == 0 {
		return s.FindUser(ctx, id)
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var u journal.User
	err := s.db.Collection(usersCollection).
		FindOneAndUpdate(ctx, bson.D{{Key: "_id", Value: id}}, bson.D{{Key: "$set", Value: set}}, opts).
		Decode(&u)
	switch {
	case errors.Is(err, mongodriver.ErrNoDocuments):
		return journal.User{}, journal.ErrNotFound
	case mongodriver.IsDuplicateKeyError(err):
		return journal.User{}, journal.ErrHandleTaken
	case err != nil:
		return journal.User{}, fmt.Errorf("update user: %w", err)
	}
	return u, nil
}
