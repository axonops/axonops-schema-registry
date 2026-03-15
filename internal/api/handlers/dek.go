package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// ListKEKs handles GET /dek-registry/v1/keks
func (h *Handler) ListKEKs(w http.ResponseWriter, r *http.Request) {
	includeDeleted := r.URL.Query().Get("deleted") == "true"

	keks, err := h.registry.ListKEKs(r.Context(), includeDeleted)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	// Return list of KEK names
	names := make([]string, 0, len(keks))
	for _, kek := range keks {
		names = append(names, kek.Name)
	}
	names = applyStringPagination(names, r)
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

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "kek"
		hints.TargetID = req.Name
		hints.AfterHash = hashKEK(kek)
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
		writeInternalError(w, err)
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
		writeInternalError(w, err)
		return
	}

	beforeHash := hashKEK(existing)

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
		writeInternalError(w, err)
		return
	}

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "kek"
		hints.TargetID = name
		hints.BeforeHash = beforeHash
		hints.AfterHash = hashKEK(existing)
	}

	writeJSON(w, http.StatusOK, kekToResponse(existing))
}

// DeleteKEK handles DELETE /dek-registry/v1/keks/{name}
func (h *Handler) DeleteKEK(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	permanent := r.URL.Query().Get("permanent") == "true"

	// Fetch before deletion for audit before_hash.
	existing, _ := h.registry.GetKEK(r.Context(), name, true)

	if err := h.registry.DeleteKEK(r.Context(), name, permanent); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+name)
			return
		}
		writeInternalError(w, err)
		return
	}

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "kek"
		hints.TargetID = name
		if existing != nil {
			hints.BeforeHash = hashKEK(existing)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// UndeleteKEK handles POST /dek-registry/v1/keks/{name}/undelete
func (h *Handler) UndeleteKEK(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Fetch deleted state for before_hash.
	existing, _ := h.registry.GetKEK(r.Context(), name, true)

	if err := h.registry.UndeleteKEK(r.Context(), name); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+name)
			return
		}
		writeInternalError(w, err)
		return
	}

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "kek"
		hints.TargetID = name
		if existing != nil {
			hints.BeforeHash = hashKEK(existing)
		}
		// After undelete, fetch restored state.
		if restored, err := h.registry.GetKEK(r.Context(), name, false); err == nil {
			hints.AfterHash = hashKEK(restored)
		}
	}

	w.WriteHeader(http.StatusNoContent)
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
		writeInternalError(w, err)
		return
	}

	if subjects == nil {
		subjects = []string{}
	}
	subjects = applyStringPagination(subjects, r)
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

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "dek"
		hints.TargetID = kekName
		hints.AfterHash = hashDEK(dek)
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
		writeInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dekToGetResponse(dek))
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
		writeInternalError(w, err)
		return
	}

	if versions == nil {
		versions = []int{}
	}
	versions = applyIntPagination(versions, r)
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
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion, "Invalid version: must be a positive integer")
		return
	}
	if version <= 0 {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion, "Invalid version: must be a positive integer")
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
		writeInternalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dekToGetResponse(dek))
}

// DeleteDEK handles DELETE /dek-registry/v1/keks/{name}/deks/{subject}
func (h *Handler) DeleteDEK(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")
	algorithm := r.URL.Query().Get("algorithm")
	permanent := r.URL.Query().Get("permanent") == "true"

	// Fetch before deletion for audit before_hash.
	existing, _ := h.registry.GetDEK(r.Context(), kekName, subject, -1, algorithm, true)

	if err := h.registry.DeleteDEK(r.Context(), kekName, subject, -1, algorithm, permanent); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeDEKNotFound, "Data encryption key not found")
			return
		}
		writeInternalError(w, err)
		return
	}

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "dek"
		hints.TargetID = kekName
		if existing != nil {
			hints.BeforeHash = hashDEK(existing)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// UndeleteDEK handles POST /dek-registry/v1/keks/{name}/deks/{subject}/undelete
func (h *Handler) UndeleteDEK(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")
	algorithm := r.URL.Query().Get("algorithm")

	// Fetch deleted state for before_hash.
	existing, _ := h.registry.GetDEK(r.Context(), kekName, subject, -1, algorithm, true)

	if err := h.registry.UndeleteDEK(r.Context(), kekName, subject, -1, algorithm); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeDEKNotFound, "Data encryption key not found")
			return
		}
		writeInternalError(w, err)
		return
	}

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "dek"
		hints.TargetID = kekName
		if existing != nil {
			hints.BeforeHash = hashDEK(existing)
		}
		if restored, err := h.registry.GetDEK(r.Context(), kekName, subject, -1, algorithm, false); err == nil {
			hints.AfterHash = hashDEK(restored)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteDEKVersion handles DELETE /dek-registry/v1/keks/{name}/deks/{subject}/versions/{version}
func (h *Handler) DeleteDEKVersion(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")
	algorithm := r.URL.Query().Get("algorithm")
	permanent := r.URL.Query().Get("permanent") == "true"

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion, "Invalid version: must be a positive integer")
		return
	}
	if version <= 0 {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion, "Invalid version: must be a positive integer")
		return
	}

	// Fetch before deletion for audit before_hash.
	existing, _ := h.registry.GetDEK(r.Context(), kekName, subject, version, algorithm, true)

	if err := h.registry.DeleteDEK(r.Context(), kekName, subject, version, algorithm, permanent); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeDEKNotFound, "Data encryption key not found")
			return
		}
		writeInternalError(w, err)
		return
	}

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "dek"
		hints.TargetID = kekName
		hints.Version = version
		if existing != nil {
			hints.BeforeHash = hashDEK(existing)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// UndeleteDEKVersion handles POST /dek-registry/v1/keks/{name}/deks/{subject}/versions/{version}/undelete
func (h *Handler) UndeleteDEKVersion(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")
	algorithm := r.URL.Query().Get("algorithm")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion, "Invalid version: must be a positive integer")
		return
	}
	if version <= 0 {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion, "Invalid version: must be a positive integer")
		return
	}

	// Fetch deleted state for before_hash.
	existing, _ := h.registry.GetDEK(r.Context(), kekName, subject, version, algorithm, true)

	if err := h.registry.UndeleteDEK(r.Context(), kekName, subject, version, algorithm); err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeDEKNotFound, "Data encryption key not found")
			return
		}
		writeInternalError(w, err)
		return
	}

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "dek"
		hints.TargetID = kekName
		hints.Version = version
		if existing != nil {
			hints.BeforeHash = hashDEK(existing)
		}
		if restored, err := h.registry.GetDEK(r.Context(), kekName, subject, version, algorithm, false); err == nil {
			hints.AfterHash = hashDEK(restored)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateDEKWithSubject handles POST /dek-registry/v1/keks/{name}/deks/{subject}
// This is Confluent's preferred form where the subject comes from the URL path.
// With ?rewrap=true, this triggers DEK rewrap (re-encryption under current KEK).
func (h *Handler) CreateDEKWithSubject(w http.ResponseWriter, r *http.Request) {
	kekName := chi.URLParam(r, "name")
	subject := chi.URLParam(r, "subject")

	if r.URL.Query().Get("rewrap") == "true" {
		h.rewrapDEK(w, r, kekName, subject)
		return
	}

	var req types.CreateDEKRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is valid — just use path params and defaults
		if err.Error() != "EOF" {
			writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
			return
		}
	}

	dek := &storage.DEKRecord{
		KEKName:              kekName,
		Subject:              subject,
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

	if hints := auth.GetAuditHints(r.Context()); hints != nil {
		hints.TargetType = "dek"
		hints.TargetID = kekName
		hints.AfterHash = hashDEK(dek)
	}

	writeJSON(w, http.StatusOK, dekToResponse(dek))
}

// rewrapDEK re-encrypts a DEK's key material under the current KEK key version.
func (h *Handler) rewrapDEK(w http.ResponseWriter, r *http.Request, kekName, subject string) {
	algorithm := r.URL.Query().Get("algorithm")

	dek, err := h.registry.RewrapDEK(r.Context(), kekName, subject, -1, algorithm)
	if err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+kekName)
			return
		}
		if errors.Is(err, storage.ErrDEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeDEKNotFound, "Data encryption key not found")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dekToGetResponse(dek))
}

// TestKEK handles POST /dek-registry/v1/keks/{name}/test
// Validates that the KEK's KMS credentials are valid by performing a round-trip encrypt/decrypt test.
func (h *Handler) TestKEK(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	kek, err := h.registry.GetKEK(r.Context(), name, false)
	if err != nil {
		if errors.Is(err, storage.ErrKEKNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeKEKNotFound, "Key encryption key not found: "+name)
			return
		}
		writeInternalError(w, err)
		return
	}

	if err := h.registry.TestKEK(r.Context(), kek); err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, "KMS connection test failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, kekToResponse(kek))
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

// dekToGetResponse is like dekToResponse but strips plaintext KeyMaterial.
// Plaintext key material MUST only be returned on create, never on retrieval.
func dekToGetResponse(dek *storage.DEKRecord) types.DEKResponse {
	return types.DEKResponse{
		KEKName:              dek.KEKName,
		Subject:              dek.Subject,
		Version:              dek.Version,
		Algorithm:            dek.Algorithm,
		EncryptedKeyMaterial: dek.EncryptedKeyMaterial,
		Ts:                   dek.Ts,
		Deleted:              dek.Deleted,
	}
}

// parsePaginationParams extracts offset and limit query parameters.
func parsePaginationParams(r *http.Request) (offset, limit int) {
	if v := r.URL.Query().Get("offset"); v != "" {
		offset, _ = strconv.Atoi(v)
		if offset < 0 {
			offset = 0
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, _ = strconv.Atoi(v)
	}
	return offset, limit
}

// applyStringPagination applies offset/limit pagination to a string slice.
func applyStringPagination(items []string, r *http.Request) []string {
	offset, limit := parsePaginationParams(r)
	if offset > 0 {
		if offset >= len(items) {
			return []string{}
		}
		items = items[offset:]
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}

// applyIntPagination applies offset/limit pagination to an int slice.
func applyIntPagination(items []int, r *http.Request) []int {
	offset, limit := parsePaginationParams(r)
	if offset > 0 {
		if offset >= len(items) {
			return []int{}
		}
		items = items[offset:]
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}
