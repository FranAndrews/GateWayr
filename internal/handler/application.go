// Package handler implements the HTTP handlers for the CRUD API endpoints.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/FranAndrews/GateWayr/internal/model"
	"github.com/FranAndrews/GateWayr/internal/store"
)

// ApplicationHandler holds the dependencies for application CRUD endpoints.
type ApplicationHandler struct {
	store store.Store
}

// NewApplicationHandler creates a handler backed by the given store.
func NewApplicationHandler(s store.Store) *ApplicationHandler {
	return &ApplicationHandler{store: s}
}

// RegisterRoutes mounts all application routes onto the given ServeMux.
// Go 1.22+ enhanced ServeMux supports method and path-parameter patterns.
func (h *ApplicationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/applications", h.Create)
	mux.HandleFunc("GET /api/v1/applications", h.List)
	mux.HandleFunc("GET /api/v1/applications/{id}", h.GetByID)
	mux.HandleFunc("PUT /api/v1/applications/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/applications/{id}", h.Delete)
}

// Create handles POST /api/v1/applications
func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body: "+err.Error())
		return
	}

	if strings.TrimSpace(req.ApplicantName) == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "applicant_name is required")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "email is required")
		return
	}

	app, err := h.store.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, app)
}

// List handles GET /api/v1/applications
func (h *ApplicationHandler) List(w http.ResponseWriter, r *http.Request) {
	apps, err := h.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  apps,
		"count": len(apps),
	})
}

// GetByID handles GET /api/v1/applications/{id}
func (h *ApplicationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	app, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "application not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, app)
}

// Update handles PUT /api/v1/applications/{id}
func (h *ApplicationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req model.UpdateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body: "+err.Error())
		return
	}

	app, err := h.store.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "application not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, app)
}

// Delete handles DELETE /api/v1/applications/{id}
func (h *ApplicationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	err := h.store.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "application not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, errType, message string) {
	writeJSON(w, status, model.ErrorResponse{
		Error:   errType,
		Code:    status,
		Message: message,
	})
}
