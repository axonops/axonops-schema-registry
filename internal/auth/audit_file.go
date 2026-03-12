// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"fmt"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// FileOutput writes audit events to a file with automatic rotation.
// It wraps lumberjack.Logger for size-based rotation with configurable
// backup count, max age, and optional compression.
type FileOutput struct {
	logger *lumberjack.Logger
}

// NewFileOutput creates a file audit output with rotation from config.
// Defaults: 100MB max size, 5 backups, 30 days max age, compression on.
func NewFileOutput(cfg config.AuditFileConfig) (*FileOutput, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("audit file path is required")
	}

	maxSize := cfg.MaxSizeMB
	if maxSize <= 0 {
		maxSize = 100
	}

	maxBackups := cfg.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 5
	}

	maxAge := cfg.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 30
	}

	compress := true
	if cfg.Compress != nil {
		compress = *cfg.Compress
	}

	return &FileOutput{
		logger: &lumberjack.Logger{
			Filename:   cfg.Path,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
			LocalTime:  false,
		},
	}, nil
}

// Write writes data to the rotating log file.
func (o *FileOutput) Write(data []byte) error {
	_, err := o.logger.Write(data)
	return err
}

// Close closes the file output.
func (o *FileOutput) Close() error {
	return o.logger.Close()
}

// Name returns "file".
func (o *FileOutput) Name() string { return "file" }
