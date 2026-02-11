package compatibility

import (
	"testing"
)

func TestNewCompatibleResult(t *testing.T) {
	r := NewCompatibleResult()
	if !r.IsCompatible {
		t.Error("expected compatible result")
	}
	if len(r.Messages) != 0 {
		t.Errorf("expected no messages, got %d", len(r.Messages))
	}
}

func TestNewIncompatibleResult(t *testing.T) {
	r := NewIncompatibleResult("field removed", "type changed")
	if r.IsCompatible {
		t.Error("expected incompatible result")
	}
	if len(r.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(r.Messages))
	}
	if r.Messages[0] != "field removed" {
		t.Errorf("expected 'field removed', got %q", r.Messages[0])
	}
	if r.Messages[1] != "type changed" {
		t.Errorf("expected 'type changed', got %q", r.Messages[1])
	}
}

func TestNewIncompatibleResult_NoMessages(t *testing.T) {
	r := NewIncompatibleResult()
	if r.IsCompatible {
		t.Error("expected incompatible result")
	}
	if len(r.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(r.Messages))
	}
}

func TestNewIncompatibleResult_SingleMessage(t *testing.T) {
	r := NewIncompatibleResult("breaking change")
	if r.IsCompatible {
		t.Error("expected incompatible result")
	}
	if len(r.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(r.Messages))
	}
}

func TestAddMessage(t *testing.T) {
	r := NewCompatibleResult()
	r.AddMessage("field %s removed from %s", "age", "User")

	if r.IsCompatible {
		t.Error("expected incompatible after AddMessage")
	}
	if len(r.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(r.Messages))
	}
	if r.Messages[0] != "field age removed from User" {
		t.Errorf("unexpected message: %q", r.Messages[0])
	}
}

func TestAddMessage_Multiple(t *testing.T) {
	r := NewCompatibleResult()
	r.AddMessage("issue 1")
	r.AddMessage("issue 2")
	r.AddMessage("issue 3")

	if r.IsCompatible {
		t.Error("expected incompatible")
	}
	if len(r.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(r.Messages))
	}
}

func TestMerge_IncompatibleIntoCompatible(t *testing.T) {
	r := NewCompatibleResult()
	other := NewIncompatibleResult("problem")

	r.Merge(other)

	if r.IsCompatible {
		t.Error("expected incompatible after merging incompatible result")
	}
	if len(r.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(r.Messages))
	}
}

func TestMerge_CompatibleIntoCompatible(t *testing.T) {
	r := NewCompatibleResult()
	other := NewCompatibleResult()

	r.Merge(other)

	if !r.IsCompatible {
		t.Error("expected compatible after merging compatible result")
	}
	if len(r.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(r.Messages))
	}
}

func TestMerge_CompatibleIntoIncompatible(t *testing.T) {
	r := NewIncompatibleResult("existing issue")
	other := NewCompatibleResult()

	r.Merge(other)

	if r.IsCompatible {
		t.Error("expected still incompatible")
	}
	if len(r.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(r.Messages))
	}
}

func TestMerge_MultipleMessages(t *testing.T) {
	r := NewIncompatibleResult("issue 1")
	other := NewIncompatibleResult("issue 2", "issue 3")

	r.Merge(other)

	if r.IsCompatible {
		t.Error("expected incompatible")
	}
	if len(r.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(r.Messages))
	}
}
