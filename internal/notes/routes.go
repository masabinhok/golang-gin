package notes

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RegisterRoutes mounts every notes endpoint under one /notes group.
// Grouping the routes keeps the router readable and makes it obvious that all
// of these handlers share the same repository and resource model.
func RegisterRoutes(r *gin.Engine, db *mongo.Database) {
	// Create the repository once so every handler uses the same collection handle.
	// This prevents opening new database connections for each request.
	repo := NewRepository(db)

	// Create one handler value and reuse its methods for each endpoint.
	// This avoids rebuilding dependencies on every request.
	h := NewHandler(repo)

	// r.Group creates a route group with a shared prefix (/notes).
	// All routes within this block will be prefixed with /notes.
	notesGroup := r.Group("/notes")
	{
		// POST /notes - Create a new note
		notesGroup.POST("", h.CreateNote)
		// GET /notes - List all notes
		notesGroup.GET("", h.ReadNotes)
		// GET /notes/:id - Fetch one note by id
		notesGroup.GET("/:id", h.ReadNoteById)
		// PATCH /notes/:id - Update an existing note
		notesGroup.PATCH("/:id", h.UpdateNoteById)
		// DELETE /notes/:id - Remove a note
		notesGroup.DELETE("/:id", h.DeleteNoteById)
	}
}
