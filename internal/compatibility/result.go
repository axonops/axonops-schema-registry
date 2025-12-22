package compatibility

import "fmt"

// Result represents the result of a compatibility check.
type Result struct {
	IsCompatible bool     `json:"is_compatible"`
	Messages     []string `json:"messages,omitempty"`
}

// NewCompatibleResult creates a result indicating compatibility.
func NewCompatibleResult() *Result {
	return &Result{
		IsCompatible: true,
	}
}

// NewIncompatibleResult creates a result indicating incompatibility.
func NewIncompatibleResult(messages ...string) *Result {
	return &Result{
		IsCompatible: false,
		Messages:     messages,
	}
}

// AddMessage adds an incompatibility message.
func (r *Result) AddMessage(format string, args ...interface{}) {
	r.Messages = append(r.Messages, fmt.Sprintf(format, args...))
	r.IsCompatible = false
}

// Merge merges another result into this one.
func (r *Result) Merge(other *Result) {
	if !other.IsCompatible {
		r.IsCompatible = false
		r.Messages = append(r.Messages, other.Messages...)
	}
}
