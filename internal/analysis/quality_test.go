package analysis

import "testing"

func TestIsGoodFieldName(t *testing.T) {
	tests := []struct {
		name string
		good bool
	}{
		{"id", true},
		{"user_name", true},
		{"user_id", true},
		{"UserName", false},
		{"user-name", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isGoodFieldName(tc.name); got != tc.good {
				t.Errorf("isGoodFieldName(%q) = %v, want %v", tc.name, got, tc.good)
			}
		})
	}
}
