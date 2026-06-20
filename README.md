## Golang Gin & MongoDB Atlas

[![Go Version](https://img.shields.io/github/go-mod/go-version/masabinhok/golang-gin)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A step-by-step backend tutorial for building a cleanly structured Gin API on MongoDB Atlas, with an emphasis on readable architecture, beginner-friendly explanations, and production-aware startup flow.



#### Prerequisites

- Required: A basic understanding of Go (Golang).
- Optional: Familiarity with other backend programming languages or frameworks.

## Setup

#### Base Setup
1. Initialize the module: `go mod init api-name` - this creates `go.mod` for dependency tracking.
2. Create a MongoDB Atlas cluster and record your connection string and any required credentials.
3. Add a `.env` file in the project root and populate it according to the [Env Variables](#env-variables) section below.
4. Fetch the project dependencies listed in [Packages to download](#packages-to-download) (for example with `go get` or `go mod tidy`). This updates `go.mod` and `go.sum`.
5. Install Air for live reloading: `go install github.com/air-verse/air@latest`. Ensure `$(go env GOPATH)/bin` is in your `PATH` so the `air` binary is available.
6. Add a `.air.toml` file in the project root using the configuration shown in [Air Config](#air-config).
7. Create `./internal/config/config.go` to load environment variables into a typed `Config` value (see [Config Setup](#config-setup)).
8. Create `./internal/db/mongo.go` to manage MongoDB connection and disconnection logic (see [DB Setup](#db-setup)).
9. Create `./internal/server/router.go` and register your routes with Gin (see [Router Setup](#router-setup)).
10. Place your application entry point at `./cmd/api/main.go` so it matches the build command used by Air.
11. In `main.go`, wire the application: load config, connect to the database, defer disconnect, initialize the router, and start the server (see [Main Setup](#main-setup)).
12. From the project root, run `air` to start the development server with live reload.
13. Verify the health endpoint with Postman or Thunder Client: `http://localhost:8080/health` should return a successful JSON response.
14. If MongoDB Atlas rejects your IP, add your machine's IP under Network Access, or use a temporary rule, in Atlas.

#### Notes CRUD
Build the CRUD feature in `./internal/notes` in small layers so each part has one job.

1. Start with [note.go](./internal/notes/note.go), which defines the database model and the request payload used for validation.
2. Add [repository.go](./internal/notes/repository.go) to isolate MongoDB operations from the HTTP layer.
3. Add [handler.go](./internal/notes/handler.go) to read the request, validate input, and forward the work to the repository.
4. Add [routes.go](./internal/notes/routes.go) to register `/notes` endpoints and connect them to the handler.
5. Wire the notes routes into [internal/server/router.go](./internal/server/router.go) so the main router can recognize them.
6. Review the setup code below and compare it with the package flow above.
7. Test the endpoints in Postman or Thunder Client after starting the server.

#### Notes Package Setup
The package split below is intentional:

- `note.go` defines the MongoDB document and request payloads.
- `repository.go` owns database access only.
- `handler.go` owns HTTP request parsing and response codes.
- `routes.go` mounts the feature under `/notes`.

#### Project Structure
```text
gin-framework/
├── .air.toml
├── .env
├── .gitignore
├── README.md
├── cmd/
│   └── api/
│       └── main.go
├── go.mod
├── go.sum
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── db/
│   │   └── mongo.go
│   ├── notes/
│   │   ├── handler.go
│   │   ├── note.go
│   │   ├── repository.go
│   │   └── routes.go
│   └── server/
│       └── router.go
└── tmp/
```

`tmp/` is created by Air and is safe to leave out of version control.

#### Env Variables

```env
MONGO_URI="your_mongodb_atlas_connection_string"
PORT=8080
MONGO_DB_NAME="your_database_name"
```

#### Packages to download
```bash
# Gin web framework
go get github.com/gin-gonic/gin@latest

# Dotenv parser to load .env variables
go get github.com/joho/godotenv

# Official MongoDB driver
go get go.mongodb.org/mongo-driver/v2/mongo

# BSON helpers (included with the driver but listed here for clarity)
go get go.mongodb.org/mongo-driver/v2/bson
```

#### Air Config
```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/api.exe ./cmd/api"
bin = "tmp/api.exe"
delay = 300
exclude_dir = ["tmp", "vendor"]
exclude_regex = ["_test.go"]

[log]
time = true
```

#### Config Setup

```go
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config groups the environment values the application needs to start.
// Keeping them in one struct makes the rest of the code easier to read because
// each package can depend on a typed value instead of reaching into os.Getenv.
// LoadEnv returns this struct plus an error, so the caller gets either a fully
// populated config snapshot or a clear reason why startup should stop.
type Config struct {
	MongoURI   string
	MongoDB    string
	ServerPort string
}

// LoadEnv reads environment values for local development and deployment.
// It returns a Config value and an error. When the error is nil, the Config is
// safe to use everywhere else in the program.
// We try to load .env for beginner-friendly local setup, but we do not fail if
// the file is missing because production usually injects environment variables
// directly. The required keys are still validated one by one below.
func LoadEnv() (Config, error) {
	// Load the optional .env file into the current process.
	// If the file does not exist, we keep going so the app can still use real
	// environment variables. Any other I/O problem is surfaced immediately.
	// This is why we do not hard fail on missing .env in production-style setups.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env file: %w", err)
	}

	// Read each required key explicitly so the error message tells the beginner
	// which value is missing instead of failing with a vague zero-value struct.
	mongoURI, err := extractEnv("MONGO_URI")
	if err != nil {
		return Config{}, err
	}

	mongoDB, err := extractEnv("MONGO_DB_NAME")
	if err != nil {
		return Config{}, err
	}

	port, err := extractEnv("PORT")
	if err != nil {
		return Config{}, err
	}

	return Config{
		MongoURI:   mongoURI,
		MongoDB:    mongoDB,
		ServerPort: port,
	}, nil
}

// extractEnv returns a required environment variable or a descriptive error.
// The return value is a string because environment variables are always text.
// The error is the signal that startup should stop instead of guessing a value.
// This keeps the missing-key check in one place so all configuration failures
// behave consistently.
func extractEnv(key string) (string, error) {
	// os.Getenv reads from the current process environment.
	// If the key is unset, we return an error instead of silently using an empty
	// string because the database connection string and port are both required.
	// Returning an empty string here would just move the failure to a later step.
	val := os.Getenv(key)

	if val == "" {
		return "", fmt.Errorf("missing required env: %s", key)
	}

	return val, nil
}
```

#### DB Setup
```go
package db

import (
	"context"
	"fmt"
	"time"

	// Replace "your/module/path" with the module path declared in your go.mod.
	// In this repository, the module path is notes-api.
	"notes-api/internal/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect opens a MongoDB client and returns the client, the selected database,
// and an error. The caller owns the client and must disconnect it when the
// program exits.
func Connect(cfg config.Config) (*mongo.Client, *mongo.Database, error) {
	// A startup timeout prevents the app from hanging forever if MongoDB is slow
	// or unreachable. That matters because the HTTP server should not begin until
	// the database connection has been verified.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel releases the timer resources created by WithTimeout.
	// Even when everything succeeds, we still close it so nothing leaks.
	defer cancel()

	// ApplyURI stores the MongoDB Atlas connection string on the client options.
	// mongo.Connect then uses those options to build a live client handle.
	clientOpts := options.Client().ApplyURI(cfg.MongoURI)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to MongoDB: %w", err)
	}

	// Ping proves the server is reachable before we start the HTTP layer.
	// If this fails, the app exits early instead of serving requests with a bad DB link.
	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	// Database returns a handle to the named database. It does not open a new
	// connection; it simply scopes future collection calls to that database.
	database := client.Database(cfg.MongoDB)

	return client, database, nil
}

// Disconnect closes the MongoDB connection on shutdown.
// We keep a timeout here too so cleanup cannot hang forever if the server is
// already going down and the driver needs to finish network work.
func Disconnect(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// Cancel is still necessary even during shutdown because it releases the
	// timeout timer and any resources attached to the context.
	defer cancel()

	// Disconnect tells the driver to close sockets and finish cleanup work.
	// The returned error is important because shutdown failures can help explain
	// why a process exited uncleanly.
	return client.Disconnect(ctx)
}
```

#### Router Setup
```go
package server

import (
	"net/http"

	"notes-api/internal/notes"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// NewRouter builds the top-level Gin engine for the application and returns it
// fully configured. The server package stays small on purpose: it owns HTTP-wide
// concerns such as middleware, health checks, and wiring feature routers together.
func NewRouter(database *mongo.Database) *gin.Engine {
	// gin.Default installs the logger and recovery middleware so beginners get
	// useful request logs and a process-safe panic handler out of the box.
	// It returns a ready-to-use *gin.Engine, which is Gin's main router type.
	r := gin.Default()

	// Health checks are useful for local testing and for deployment probes.
	// This route returns JSON so tools like Postman and load balancers can read it.
	r.GET("/health", func(c *gin.Context) {
		// gin.H is a tiny convenience alias for JSON objects.
		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"status": "healthy",
		})
	})

	// Mount the feature-specific notes routes after the shared health endpoint.
	// Passing the database here gives the notes package one shared dependency
	// instead of making every handler open its own connection.
	notes.RegisterRoutes(r, database)

	return r
}
```

#### Main Setup
```go
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
// It returns a single error value so the caller can decide how to exit.
// Keeping the full startup sequence in one place makes the dependency flow easy
// to explain and keeps resource ownership obvious.
func run() (err error) {
	// Load environment values from .env and the process environment.
	// The application needs these values before anything else because the MongoDB
	// connection string and server port both come from configuration.
	// If this fails, we do not continue because later steps depend on it.
	cfg, err := config.LoadEnv()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	// Open the MongoDB connection before constructing the router because the HTTP
	// handlers will need a shared database handle to read and write notes.
	// If this step fails, the server cannot serve requests safely, so we stop.
	// The function returns both the client and the database handle, and we keep
	// both because the router needs the database and the shutdown path needs client.
	client, database, err := db.Connect(cfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	// Keep cleanup directly next to the resource acquisition so the reader can see
	// which object owns the connection. The named return value lets us preserve a
	// startup or runtime error while still reporting shutdown problems clearly.
	defer func() {
		// This runs even if run returns early, which is why defer is the right tool
		// for cleanup. It keeps the database close call tied to the connection open.
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
	// This keeps connection ownership in one place instead of spreading it around.
	router := server.NewRouter(database)

	// Format the listen address as ":8080" so Gin binds to the configured port
	// on all interfaces. This keeps the server portable across local machines.
	// The string passed to Run is the TCP address the server listens on.
	addr := fmt.Sprintf(":%s", cfg.ServerPort)

	// Start the HTTP server. Gin blocks here until the process stops or returns an
	// error, so this is the last step in the request-serving path.
	// If Run returns nil, the server handled shutdown normally.
	if err := router.Run(addr); err != nil {
		return fmt.Errorf("run server on %s: %w", addr, err)
	}

	return nil
}
```

#### Notes API Roadmap

| HTTP Method | Endpoint Path | Description | Expected Input Payload (JSON) | Expected Success HTTP Status Code |
| --- | --- | --- | --- | --- |
| `POST` | `/notes` | Create a new note document. | `{"title":"My note","content":"Short body","pinned":false}` | `201 Created` |
| `GET` | `/notes` | Return the full notes collection. | `N/A` | `200 OK` |
| `GET` | `/notes/:id` | Return one note by MongoDB ObjectID. | `N/A` | `200 OK` |
| `PATCH` | `/notes/:id` | Update an existing note by ObjectID. | `{"title":"Updated title","content":"Updated body","pinned":true}` | `200 OK` |
| `DELETE` | `/notes/:id` | Delete a note by ObjectID. | `N/A` | `200 OK` |

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

![Views](https://api.visitorbadge.io/api/visitors?path=https%3A%2F%2Fgithub.com%2Fmasabinhok%2Fgolang-gin&label=VIEWS&countColor=%23263159)