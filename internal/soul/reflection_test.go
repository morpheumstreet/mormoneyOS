package soul

import (
	"path/filepath"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

func TestReflectOnSoul_NoSoul(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ref.db")
	db, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ref, err := ReflectOnSoul(db)
	if err != nil {
		t.Fatal(err)
	}
	if ref.CurrentAlignment != 0 || len(ref.SuggestedUpdates) != 0 {
		t.Errorf("expected empty reflection, got alignment=%.2f, suggested=%d", ref.CurrentAlignment, len(ref.SuggestedUpdates))
	}
}

func TestReflectOnSoul_WithSoulAndGenesis(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ref2.db")
	db, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	soulContent := `## Core Purpose
I am an autonomous agent that manages finances and executes tasks.
## Mission
Same as core purpose.`
	_ = db.SetKV("soul_content", soulContent)
	_ = db.SetKV("genesis_prompt", "I am an autonomous agent that manages finances and executes tasks.")

	ref, err := ReflectOnSoul(db)
	if err != nil {
		t.Fatal(err)
	}
	if ref.CurrentAlignment < 0.5 {
		t.Errorf("expected high alignment for matching purpose, got %.2f", ref.CurrentAlignment)
	}
}

func TestComputeGenesisAlignment(t *testing.T) {
	tests := []struct {
		purpose, genesis string
		wantMin          float64
	}{
		{"hello world", "hello world", 0.99},
		{"hello world", "hello", 0.5},
		{"x y z", "a b c", 0},
		{"", "genesis", 0},
		{"purpose", "", 0},
	}
	for _, tt := range tests {
		got := computeGenesisAlignment(tt.purpose, tt.genesis)
		if got < tt.wantMin {
			t.Errorf("computeGenesisAlignment(%q, %q) = %.2f, want >= %.2f", tt.purpose, tt.genesis, got, tt.wantMin)
		}
	}
}

func TestExtractCorePurpose(t *testing.T) {
	body := `## Core Purpose
I am a helpful agent.
## Values
- Honesty
`
	got := extractCorePurpose(body)
	if got != "I am a helpful agent." {
		t.Errorf("extractCorePurpose = %q", got)
	}
}

// Ensure *state.Database implements ReflectionStore.
var _ ReflectionStore = (*state.Database)(nil)
