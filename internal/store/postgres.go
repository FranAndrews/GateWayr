package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/FranAndrews/GateWayr/internal/model"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStore implements Store backed by PostgreSQL (and Google Cloud SQL).
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Ensure PostgresStore satisfies Store interface.
var _ Store = (*PostgresStore)(nil)

// InitSchema ensures the required tables and indexes exist.
func (s *PostgresStore) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS applications (
		id VARCHAR(64) PRIMARY KEY,
		applicant_name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL,
		income_amount NUMERIC(14, 2) NOT NULL,
		status VARCHAR(32) NOT NULL DEFAULT 'pending',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_applications_created_at ON applications (created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_applications_status ON applications (status);
	`
	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

func (s *PostgresStore) Create(ctx context.Context, req model.CreateApplicationRequest) (*model.Application, error) {
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

	query := `INSERT INTO applications (id, applicant_name, email, income_amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.db.ExecContext(ctx, query,
		app.ID,
		app.ApplicantName,
		app.Email,
		app.IncomeAmount,
		string(app.Status),
		app.CreatedAt,
		app.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert application: %w", err)
	}
	return &app, nil
}

func (s *PostgresStore) GetByID(ctx context.Context, id string) (*model.Application, error) {
	query := `SELECT id, applicant_name, email, income_amount, status, created_at, updated_at
		FROM applications WHERE id = $1`
	var app model.Application
	var status string
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&app.ID,
		&app.ApplicantName,
		&app.Email,
		&app.IncomeAmount,
		&status,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get application by id: %w", err)
	}
	app.Status = model.ApplicationStatus(status)
	return &app, nil
}

func (s *PostgresStore) List(ctx context.Context) ([]model.Application, error) {
	query := `SELECT id, applicant_name, email, income_amount, status, created_at, updated_at
		FROM applications ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	apps := make([]model.Application, 0)
	for rows.Next() {
		var app model.Application
		var status string
		if err := rows.Scan(
			&app.ID,
			&app.ApplicantName,
			&app.Email,
			&app.IncomeAmount,
			&status,
			&app.CreatedAt,
			&app.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		app.Status = model.ApplicationStatus(status)
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application rows: %w", err)
	}
	return apps, nil
}

func (s *PostgresStore) Update(ctx context.Context, id string, req model.UpdateApplicationRequest) (*model.Application, error) {
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.ApplicantName != nil {
		current.ApplicantName = *req.ApplicantName
	}
	if req.Email != nil {
		current.Email = *req.Email
	}
	if req.IncomeAmount != nil {
		current.IncomeAmount = *req.IncomeAmount
	}
	if req.Status != nil {
		current.Status = *req.Status
	}
	current.UpdatedAt = time.Now().UTC()

	query := `UPDATE applications
		SET applicant_name = $1, email = $2, income_amount = $3, status = $4, updated_at = $5
		WHERE id = $6`
	res, err := s.db.ExecContext(ctx, query,
		current.ApplicantName,
		current.Email,
		current.IncomeAmount,
		string(current.Status),
		current.UpdatedAt,
		current.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update application: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	return current, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM applications WHERE id = $1`
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
