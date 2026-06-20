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
type Config struct {
	// MongoURI is the connection string for MongoDB (e.g., mongodb://localhost:27017)
	MongoURI string
	// MongoDB is the name of the database within the MongoDB server.
	MongoDB string
	// ServerPort is the port number the HTTP server listens on (e.g., 8080).
	ServerPort string
}

// LoadEnv reads environment values for local development and deployment.
// We try to load .env for beginner-friendly local setup, but we do not fail if
// the file is missing because production usually injects environment variables
// directly. The required keys are still validated one by one below.
func LoadEnv() (Config, error) {
	// Load the optional .env file into the current process environment.
	// godotenv.Load reads .env and sets each KEY=VALUE as an environment variable.
	// If the file does not exist (os.ErrNotExist), we keep going so the app can
	// still use environment variables set by the deployment system.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Any error other than "file not found" is a real problem, so we fail.
		return Config{}, fmt.Errorf("load .env file: %w", err)
	}

	// Read each required key explicitly so the error message tells the beginner
	// which value is missing instead of failing with a vague zero-value struct.
	mongoURI, err := extractEnv("MONGO_URI")
	if err != nil {
		return Config{}, err
	}

	// Extract the database name from the environment.
	mongoDB, err := extractEnv("MONGO_DB_NAME")
	if err != nil {
		return Config{}, err
	}

	// Extract the port that the HTTP server will listen on.
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
// This keeps the missing-key check in one place so all configuration failures
// behave consistently.
func extractEnv(key string) (string, error) {
	// os.Getenv reads from the current process environment.
	// If the key is unset, we return an error instead of silently using an empty
	// string because the database connection string and port are both required.
	val := os.Getenv(key)

	if val == "" {
		return "", fmt.Errorf("missing required env: %s", key)
	}

	return val, nil
}
