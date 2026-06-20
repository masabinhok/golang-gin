package server

import (
	"net/http"

	"notes-api/internal/notes"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// NewRouter builds the top-level Gin engine for the application.
// The server package stays small on purpose: it owns HTTP-wide concerns such as
// middleware, health checks, and wiring feature routers together.
func NewRouter(database *mongo.Database) *gin.Engine {
	// gin.Default creates the Gin router with two built-in middleware:
	// 1. Logger middleware - logs every incoming request
	// 2. Recovery middleware - catches panics and returns 500 instead of crashing
	r := gin.Default()

	// Health checks are useful for local testing (e.g., curl http://localhost:8080/health)
	// and for deployment probes that verify the server is running.
	r.GET("/health", func(c *gin.Context) {
		// gin.H is a shorthand for map[string]interface{} to build JSON responses.
		// It is more concise than using map[string]any directly.
		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"status": "healthy",
		})
	})

	// Mount the feature-specific notes routes after the shared health endpoint.
	// This keeps the main router simple and delegates endpoint details to the notes package.
	notes.RegisterRoutes(r, database)

	return r
}
