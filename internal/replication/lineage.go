package replication

import (
	"strings"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// LineageStore provides children for lineage summary.
type LineageStore interface {
	GetAllChildren() ([]state.Child, bool)
}

// LineageEntry represents one child in lineage (TS-aligned).
type LineageEntry struct {
	ID     string
	Name   string
	Status string
}

// GetLineage returns lineage entries for display (alive children, excludes dead/cleaned_up).
func GetLineage(store LineageStore) []LineageEntry {
	if store == nil {
		return nil
	}
	children, ok := store.GetAllChildren()
	if !ok || len(children) == 0 {
		return nil
	}
	var out []LineageEntry
	for _, c := range children {
		if c.Status == StateDead || c.Status == StateCleanedUp {
			continue
		}
		out = append(out, LineageEntry{ID: c.ID, Name: c.Name, Status: c.Status})
	}
	return out
}

// GetLineageSummary returns a short text summary for system prompt (TS getLineageSummary-aligned).
func GetLineageSummary(store LineageStore) string {
	entries := GetLineage(store)
	if len(entries) == 0 {
		return "No children."
	}
	var b strings.Builder
	b.WriteString("Children: ")
	for i, e := range entries {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(e.Name)
		b.WriteString(" (")
		b.WriteString(e.Status)
		b.WriteString(")")
	}
	b.WriteString(".")
	return b.String()
}

// PruneStore provides children and status updates for marking stale children dead.
type PruneStore interface {
	GetAllChildren() ([]state.Child, bool)
	UpdateChildStatus(id, status string) error
}

// PruneDeadChildren marks children as dead when last_checked exceeds StaleThreshold.
// Does not delete sandboxes; use SandboxCleanup.PruneDead for that.
func PruneDeadChildren(store PruneStore) int {
	if store == nil {
		return 0
	}
	children, ok := store.GetAllChildren()
	if !ok {
		return 0
	}
	pruned := 0
	for _, c := range children {
		if c.Status == StateDead || c.Status == StateCleanedUp {
			continue
		}
		if c.LastChecked == "" {
			continue
		}
		t, err := parseTime(c.LastChecked)
		if err != nil {
			continue
		}
		if time.Since(t) > StaleThreshold {
			if store.UpdateChildStatus(c.ID, StateDead) == nil {
				pruned++
			}
		}
	}
	return pruned
}

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", s)
}
