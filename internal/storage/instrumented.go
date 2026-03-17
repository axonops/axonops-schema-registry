package storage

import (
	"context"
	"time"
)

// MetricsRecorder is the interface that the instrumented storage wrapper uses
// to record storage operation metrics. This avoids a circular import between
// the storage and metrics packages.
type MetricsRecorder interface {
	RecordStorageOperation(backend, operation string, duration time.Duration, err error)
}

// InstrumentedStorage wraps a Storage implementation and records metrics
// for each storage operation using the provided MetricsRecorder.
type InstrumentedStorage struct {
	Storage
	backend  string
	recorder MetricsRecorder
}

// NewInstrumentedStorage creates a new InstrumentedStorage that wraps the given
// storage backend and records operation metrics via the recorder.
func NewInstrumentedStorage(store Storage, backend string, recorder MetricsRecorder) *InstrumentedStorage {
	return &InstrumentedStorage{
		Storage:  store,
		backend:  backend,
		recorder: recorder,
	}
}

// record is a helper that records a storage operation's duration and error.
func (s *InstrumentedStorage) record(operation string, start time.Time, err error) {
	s.recorder.RecordStorageOperation(s.backend, operation, time.Since(start), err)
}

// --- Schema operations ---

func (s *InstrumentedStorage) CreateSchema(ctx context.Context, registryCtx string, record *SchemaRecord) error {
	start := time.Now()
	err := s.Storage.CreateSchema(ctx, registryCtx, record)
	s.record("create_schema", start, err)
	return err
}

func (s *InstrumentedStorage) GetSchemaByID(ctx context.Context, registryCtx string, id int64) (*SchemaRecord, error) {
	start := time.Now()
	rec, err := s.Storage.GetSchemaByID(ctx, registryCtx, id)
	s.record("get_schema_by_id", start, err)
	return rec, err
}

func (s *InstrumentedStorage) GetSchemaBySubjectVersion(ctx context.Context, registryCtx string, subject string, version int) (*SchemaRecord, error) {
	start := time.Now()
	rec, err := s.Storage.GetSchemaBySubjectVersion(ctx, registryCtx, subject, version)
	s.record("get_schema_by_subject_version", start, err)
	return rec, err
}

func (s *InstrumentedStorage) GetSchemasBySubject(ctx context.Context, registryCtx string, subject string, includeDeleted bool) ([]*SchemaRecord, error) {
	start := time.Now()
	recs, err := s.Storage.GetSchemasBySubject(ctx, registryCtx, subject, includeDeleted)
	s.record("get_schemas_by_subject", start, err)
	return recs, err
}

func (s *InstrumentedStorage) GetSchemaByFingerprint(ctx context.Context, registryCtx string, subject, fingerprint string, includeDeleted bool) (*SchemaRecord, error) {
	start := time.Now()
	rec, err := s.Storage.GetSchemaByFingerprint(ctx, registryCtx, subject, fingerprint, includeDeleted)
	s.record("get_schema_by_fingerprint", start, err)
	return rec, err
}

func (s *InstrumentedStorage) GetSchemaByGlobalFingerprint(ctx context.Context, registryCtx string, fingerprint string) (*SchemaRecord, error) {
	start := time.Now()
	rec, err := s.Storage.GetSchemaByGlobalFingerprint(ctx, registryCtx, fingerprint)
	s.record("get_schema_by_global_fingerprint", start, err)
	return rec, err
}

func (s *InstrumentedStorage) GetLatestSchema(ctx context.Context, registryCtx string, subject string) (*SchemaRecord, error) {
	start := time.Now()
	rec, err := s.Storage.GetLatestSchema(ctx, registryCtx, subject)
	s.record("get_latest_schema", start, err)
	return rec, err
}

func (s *InstrumentedStorage) DeleteSchema(ctx context.Context, registryCtx string, subject string, version int, permanent bool) error {
	start := time.Now()
	err := s.Storage.DeleteSchema(ctx, registryCtx, subject, version, permanent)
	s.record("delete_schema", start, err)
	return err
}

// --- Subject operations ---

func (s *InstrumentedStorage) ListSubjects(ctx context.Context, registryCtx string, includeDeleted bool) ([]string, error) {
	start := time.Now()
	subjects, err := s.Storage.ListSubjects(ctx, registryCtx, includeDeleted)
	s.record("list_subjects", start, err)
	return subjects, err
}

func (s *InstrumentedStorage) DeleteSubject(ctx context.Context, registryCtx string, subject string, permanent bool) ([]int, error) {
	start := time.Now()
	versions, err := s.Storage.DeleteSubject(ctx, registryCtx, subject, permanent)
	s.record("delete_subject", start, err)
	return versions, err
}

func (s *InstrumentedStorage) SubjectExists(ctx context.Context, registryCtx string, subject string) (bool, error) {
	start := time.Now()
	exists, err := s.Storage.SubjectExists(ctx, registryCtx, subject)
	s.record("subject_exists", start, err)
	return exists, err
}

// --- Config operations ---

func (s *InstrumentedStorage) GetConfig(ctx context.Context, registryCtx string, subject string) (*ConfigRecord, error) {
	start := time.Now()
	cfg, err := s.Storage.GetConfig(ctx, registryCtx, subject)
	s.record("get_config", start, err)
	return cfg, err
}

func (s *InstrumentedStorage) SetConfig(ctx context.Context, registryCtx string, subject string, config *ConfigRecord) error {
	start := time.Now()
	err := s.Storage.SetConfig(ctx, registryCtx, subject, config)
	s.record("set_config", start, err)
	return err
}

func (s *InstrumentedStorage) GetGlobalConfig(ctx context.Context, registryCtx string) (*ConfigRecord, error) {
	start := time.Now()
	cfg, err := s.Storage.GetGlobalConfig(ctx, registryCtx)
	s.record("get_global_config", start, err)
	return cfg, err
}

func (s *InstrumentedStorage) SetGlobalConfig(ctx context.Context, registryCtx string, config *ConfigRecord) error {
	start := time.Now()
	err := s.Storage.SetGlobalConfig(ctx, registryCtx, config)
	s.record("set_global_config", start, err)
	return err
}

// --- ID operations ---

func (s *InstrumentedStorage) NextID(ctx context.Context, registryCtx string) (int64, error) {
	start := time.Now()
	id, err := s.Storage.NextID(ctx, registryCtx)
	s.record("next_id", start, err)
	return id, err
}

// --- Import operations ---

func (s *InstrumentedStorage) ImportSchema(ctx context.Context, registryCtx string, record *SchemaRecord) error {
	start := time.Now()
	err := s.Storage.ImportSchema(ctx, registryCtx, record)
	s.record("import_schema", start, err)
	return err
}

// --- Schema listing ---

func (s *InstrumentedStorage) ListSchemas(ctx context.Context, registryCtx string, params *ListSchemasParams) ([]*SchemaRecord, error) {
	start := time.Now()
	recs, err := s.Storage.ListSchemas(ctx, registryCtx, params)
	s.record("list_schemas", start, err)
	return recs, err
}

// --- Lifecycle ---

func (s *InstrumentedStorage) IsHealthy(ctx context.Context) bool {
	start := time.Now()
	healthy := s.Storage.IsHealthy(ctx)
	var err error
	if !healthy {
		err = ErrNotFound // Use a sentinel to signal unhealthy
	}
	s.record("health_check", start, err)
	return healthy
}
