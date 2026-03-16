package memory

import (
	"encoding/json"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

func TestParseExtraction(t *testing.T) {
	valid := `{"facts":[{"category":"env","key":"provider","value":"conway","confidence":0.9}],"episodes":[{"event_type":"event","summary":"Checked balance","outcome":"success","importance":0.6}],"procedures":[],"relationships":[],"importance":0.7}`
	ext, imp := parseExtraction(valid)
	if ext == nil {
		t.Fatal("expected non-nil extraction")
	}
	if imp != 0.7 {
		t.Errorf("importance = %v, want 0.7", imp)
	}
	if len(ext.Facts) != 1 {
		t.Errorf("facts len = %d, want 1", len(ext.Facts))
	}
	if ext.Facts[0].Category != "env" || ext.Facts[0].Key != "provider" || ext.Facts[0].Value != "conway" {
		t.Errorf("fact = %+v", ext.Facts[0])
	}
	if len(ext.Episodes) != 1 {
		t.Errorf("episodes len = %d, want 1", len(ext.Episodes))
	}
	if ext.Episodes[0].Summary != "Checked balance" {
		t.Errorf("episode summary = %q", ext.Episodes[0].Summary)
	}
}

func TestParseExtraction_WithMarkdown(t *testing.T) {
	wrapped := "```json\n{\"facts\":[],\"episodes\":[],\"procedures\":[],\"relationships\":[],\"importance\":0.5}\n```"
	ext, imp := parseExtraction(wrapped)
	if ext == nil {
		t.Fatal("expected non-nil extraction")
	}
	if imp != 0.5 {
		t.Errorf("importance = %v, want 0.5", imp)
	}
}

func TestInsertIngestCandidate_Integration(t *testing.T) {
	db, err := state.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ext := &Extraction{
		Facts:      []Fact{{Category: "test", Key: "k1", Value: "v1", Confidence: 0.9}},
		Importance: 0.6,
	}
	extJSON, _ := json.Marshal(ext)
	if err := db.InsertIngestCandidate("s1", "turn-1", string(extJSON), 0.6); err != nil {
		t.Fatal(err)
	}
	candidates, err := db.GetUnprocessedIngestCandidates(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates len = %d, want 1", len(candidates))
	}
	if candidates[0].SessionID != "s1" || candidates[0].TurnID != "turn-1" {
		t.Errorf("candidate = %+v", candidates[0])
	}
	if err := db.MarkIngestCandidatesProcessed([]int64{candidates[0].ID}); err != nil {
		t.Fatal(err)
	}
	candidates2, _ := db.GetUnprocessedIngestCandidates(10)
	if len(candidates2) != 0 {
		t.Errorf("after mark: %d unprocessed", len(candidates2))
	}
}
