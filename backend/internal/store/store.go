package store

import (
	"context"
	"errors"
	"time"

	"phystech-answer/backend/internal/model"
)

var ErrNotFound = errors.New("not found")

// Store is the persistence boundary. The in-memory implementation below is
// meant to be swapped for a real database (e.g. Postgres) without touching
// the HTTP layer.
type Store interface {
	ListDocuments(ctx context.Context) ([]model.Document, error)
	GetDocument(ctx context.Context, id string) (model.Document, error)
	CreateDocument(ctx context.Context, doc model.Document) (model.Document, error)
	UpdateDocument(ctx context.Context, doc model.Document) (model.Document, error)
	DeleteDocument(ctx context.Context, id string) error

	ListUsers(ctx context.Context) ([]model.User, error)
	CreateUser(ctx context.Context, user model.User) (model.User, error)

	// MarkRead records that userID has read docID; returns the timestamp.
	MarkRead(ctx context.Context, docID, userID string) (time.Time, error)
	// ReadStatus returns docID -> readAt for all documents read by userID.
	ReadStatus(ctx context.Context, userID string) (map[string]time.Time, error)
}
