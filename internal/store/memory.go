package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/FranAndrews/GateWayr/internal/model"
	"github.com/google/uuid"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("entity not found")

// MemoryStore is a thread-safe, in-memory implementation of the Store interface.
// Suitable for local development and testing. Will be replaced by Firestore in production.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]model.Application
}

// NewMemoryStore creates an initialised in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]model.Application),
	}
}

func (s *MemoryStore) Create(_ context.Context, req model.CreateApplicationRequest) (*model.Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	app := model.Application{
		ID:            uuid.New().String(),
		ApplicantName: req.ApplicantName,
		Email:         req.Email,
		IncomeAmount:  req.IncomeAmount,
		Status:        model.StatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.data[app.ID] = app
	return &app, nil
}

func (s *MemoryStore) GetByID(_ context.Context, id string) (*model.Application, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, ok := s.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &app, nil
}

func (s *MemoryStore) List(_ context.Context) ([]model.Application, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apps := make([]model.Application, 0, len(s.data))
	for _, app := range s.data {
		apps = append(apps, app)
	}
	return apps, nil
}

func (s *MemoryStore) Update(_ context.Context, id string, req model.UpdateApplicationRequest) (*model.Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.data[id]
	if !ok {
		return nil, ErrNotFound
	}

	if req.ApplicantName != nil {
		app.ApplicantName = *req.ApplicantName
	}
	if req.Email != nil {
		app.Email = *req.Email
	}
	if req.IncomeAmount != nil {
		app.IncomeAmount = *req.IncomeAmount
	}
	if req.Status != nil {
		app.Status = *req.Status
	}
	app.UpdatedAt = time.Now().UTC()

	s.data[id] = app
	return &app, nil
}

func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[id]; !ok {
		return ErrNotFound
	}
	delete(s.data, id)
	return nil
}
