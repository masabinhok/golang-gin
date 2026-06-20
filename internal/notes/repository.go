package notes

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository owns all MongoDB access for notes.
// This keeps database code out of the HTTP layer so handlers stay focused on
// request parsing, response shaping, and status codes.
type Repository struct {
	coll *mongo.Collection
}

// NewRepository binds the repository to the notes collection.
// We create the collection handle once and reuse it for every query because the
// collection name is stable for the lifetime of the application.
func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		coll: db.Collection("notes"),
	}
}

// Create inserts a note document into MongoDB.
// The 5-second timeout ensures one slow database call does not tie up the
// request forever, which is a simple production safeguard for each operation.
func (r *Repository) Create(ctx context.Context, note *Note) (*Note, error) {
	// context.WithTimeout creates a child context that cancels after 5 seconds.
	// If the database operation takes longer, the driver stops waiting.
	operationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

	// defer cancel() ensures the timeout is cleaned up even if we return early.
	defer cancel()

	// InsertOne sends the note to MongoDB. If the write fails, we return a
	// wrapped error so the handler can decide which HTTP status to send.
	// The underscore (_) means we don't use the returned InsertedID.
	_, err := r.coll.InsertOne(operationCtx, note)
	if err != nil {
		return nil, fmt.Errorf("insert note: %w", err)
	}

	// Return the same note because MongoDB auto-assigned the ObjectID
	// that is already set in the note struct.
	return note, nil
}

// Read returns every note document in the collection.
// An empty filter means "match everything", which is the simplest possible
// read path for a tutorial repository.
func (r *Repository) Read(ctx context.Context) ([]*Note, error) {
	operationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// bson.M{} is a BSON map with no filters, so it matches all documents.
	// If you wanted to filter (e.g., pinned notes only), you would add conditions here.
	filter := bson.M{}

	// Find returns a cursor, which is a streaming handle over the result set.
	// The cursor is lazy: it does not fetch all documents immediately.
	cursor, err := r.coll.Find(operationCtx, filter)
	if err != nil {
		return nil, fmt.Errorf("find notes: %w", err)
	}

	// The cursor must be closed after use so server resources are released even
	// if decoding fails partway through the result set.
	// This is a best practice to avoid leaving open connections.
	defer cursor.Close(operationCtx)

	// Initialize an empty slice to hold the results.
	var notes []*Note

	// cursor.All fetches all remaining documents and decodes them into the slice.
	// If decoding fails, the caller gets a wrapped error instead of partial data.
	if err := cursor.All(operationCtx, &notes); err != nil {
		return nil, fmt.Errorf("decode notes: %w", err)
	}

	return notes, nil
}

// ReadOne looks up a single note by its MongoDB ObjectID.
func (r *Repository) ReadOne(ctx context.Context, id primitive.ObjectID) (*Note, error) {
	operationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Filter by the MongoDB primary key ("_id") to find one document.
	filter := bson.M{
		"_id": id,
	}

	// Declare a zero-value Note struct to hold the result.
	// We use a value (not a pointer) because Decode will populate it.
	var note Note

	// FindOne queries for one document and returns a SingleResult.
	// options.FindOne() allows for additional query options (not needed here).
	// Decode immediately parses the result into the Note struct.
	err := r.coll.FindOne(operationCtx, filter, options.FindOne()).Decode(&note)
	if err != nil {
		// If err is mongo.ErrNoDocuments, no note matched the query.
		return nil, fmt.Errorf("find note by id: %w", err)
	}

	// Return a pointer to the now-populated note.
	return &note, nil
}

// UpdateOne applies a field update and returns the updated note document.
func (r *Repository) UpdateOne(ctx context.Context, id primitive.ObjectID, updateDto UpdateNoteRequest) (*Note, error) {
	operationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Filter by the document id so only one note is updated.
	filter := bson.M{
		"_id": id,
	}

	// The $set operator tells MongoDB to update only these fields, leaving others
	// unchanged. This is safer than $replace, which would remove unspecified fields.
	update := bson.M{
		"$set": bson.M{
			"title":   updateDto.Title,
			"content": updateDto.Content,
			"pinned":  updateDto.Pinned,
		},
	}

	// SetReturnDocument(options.After) tells MongoDB to return the new document
	// after the update, not the old one. This lets the API send the client the
	// current state immediately without a second query.
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedNote Note

	// FindOneAndUpdate applies the update and returns the result (or error).
	err := r.coll.FindOneAndUpdate(operationCtx, filter, update, opts).Decode(&updatedNote)
	if err != nil {
		// Errors include ErrNoDocuments if the ID did not match any document.
		return nil, fmt.Errorf("update note: %w", err)
	}

	return &updatedNote, nil
}

// DeleteOne removes a note by id.
// The bool return value tells the handler whether a document was actually
// removed, while the error is reserved for real database or lookup failures.
func (r *Repository) DeleteOne(ctx context.Context, id primitive.ObjectID) (bool, error) {
	operationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Filter to match only the note with this ObjectID.
	filter := bson.M{
		"_id": id,
	}

	// DeleteOne removes the first (and only) document matching the filter.
	// It returns a DeleteResult with metadata about how many documents were deleted.
	result, err := r.coll.DeleteOne(operationCtx, filter)
	if err != nil {
		return false, fmt.Errorf("delete note: %w", err)
	}

	// result.DeletedCount is the number of documents removed by this operation.
	// It should be 0 (not found) or 1 (found and deleted).
	// Returning the count as a boolean makes the handler logic clear: true = deleted, false = not found.
	return result.DeletedCount == 1, nil
}
