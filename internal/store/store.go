// Package store defines the data persistence interface for the GateWayr system.
// Implementations can range from an in-memory map to Firestore or any other backend.
package store

import (
	"context"

	"github.com/FranAndrews/GateWayr/internal/model"
)

// Store is the interface that all persistence backends must implement.
type Store interface {
	// Create persists a new application and returns it with a generated ID.
	Create(ctx context.Context, req model.CreateApplicationRequest) (*model.Application, error)

	// GetByID retrieves an application by its unique identifier.
	// Returns ErrNotFound if no matching record exists.
	GetByID(ctx context.Context, id string) (*model.Application, error)

	// List returns all applications in the store.
	List(ctx context.Context) ([]model.Application, error)

	// Update modifies an existing application.
	// Returns ErrNotFound if no matching record exists.
	Update(ctx context.Context, id string, req model.UpdateApplicationRequest) (*model.Application, error)

	// Delete removes an application by its unique identifier.
	// Returns ErrNotFound if no matching record exists.
	Delete(ctx context.Context, id string) error
}
