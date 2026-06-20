package notes

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Note is the document shape stored in MongoDB and returned from the API.
// The bson tags tell the MongoDB driver how to serialize the struct, and the
// json tags tell Gin how to render the same data back to clients.
type Note struct {
	// ID is the MongoDB primary key. The bson tag "_id" matches MongoDB's
	// internal field name. The json tag "id" (without underscore) gives API
	// clients a cleaner field name in the response.
	ID primitive.ObjectID `bson:"_id" json:"id"`

	// Title is the note's heading.
	Title string `bson:"title" json:"title"`

	// Content is the note's body text.
	Content string `bson:"content" json:"content"`

	// Pinned marks whether the note should appear at the top of the list.
	Pinned bool `bson:"pinned" json:"pinned"`

	// CreatedAt records when the note was first inserted into the database.
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`

	// UpdatedAt records the most recent change to the note.
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// CreateNoteRequest is the incoming JSON body for POST /notes.
// The binding:"required" tags let Gin and go-playground/validator reject bad
// requests automatically before the handler tries to use missing values.
type CreateNoteRequest struct {
	// Title is required; the request fails if it is missing or empty.
	Title string `json:"title" binding:"required"`

	// Content is required; the request fails if it is missing or empty.
	Content string `json:"content" binding:"required"`

	// Pinned is optional; if omitted, it defaults to false.
	Pinned bool `json:"pinned"`
}

// UpdateNoteRequest uses the same required fields for PATCH /notes/:id.
// Keeping a separate request type makes it obvious which payload each endpoint
// accepts, even when the final fields look similar.
type UpdateNoteRequest struct {
	// Title is required for updates; the endpoint will not apply a partial update.
	Title string `json:"title" binding:"required"`

	// Content is required for updates.
	Content string `json:"content" binding:"required"`

	// Pinned is optional; if omitted, it defaults to false in the update.
	Pinned bool `json:"pinned"`
}
