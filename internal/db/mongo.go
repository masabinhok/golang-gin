package db

import (
	"context"
	"fmt"

	"notes-api/internal/config"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect opens a MongoDB client and returns both the client and the selected database.
// The caller owns the client and must disconnect it when the program exits.
func Connect(cfg config.Config) (*mongo.Client, *mongo.Database, error) {
	// context.WithTimeout creates a deadline for the connection startup.
	// If MongoDB does not respond within 10 seconds, we cancel and return an error.
	// This prevents the HTTP server from hanging if MongoDB is down or unreachable.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// options.Client creates a client options builder; ApplyURI parses the
	// connection string and configures the client accordingly.
	clientOpts := options.Client().ApplyURI(cfg.MongoURI)

	// mongo.Connect creates the client but does not immediately reach MongoDB.
	// It prepares the connection pool and returns an error if the URI is invalid.
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to MongoDB: %w", err)
	}

	// Ping sends a test command to MongoDB to verify the connection actually works.
	// If the server is down or the credentials are wrong, we find out here.
	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	// Select the database by name. This does not create it; MongoDB will create it
	// automatically when the first document is inserted.
	database := client.Database(cfg.MongoDB)

	return client, database, nil
}

// Disconnect closes the MongoDB connection on shutdown.
// We keep a timeout here too so cleanup cannot hang forever if the server is
// already going down and the driver needs to finish network work.
func Disconnect(client *mongo.Client) error {
	// Create a fresh context with a timeout for the disconnect operation.
	// Disconnect may take time if there are pending operations to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Disconnect closes all open connections and releases resources.
	// The context timeout ensures the cleanup does not stall the shutdown process.
	return client.Disconnect(ctx)
}
