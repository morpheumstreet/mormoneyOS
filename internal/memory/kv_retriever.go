package memory

import (
	"context"
	"encoding/json"
	"strings"
)

// KVReader provides read-only KV access for memory retrieval.
type KVReader interface {
	GetKV(key string) (string, bool, error)
	ListKeysWithPrefix(prefix string) ([]string, error)
}

const (
	memoryFactsKey  = "memory_facts"
	goalsKey        = "goals"
	procedurePrefix = "procedure:"
)

const (
	maxFacts      = 50
	maxGoals      = 20
	maxProcedures = 20
)

// KVMemoryRetriever retrieves facts, goals, and procedures from KV store (Phase 1).
type KVMemoryRetriever struct {
	kv KVReader
}

// NewKVMemoryRetriever creates a retriever that reads from the given KV store.
func NewKVMemoryRetriever(kv KVReader) *KVMemoryRetriever {
	return &KVMemoryRetriever{kv: kv}
}

// Retrieve fetches facts, pending goals, and procedures; formats and returns the block.
// sessionID is reserved for Phase 2; Phase 1 ignores it.
func (r *KVMemoryRetriever) Retrieve(ctx context.Context, sessionID string, currentInput string) (string, error) {
	_ = sessionID
	_ = currentInput
	block := &MemoryBlock{}

	// Facts
	if raw, ok, err := r.kv.GetKV(memoryFactsKey); err == nil && ok && raw != "" {
		var facts []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		}
		if json.Unmarshal([]byte(raw), &facts) == nil {
			for i, f := range facts {
				if i >= maxFacts {
					break
				}
				if c := strings.TrimSpace(f.Content); c != "" {
					block.Facts = append(block.Facts, c)
				}
			}
		}
	}

	// Goals (pending only)
	if raw, ok, err := r.kv.GetKV(goalsKey); err == nil && ok && raw != "" {
		var goals []struct {
			ID     string `json:"id"`
			Goal   string `json:"goal"`
			DoneAt string `json:"done_at,omitempty"`
		}
		if json.Unmarshal([]byte(raw), &goals) == nil {
			for i, g := range goals {
				if i >= maxGoals {
					break
				}
				if g.DoneAt == "" {
					if goal := strings.TrimSpace(g.Goal); goal != "" {
						block.Goals = append(block.Goals, goal)
					}
				}
			}
		}
	}

	// Procedures
	if keys, err := r.kv.ListKeysWithPrefix(procedurePrefix); err == nil {
		for i, key := range keys {
			if i >= maxProcedures {
				break
			}
			name := strings.TrimPrefix(key, procedurePrefix)
			if name == "" {
				continue
			}
			val, ok, err := r.kv.GetKV(key)
			if err != nil || !ok || val == "" {
				continue
			}
			steps := countSteps(val)
			block.Procedures = append(block.Procedures, ProcedureEntry{Name: name, Steps: steps})
		}
	}

	return FormatMemoryBlock(block), nil
}

func countSteps(steps string) int {
	n := 0
	for _, line := range strings.Split(steps, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	if n == 0 && strings.TrimSpace(steps) != "" {
		return 1
	}
	return n
}
