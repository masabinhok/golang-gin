package main

import (
	"fmt"
	"log"

	"notes-api/internal/config"
	"notes-api/internal/db"
	"notes-api/internal/server"
)

// main stays intentionally small so the startup story is easy to follow in a
// tutorial. It delegates to run, which returns errors instead of exiting early
// so deferred cleanup still has a chance to execute.
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run wires the application together in the order a beginner should learn it:
// load configuration, connect to MongoDB, build the router, and start HTTP.
// Keeping the full startup sequence in one place makes the dependency flow easy
// to explain and keeps resource ownership obvious.
func run() (err error) {
	// Load environment values from .env and the process environment.
	// The application needs these values before anything else because the MongoDB
	// connection string and server port both come from configuration.
	cfg, err := config.LoadEnv()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	// Open the MongoDB connection before constructing the router because the HTTP
	// handlers will need a shared database handle to read and write notes.
	// If this step fails, the server cannot serve requests safely, so we stop.
	client, database, err := db.Connect(cfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	// Keep cleanup directly next to the resource acquisition so the reader can see
	// which object owns the connection. The named return value lets us preserve a
	// startup or runtime error while still reporting shutdown problems clearly.
	defer func() {
		if disconnectErr := db.Disconnect(client); disconnectErr != nil {
			if err == nil {
				err = fmt.Errorf("disconnect from database: %w", disconnectErr)
				return
			}

			// If the server already failed for another reason, keep that primary
			// error and record cleanup failure as a warning for debugging.
			log.Printf("warning: failed to disconnect from database: %v", disconnectErr)
		}
	}()

	// Build the Gin router after the database is available so route handlers can
	// receive the shared dependency they need without creating their own client.
	router := server.NewRouter(database)

	// Format the listen address as ":8080" so Gin binds to the configured port
	// on all interfaces. This keeps the server portable across local machines.
	addr := fmt.Sprintf(":%s", cfg.ServerPort)

	// Start the HTTP server. Gin blocks here until the process stops or returns an
	// error, so this is the last step in the request-serving path.
	if err := router.Run(addr); err != nil {
		return fmt.Errorf("run server on %s: %w", addr, err)
	}

	return nil
}
