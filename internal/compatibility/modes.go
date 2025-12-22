// Package compatibility provides schema compatibility checking.
package compatibility

// Mode represents a compatibility mode.
type Mode string

const (
	// ModeNone disables compatibility checking.
	ModeNone Mode = "NONE"

	// ModeBackward means new schema can read data written by old schema.
	// New schema is the "reader", old schema is the "writer".
	ModeBackward Mode = "BACKWARD"

	// ModeBackwardTransitive means new schema can read data written by ALL previous schemas.
	ModeBackwardTransitive Mode = "BACKWARD_TRANSITIVE"

	// ModeForward means old schema can read data written by new schema.
	// Old schema is the "reader", new schema is the "writer".
	ModeForward Mode = "FORWARD"

	// ModeForwardTransitive means ALL previous schemas can read data written by new schema.
	ModeForwardTransitive Mode = "FORWARD_TRANSITIVE"

	// ModeFull means both backward and forward compatible.
	ModeFull Mode = "FULL"

	// ModeFullTransitive means full compatibility with ALL previous schemas.
	ModeFullTransitive Mode = "FULL_TRANSITIVE"
)

// IsValid returns true if the mode is valid.
func (m Mode) IsValid() bool {
	switch m {
	case ModeNone, ModeBackward, ModeBackwardTransitive,
		ModeForward, ModeForwardTransitive, ModeFull, ModeFullTransitive:
		return true
	default:
		return false
	}
}

// IsTransitive returns true if the mode requires checking against all versions.
func (m Mode) IsTransitive() bool {
	switch m {
	case ModeBackwardTransitive, ModeForwardTransitive, ModeFullTransitive:
		return true
	default:
		return false
	}
}

// RequiresBackward returns true if backward compatibility is required.
func (m Mode) RequiresBackward() bool {
	switch m {
	case ModeBackward, ModeBackwardTransitive, ModeFull, ModeFullTransitive:
		return true
	default:
		return false
	}
}

// RequiresForward returns true if forward compatibility is required.
func (m Mode) RequiresForward() bool {
	switch m {
	case ModeForward, ModeForwardTransitive, ModeFull, ModeFullTransitive:
		return true
	default:
		return false
	}
}

// ParseMode parses a string into a Mode.
func ParseMode(s string) (Mode, bool) {
	mode := Mode(s)
	return mode, mode.IsValid()
}
