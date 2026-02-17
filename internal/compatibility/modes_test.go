package compatibility

import (
	"testing"
)

func TestMode_IsValid(t *testing.T) {
	valid := []Mode{
		ModeNone, ModeBackward, ModeBackwardTransitive,
		ModeForward, ModeForwardTransitive,
		ModeFull, ModeFullTransitive,
	}
	for _, m := range valid {
		if !m.IsValid() {
			t.Errorf("expected %s to be valid", m)
		}
	}

	invalid := []Mode{"", "INVALID", "backward", "none"}
	for _, m := range invalid {
		if m.IsValid() {
			t.Errorf("expected %q to be invalid", m)
		}
	}
}

func TestMode_IsTransitive(t *testing.T) {
	transitive := []Mode{ModeBackwardTransitive, ModeForwardTransitive, ModeFullTransitive}
	for _, m := range transitive {
		if !m.IsTransitive() {
			t.Errorf("expected %s to be transitive", m)
		}
	}

	nonTransitive := []Mode{ModeNone, ModeBackward, ModeForward, ModeFull}
	for _, m := range nonTransitive {
		if m.IsTransitive() {
			t.Errorf("expected %s to not be transitive", m)
		}
	}
}

func TestMode_RequiresBackward(t *testing.T) {
	requiresBackward := []Mode{ModeBackward, ModeBackwardTransitive, ModeFull, ModeFullTransitive}
	for _, m := range requiresBackward {
		if !m.RequiresBackward() {
			t.Errorf("expected %s to require backward", m)
		}
	}

	noBackward := []Mode{ModeNone, ModeForward, ModeForwardTransitive}
	for _, m := range noBackward {
		if m.RequiresBackward() {
			t.Errorf("expected %s to not require backward", m)
		}
	}
}

func TestMode_RequiresForward(t *testing.T) {
	requiresForward := []Mode{ModeForward, ModeForwardTransitive, ModeFull, ModeFullTransitive}
	for _, m := range requiresForward {
		if !m.RequiresForward() {
			t.Errorf("expected %s to require forward", m)
		}
	}

	noForward := []Mode{ModeNone, ModeBackward, ModeBackwardTransitive}
	for _, m := range noForward {
		if m.RequiresForward() {
			t.Errorf("expected %s to not require forward", m)
		}
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		input string
		valid bool
		mode  Mode
	}{
		{"NONE", true, ModeNone},
		{"BACKWARD", true, ModeBackward},
		{"BACKWARD_TRANSITIVE", true, ModeBackwardTransitive},
		{"FORWARD", true, ModeForward},
		{"FORWARD_TRANSITIVE", true, ModeForwardTransitive},
		{"FULL", true, ModeFull},
		{"FULL_TRANSITIVE", true, ModeFullTransitive},
		{"INVALID", false, "INVALID"},
		{"", false, ""},
		{"backward", false, "backward"},
	}

	for _, tt := range tests {
		mode, ok := ParseMode(tt.input)
		if ok != tt.valid {
			t.Errorf("ParseMode(%q): valid=%v, want %v", tt.input, ok, tt.valid)
		}
		if mode != tt.mode {
			t.Errorf("ParseMode(%q): mode=%v, want %v", tt.input, mode, tt.mode)
		}
	}
}

func TestChecker_NoneMode(t *testing.T) {
	c := NewChecker()
	result := c.Check(ModeNone, "AVRO", SchemaWithRefs{Schema: "anything"}, []SchemaWithRefs{{Schema: "old"}})
	if !result.IsCompatible {
		t.Error("NONE mode should always be compatible")
	}
}

func TestChecker_NoExistingSchemas(t *testing.T) {
	c := NewChecker()
	result := c.Check(ModeBackward, "AVRO", SchemaWithRefs{Schema: "new"}, nil)
	if !result.IsCompatible {
		t.Error("no existing schemas should be compatible")
	}
}

func TestChecker_UnregisteredType(t *testing.T) {
	c := NewChecker()
	result := c.Check(ModeBackward, "UNKNOWN", SchemaWithRefs{Schema: "new"}, []SchemaWithRefs{{Schema: "old"}})
	if result.IsCompatible {
		t.Error("unregistered type should be incompatible")
	}
	if len(result.Messages) == 0 {
		t.Error("expected error message")
	}
}

func TestChecker_CheckPair(t *testing.T) {
	c := NewChecker()
	// Without a registered checker, it should fail
	result := c.CheckPair(ModeBackward, "UNKNOWN", SchemaWithRefs{Schema: "new"}, SchemaWithRefs{Schema: "old"})
	if result.IsCompatible {
		t.Error("unregistered type should be incompatible")
	}
}

func TestChecker_CheckPair_NoneMode(t *testing.T) {
	c := NewChecker()
	result := c.CheckPair(ModeNone, "ANY", SchemaWithRefs{Schema: "new"}, SchemaWithRefs{Schema: "old"})
	if !result.IsCompatible {
		t.Error("NONE mode should always be compatible via CheckPair")
	}
}
