// Package model defines the domain entities for the GateWayr system.
package model

import "time"

// ApplicationStatus represents the lifecycle stage of a loan application.
type ApplicationStatus string

const (
	StatusPending  ApplicationStatus = "pending"
	StatusReview   ApplicationStatus = "in_review"
	StatusApproved ApplicationStatus = "approved"
	StatusRejected ApplicationStatus = "rejected"
)

// Application represents a loan/credit application submitted by an applicant.
type Application struct {
	ID            string            `json:"id"`
	ApplicantName string            `json:"applicant_name"`
	Email         string            `json:"email"`
	IncomeAmount  float64           `json:"income_amount"`
	Status        ApplicationStatus `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// CreateApplicationRequest is the payload for creating a new application.
type CreateApplicationRequest struct {
	ApplicantName string  `json:"applicant_name"`
	Email         string  `json:"email"`
	IncomeAmount  float64 `json:"income_amount"`
}

// UpdateApplicationRequest is the payload for updating an existing application.
type UpdateApplicationRequest struct {
	ApplicantName *string            `json:"applicant_name,omitempty"`
	Email         *string            `json:"email,omitempty"`
	IncomeAmount  *float64           `json:"income_amount,omitempty"`
	Status        *ApplicationStatus `json:"status,omitempty"`
}

// ErrorResponse is a standard JSON error body returned by all endpoints.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}
