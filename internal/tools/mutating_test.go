package tools

import "testing"

func TestIsMutatingTool(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"sleep", true},
		{"shell", true},
		{"write_file", true},
		{"remember_fact", true},
		{"check_credits", false},
		{"list_skills", false},
		{"recall_facts", false},
		{"unknown_tool", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMutatingTool(tt.name); got != tt.want {
				t.Errorf("IsMutatingTool(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
