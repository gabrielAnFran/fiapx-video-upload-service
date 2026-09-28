package entities

import "testing"

func TestIsValidVideoStatus(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"uploaded", "UPLOADED", true},
		{"processing", "PROCESSING", true},
		{"completed", "COMPLETED", true},
		{"failed", "FAILED", true},
		{"unknown", "BOGUS", false},
		{"empty", "", false},
		{"lowercase", "uploaded", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidVideoStatus(tt.in); got != tt.want {
				t.Errorf("IsValidVideoStatus(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
