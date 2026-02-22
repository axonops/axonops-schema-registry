package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// ListExporters handles GET /exporters
func (h *Handler) ListExporters(w http.ResponseWriter, r *http.Request) {
	names, err := h.registry.ListExporters(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}
	if names == nil {
		names = []string{}
	}
	writeJSON(w, http.StatusOK, names)
}

// CreateExporter handles POST /exporters
func (h *Handler) CreateExporter(w http.ResponseWriter, r *http.Request) {
	var req types.CreateExporterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, "Exporter name is required")
		return
	}

	exporter := &storage.ExporterRecord{
		Name:                req.Name,
		ContextType:         req.ContextType,
		Context:             req.Context,
		Subjects:            req.Subjects,
		SubjectRenameFormat: req.SubjectRenameFormat,
		Config:              req.Config,
	}

	if err := h.registry.CreateExporter(r.Context(), exporter); err != nil {
		if errors.Is(err, storage.ErrExporterExists) {
			writeError(w, http.StatusConflict, types.ErrorCodeExporterExists, "Exporter already exists: "+req.Name)
			return
		}
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ExporterNameResponse{Name: req.Name})
}

// GetExporter handles GET /exporters/{name}
func (h *Handler) GetExporter(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	exporter, err := h.registry.GetExporter(r.Context(), name)
	if err != nil {
		if errors.Is(err, storage.ErrExporterNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeExporterNotFound, "Exporter not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ExporterResponse{
		Name:                exporter.Name,
		ContextType:         exporter.ContextType,
		Context:             exporter.Context,
		Subjects:            exporter.Subjects,
		SubjectRenameFormat: exporter.SubjectRenameFormat,
		Config:              exporter.Config,
	})
}

// UpdateExporter handles PUT /exporters/{name}
func (h *Handler) UpdateExporter(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req types.UpdateExporterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	exporter := &storage.ExporterRecord{
		Name:                name,
		ContextType:         req.ContextType,
		Context:             req.Context,
		Subjects:            req.Subjects,
		SubjectRenameFormat: req.SubjectRenameFormat,
		Config:              req.Config,
	}

	if err := h.registry.UpdateExporter(r.Context(), exporter); err != nil {
		if errors.Is(err, storage.ErrExporterNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeExporterNotFound, "Exporter not found: "+name)
			return
		}
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ExporterNameResponse{Name: name})
}

// DeleteExporter handles DELETE /exporters/{name}
func (h *Handler) DeleteExporter(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.registry.DeleteExporter(r.Context(), name); err != nil {
		if errors.Is(err, storage.ErrExporterNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeExporterNotFound, "Exporter not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ExporterNameResponse{Name: name})
}

// PauseExporter handles PUT /exporters/{name}/pause
func (h *Handler) PauseExporter(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.registry.PauseExporter(r.Context(), name); err != nil {
		if errors.Is(err, storage.ErrExporterNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeExporterNotFound, "Exporter not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ExporterNameResponse{Name: name})
}

// ResumeExporter handles PUT /exporters/{name}/resume
func (h *Handler) ResumeExporter(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.registry.ResumeExporter(r.Context(), name); err != nil {
		if errors.Is(err, storage.ErrExporterNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeExporterNotFound, "Exporter not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ExporterNameResponse{Name: name})
}

// ResetExporter handles PUT /exporters/{name}/reset
func (h *Handler) ResetExporter(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.registry.ResetExporter(r.Context(), name); err != nil {
		if errors.Is(err, storage.ErrExporterNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeExporterNotFound, "Exporter not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ExporterNameResponse{Name: name})
}

// GetExporterStatus handles GET /exporters/{name}/status
func (h *Handler) GetExporterStatus(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	status, err := h.registry.GetExporterStatus(r.Context(), name)
	if err != nil {
		if errors.Is(err, storage.ErrExporterNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeExporterNotFound, "Exporter not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ExporterStatusResponse{
		Name:   status.Name,
		State:  status.State,
		Offset: status.Offset,
		Ts:     status.Ts,
		Trace:  status.Trace,
	})
}

// GetExporterConfig handles GET /exporters/{name}/config
func (h *Handler) GetExporterConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	config, err := h.registry.GetExporterConfig(r.Context(), name)
	if err != nil {
		if errors.Is(err, storage.ErrExporterNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeExporterNotFound, "Exporter not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	if config == nil {
		config = map[string]string{}
	}
	writeJSON(w, http.StatusOK, config)
}

// UpdateExporterConfig handles PUT /exporters/{name}/config
func (h *Handler) UpdateExporterConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req types.UpdateExporterConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if err := h.registry.UpdateExporterConfig(r.Context(), name, req.Config); err != nil {
		if errors.Is(err, storage.ErrExporterNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeExporterNotFound, "Exporter not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ExporterNameResponse{Name: name})
}
