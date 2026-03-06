// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"crypto/subtle"
	"strings"
)

// constantTimeEqualFold performs a constant-time case-insensitive string comparison.
// Unlike strings.EqualFold, this does not leak timing information about the
// position of the first differing character, preventing timing side-channel attacks
// on security-sensitive values like role names and group DNs.
func constantTimeEqualFold(a, b string) bool {
	la := strings.ToLower(a)
	lb := strings.ToLower(b)
	if len(la) != len(lb) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(la), []byte(lb)) == 1
}
