package notes

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Handler translates HTTP requests into repository calls.
// It keeps request parsing, validation, and response codes in one place so the
// database layer does not need to know anything about Gin.
type Handler struct {
	repo *Repository
}

// NewHandler wires the repository into the HTTP handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateNote validates the incoming JSON, builds a Note, and stores it.
func (h *Handler) CreateNote(c *gin.Context) {
	var req CreateNoteRequest

	// ShouldBindJSON reads the request body, parses JSON, and then runs Gin's
	// validator against the struct tags such as binding:"required".
	// If the client sends malformed JSON, or omits a required field, this call
	// returns an error before we try to use the payload.
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON body or missing required fields",
		})
		return
	}

	// Use UTC so stored timestamps are consistent regardless of where the server
	// is running. That makes later debugging and testing easier.
	now := time.Now().UTC()

	// Build the full document here because the handler owns request translation
	// and can attach server-generated fields like IDs and timestamps.
	note := Note{
		ID: primitive.NewObjectID(),

		Title:   req.Title,
		Content: req.Content,
		Pinned:  req.Pinned,

		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := h.repo.Create(c.Request.Context(), &note)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create note",
		})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// ReadNotes returns the full collection.
func (h *Handler) ReadNotes(c *gin.Context) {
	notes, err := h.repo.Read(c.Request.Context())
	if err != nil {
		// Return 500 for database failures.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read notes",
		})
		return
	}

	// StatusOK (200) is the default for successful GET requests.
	// Wrap the notes in an object so the response structure is consistent
	// with other endpoints that use named fields.
	c.JSON(http.StatusOK, gin.H{
		"notes": notes,
	})
}

// ReadNoteById looks up a single note using the id path parameter.
func (h *Handler) ReadNoteById(c *gin.Context) {
	// c.Param extracts the named route parameter (e.g., ":id" from GET /notes/:id).
	idStr := c.Param("id")

	// The route parameter arrives as a hex string, but MongoDB expects an
	// ObjectID value. Conversion is required before the repository can query.
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		// If the client sends a malformed ID, return 400 (Bad Request).
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id",
		})
		return
	}

	note, err := h.repo.ReadOne(c.Request.Context(), objID)
	if err != nil {
		// errors.Is checks if the error is exactly mongo.ErrNoDocuments,
		// which means the database search succeeded but found no matching document.
		if errors.Is(err, mongo.ErrNoDocuments) {
			// StatusNotFound (404) means the resource does not exist.
			c.JSON(http.StatusNotFound, gin.H{
				"error": "note not found",
			})
			return
		}
		// For any other error, return 500 (Internal Server Error).
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch note",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"note": note,
	})
}

// UpdateNoteById validates the path parameter and request body, then writes the
// updated fields back to MongoDB.
func (h *Handler) UpdateNoteById(c *gin.Context) {
	// Extract the :id from the URL path.
	idStr := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		// Invalid hex string = bad request.
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id",
		})
		return
	}

	var updateDto UpdateNoteRequest

	// The same binding path used in CreateNote works here too. The required tags
	// on UpdateNoteRequest let Gin validate the body before we touch the repo.
	if err := c.ShouldBindJSON(&updateDto); err != nil {
		// Malformed JSON or missing required fields.
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON body or missing required fields",
		})
		return
	}

	// Pass the update request and get back the updated note.
	note, err := h.repo.UpdateOne(c.Request.Context(), objID, updateDto)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// No document matched the ID.
			c.JSON(http.StatusNotFound, gin.H{
				"error": "note not found",
			})
			return
		}
		// Database error during update.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update note",
		})
		return
	}

	// Return the updated document so the client sees the current state.
	c.JSON(http.StatusOK, gin.H{
		"note": note,
	})
}

// DeleteNoteById removes a note if the repository finds a matching document.
func (h *Handler) DeleteNoteById(c *gin.Context) {
	// Extract the :id parameter.
	idStr := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		// Malformed ObjectID hex string.
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id",
		})
		return
	}

	// DeleteOne returns (bool, error) where the bool indicates if a document was deleted.
	deleted, err := h.repo.DeleteOne(c.Request.Context(), objID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// No document found with this ID.
			c.JSON(http.StatusNotFound, gin.H{
				"error": "note not found",
			})
			return
		}

		// Database error during delete operation.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete the note",
		})
		return
	}

	// If deleted is false, the repository found no matching document.
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "note not found",
		})
		return
	}

	// Successful deletion; confirm with a simple success response.
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
	})
}
