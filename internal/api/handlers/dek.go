package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// ListKEKs handles GET /dek-registry/v1/keks
func (h *Handler) ListKEKs(w http.ResponseWriter, r *http.Request) {
	includeDeleted := r.URL.Query().Get("deleted") == "true"

	keks, err := h.registry.ListKEKs(r.Context(), includeDeleted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Return list of KEK names
	names := make([]string, 0, len(keks))
	for _, kek := range keks {
		names = append(names, kek.Name)
	}
	writeJSON(w, http.StatusOK, names)
}

// CreateKEK handles POST /dek-registry/v1/keks
func (h *Handler) CreateKEK(w http.ResponseWriter, r *http.Request) {
	var req types.CreateKEKRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	kek := &storage.KEKRecord{
		Name:     req.Name,
		KmsType:  req.KmsType,
		KmsKeyID: req.KmsKeyID,
		KmsProps: req.KmsProps,
		Doc:      req.Doc,
		Shared:   req.Shared,
	}

	if err := h.registry.CreateKEK(r.Context(), kek); err != nil {
		if errors.Is(err, storage.ErrKEKExists) {
			writeError(w, http.StatusConflict, types.ErrorCodeKEKExists, "Key encryption key already exists: "+req.Name)
			return
		}
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, kekToResponse(kek))
}

// GetKEK handles GET /dek-registry/v1/keks/{name}
func (h *Handler) GetKEK(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	includeDeleted := r.URL.Query().Get("deleted") == "true"

	kek, err := h.registry.GetKEK(r.Context(), name, includeDeleted)
	if err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, kekToResponse(kek))
}

// UpdateKEK handles PUT /dek-registry/v1/keks/{name}
func (h *Handler) UpdateKEK(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req types.UpdateKEKRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	// Get existing to merge
	existing, err := h.registry.GetKEK(r.Context(), name, false)
	if err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Apply updates
	if req.KmsProps != nil {
		existing.KmsProps = req.KmsProps
	}
	if req.Doc != "" {
		existing.Doc = req.Doc
	}
	if req.Shared != nil {
		existing.Shared = *req.Shared
	}

	if err := h.registry.UpdateKEK(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, kekToResponse(existing))
}

// DeleteKEK handles DELETE /dek-registry/v1/keks/{name}
func (h *Handler) DeleteKEK(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	permanent := r.URL.Query().Get("permanent") == "true"

	if err := h.registry.DeleteKEK(r.Context(), name, permanent); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"name": name})
}

// UndeleteKEK handles PUT /dek-registry/v1/keks/{name}/undelete
func (h *Handler) UndeleteKEK(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.registry.UndeleteKEK(r.Context(), name); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+name)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"name": name})
}

// ListDEKs handles GET /dek-registry/v1/keks/{name}/deks
func (h *Handler) ListDEKs(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	includeDeleted := r.URL.Query().Get("deleted") == "true"

	subjects, err := h.registry.ListDEKs(r.Context(), kekName, includeDeleted)
	if err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	if subjects == nil {
		subjects = []string{}
	}
	writeJSON(w, http.StatusOK, subjects)
}

// CreateDEK handles POST /dek-registry/v1/keks/{name}/deks
func (h *Handler) CreateDEK(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")

	var req types.CreateDEKRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	dek := &storage.DEKRecord{
		KEKName:              kekName,
		Subject:              req.Subject,
		Version:              req.Version,
		Algorithm:            req.Algorithm,
		EncryptedKeyMaterial: req.EncryptedKeyMaterial,
	}

	if err := h.registry.CreateDEK(r.Context(), dek); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKExists) {
			writeError(w, http.StatusConflict, types.ErrorCodeDEKExists, "Data encryption key already exists")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dekToResponse(dek))
}

// GetDEK handles GET /dek-registry/v1/keks/{name}/deks/{subject}
func (h *Handler) GetDEK(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")
	algorithm := r.URL.Query().Get("algorithm")
	includeDeleted := r.URL.Query().Get("deleted") == "true"

	dek, err := h.registry.GetDEK(r.Context(), kekName, subject, -1, algorithm, includeDeleted)
	if err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeDEKNotFound, "Data encryption key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dekToResponse(dek))
}

// ListDEKVersions handles GET /dek-registry/v1/keks/{name}/deks/{subject}/versions
func (h *Handler) ListDEKVersions(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")
	algorithm := r.URL.Query().Get("algorithm")
	includeDeleted := r.URL.Query().Get("deleted") == "true"

	versions, err := h.registry.ListDEKVersions(r.Context(), kekName, subject, algorithm, includeDeleted)
	if err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	if versions == nil {
		versions = []int{}
	}
	writeJSON(w, http.StatusOK, versions)
}

// GetDEKVersion handles GET /dek-registry/v1/keks/{name}/deks/{subject}/versions/{version}
func (h *Handler) GetDEKVersion(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")
	algorithm := r.URL.Query().Get("algorithm")
	includeDeleted := r.URL.Query().Get("deleted") == "true"

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid version")
		return
	}

	dek, err := h.registry.GetDEK(r.Context(), kekName, subject, version, algorithm, includeDeleted)
	if err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeDEKNotFound, "Data encryption key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dekToResponse(dek))
}

// DeleteDEK handles DELETE /dek-registry/v1/keks/{name}/deks/{subject}
func (h *Handler) DeleteDEK(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")
	algorithm := r.URL.Query().Get("algorithm")
	permanent := r.URL.Query().Get("permanent") == "true"

	if err := h.registry.DeleteDEK(r.Context(), kekName, subject, -1, algorithm, permanent); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeDEKNotFound, "Data encryption key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"subject": subject})
}

// UndeleteDEK handles PUT /dek-registry/v1/keks/{name}/deks/{subject}/undelete
func (h *Handler) UndeleteDEK(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")
	algorithm := r.URL.Query().Get("algorithm")

	if err := h.registry.UndeleteDEK(r.Context(), kekName, subject, -1, algorithm); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeDEKNotFound, "Data encryption key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"subject": subject})
}

func kekToResponse(kek *storage.KEKRecord) types.KEKResponse {
	return types.KEKResponse{
		Name:     kek.Name,
		KmsType:  kek.KmsType,
		KmsKeyID: kek.KmsKeyID,
		KmsProps: kek.KmsProps,
		Doc:      kek.Doc,
		Shared:   kek.Shared,
		Ts:       kek.Ts,
		Deleted:  kek.Deleted,
	}
}

func dekToResponse(dek *storage.DEKRecord) types.DEKResponse {
	return types.DEKResponse{
		KEKName:              dek.KEKName,
		Subject:              dek.Subject,
		Version:              dek.Version,
		Algorithm:            dek.Algorithm,
		EncryptedKeyMaterial: dek.EncryptedKeyMaterial,
		KeyMaterial:          dek.KeyMaterial,
		Ts:                   dek.Ts,
		Deleted:              dek.Deleted,
	}
}
