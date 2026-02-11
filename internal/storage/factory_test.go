package storage

import (
	"testing"
)

func TestRegister_AndCreate(t *testing.T) {
	// Save original factories and restore after test
	origFactories := factories
	factories = make(map[StorageType]Factory)
	defer func() { factories = origFactories }()

	called := false
	mockFactory := func(config map[string]interface{}) (Storage, error) {
		called = true
		return nil, nil
	}

	Register("test-backend", mockFactory)

	_, err := Create("test-backend", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("factory function was not called")
	}
}

func TestCreate_UnknownType(t *testing.T) {
	origFactories := factories
	factories = make(map[StorageType]Factory)
	defer func() { factories = origFactories }()

	_, err := Create("nonexistent", nil)
	if err == nil {
		t.Error("expected error for unknown storage type")
	}
}

func TestSupportedTypes(t *testing.T) {
	origFactories := factories
	factories = make(map[StorageType]Factory)
	defer func() { factories = origFactories }()

	dummyFactory := func(config map[string]interface{}) (Storage, error) { return nil, nil }
	Register("type-a", dummyFactory)
	Register("type-b", dummyFactory)

	types := SupportedTypes()
	if len(types) != 2 {
		t.Errorf("expected 2 types, got %d", len(types))
	}

	typeSet := make(map[StorageType]bool)
	for _, tp := range types {
		typeSet[tp] = true
	}
	if !typeSet["type-a"] || !typeSet["type-b"] {
		t.Errorf("expected type-a and type-b in list, got %v", types)
	}
}

func TestIsSupported(t *testing.T) {
	origFactories := factories
	factories = make(map[StorageType]Factory)
	defer func() { factories = origFactories }()

	dummyFactory := func(config map[string]interface{}) (Storage, error) { return nil, nil }
	Register("supported", dummyFactory)

	if !IsSupported("supported") {
		t.Error("expected 'supported' to be supported")
	}
	if IsSupported("unsupported") {
		t.Error("expected 'unsupported' to not be supported")
	}
}

func TestSupportedTypes_Empty(t *testing.T) {
	origFactories := factories
	factories = make(map[StorageType]Factory)
	defer func() { factories = origFactories }()

	types := SupportedTypes()
	if len(types) != 0 {
		t.Errorf("expected 0 types, got %d", len(types))
	}
}

func TestRegister_Overwrite(t *testing.T) {
	origFactories := factories
	factories = make(map[StorageType]Factory)
	defer func() { factories = origFactories }()

	callCount := 0
	factory1 := func(config map[string]interface{}) (Storage, error) {
		callCount = 1
		return nil, nil
	}
	factory2 := func(config map[string]interface{}) (Storage, error) {
		callCount = 2
		return nil, nil
	}

	Register("test", factory1)
	Register("test", factory2) // Overwrite

	_, _ = Create("test", nil)
	if callCount != 2 {
		t.Errorf("expected factory2 to be called (callCount=2), got %d", callCount)
	}
}
