package web

import "testing"

func TestSanitizeForStorage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"normal text", "Hello world", 100, "Hello world"},
		{"null bytes", "Hello\x00world\x00", 100, "Helloworld"},
		{"control chars", "Hello\x01\x02\x1fworld", 100, "Helloworld"},
		{"allows newline tab", "Line1\nLine2\tTab", 100, "Line1\nLine2\tTab"},
		{"truncates to max", "abcdefghij", 5, "abcde"},
		{"empty after trim", "   \x00\x01   ", 100, ""},
		{"replacement char", "Hello\ufffdWorld", 100, "HelloWorld"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeForStorage(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("sanitizeForStorage(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}
