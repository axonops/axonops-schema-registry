package storage

import (
	"fmt"
)

// StorageType represents the type of storage backend.
type StorageType string

const (
	StorageTypeMemory   StorageType = "memory"
	StorageTypePostgres StorageType = "postgres"
	StorageTypeMySQL    StorageType = "mysql"
)

// Factory is a function type that creates a Storage instance.
type Factory func(config map[string]interface{}) (Storage, error)

// factories holds registered storage factories.
var factories = make(map[StorageType]Factory)

// Register registers a storage factory.
func Register(storageType StorageType, factory Factory) {
	factories[storageType] = factory
}

// Create creates a new Storage instance based on the storage type.
func Create(storageType StorageType, config map[string]interface{}) (Storage, error) {
	factory, ok := factories[storageType]
	if !ok {
		return nil, fmt.Errorf("unknown storage type: %s", storageType)
	}
	return factory(config)
}

// SupportedTypes returns a list of supported storage types.
func SupportedTypes() []StorageType {
	types := make([]StorageType, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}

// IsSupported returns true if the storage type is supported.
func IsSupported(storageType StorageType) bool {
	_, ok := factories[storageType]
	return ok
}
