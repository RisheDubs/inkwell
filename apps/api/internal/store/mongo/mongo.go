// Package mongo wraps the MongoDB driver for the Inkwell API.
package mongo

import (
	"context"
	"fmt"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Store holds the MongoDB client shared by the whole API.
type Store struct {
	client *mongodriver.Client
}

// Connect creates a MongoDB client and checks the server is reachable.
func Connect(ctx context.Context, uri string) (*Store, error) {
	client, err := mongodriver.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("create mongo client: %w", err)
	}

	s := &Store{client: client}
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
