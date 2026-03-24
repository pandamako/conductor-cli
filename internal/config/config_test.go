package config

import "testing"

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "main"},
		{"feature/login", "feature%2Flogin"},
		{"feature/deep/nested", "feature%2Fdeep%2Fnested"},
		{"has%percent", "has%25percent"},
		{"feature%2Flogin", "feature%252Flogin"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeBranchName(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestUnsanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "main"},
		{"feature%2Flogin", "feature/login"},
		{"feature%2Fdeep%2Fnested", "feature/deep/nested"},
		{"has%25percent", "has%percent"},
		{"feature%252Flogin", "feature%2Flogin"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := UnsanitizeBranchName(tt.input)
			if got != tt.expected {
				t.Errorf("UnsanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeRoundTrip(t *testing.T) {
	names := []string{"main", "feature/login", "a%b/c%2Fd", "deep/a/b/c"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			got := UnsanitizeBranchName(SanitizeBranchName(name))
			if got != name {
				t.Errorf("round-trip failed for %q: got %q", name, got)
			}
		})
	}
}
