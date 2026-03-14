package social

import "testing"

func TestEscapeMarkdownV2(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"hello", "hello"},
		{"10.50", "10\\.50"},
		{"a-b", "a\\-b"},
		{"a.b", "a\\.b"},
		{"(parens)", "\\(parens\\)"},
		{"[brackets]", "\\[brackets\\]"},
		{"0x1234…5678", "0x1234…5678"},
	}
	for _, tt := range tests {
		got := EscapeMarkdownV2(tt.in)
		if got != tt.want {
			t.Errorf("EscapeMarkdownV2(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
