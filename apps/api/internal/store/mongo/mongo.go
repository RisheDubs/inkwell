// Package mongo stores Inkwell's data in MongoDB.
package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	usersCollection    = "users"
	journalsCollection = "journals"
)

// Store holds the MongoDB client shared by the whole API.
type Store struct {
	client *mongodriver.Client
	db     *mongodriver.Database
}

// Connect creates a MongoDB client for the named database and checks the server is reachable.
func Connect(ctx context.Context, uri, database string) (*Store, error) {
	client, err := mongodriver.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("create mongo client: %w", err)
	}

	s := &Store{client: client, db: client.Database(database)}
	if err := s.Ping(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongo: %w", err)
	}
	return s, nil
}

// Ping checks that the primary MongoDB server is reachable.
func (s *Store) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, readpref.Primary())
}

// Close disconnects from MongoDB, waiting for in-flight operations up to ctx's deadline.
func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// EnsureIndexes creates the indexes the API relies on. It is safe to run on every startup:
// creating an index that already exists with the same definition does nothing.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	// Handles are unique, but only among users who have set one. A plain unique index
	// would treat every "no handle" as the same value and allow just one such user.
	_, err := s.db.Collection(usersCollection).Indexes().CreateOne(ctx, mongodriver.IndexModel{
		Keys: bson.D{{Key: "handle", Value: 1}},
		Options: options.Index().
			SetName("handle_unique").
			SetUnique(true).
			SetPartialFilterExpression(bson.D{{Key: "handle", Value: bson.D{{Key: "$type", Value: "string"}}}}),
	})
	if err != nil {
		return fmt.Errorf("create users.handle index: %w", err)
	}

	// One journal per user for now. Unique also makes concurrent first sign-ins safe.
	_, err = s.db.Collection(journalsCollection).Indexes().CreateOne(ctx, mongodriver.IndexModel{
		Keys:    bson.D{{Key: "ownerId", Value: 1}},
		Options: options.Index().SetName("ownerId_unique").SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create journals.ownerId index: %w", err)
	}
	return nil
}
