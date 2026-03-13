package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const memoryFactsKey = "memory_facts"
const maxFacts = 100
const maxFactLen = 2000

type memoryFact struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	At      string `json:"at"`
}

// RememberFactTool stores a fact in memory.
type RememberFactTool struct {
	Store ToolStore
}

func (RememberFactTool) Name() string        { return "remember_fact" }
func (RememberFactTool) Description() string { return "Remember a fact for later recall." }
func (RememberFactTool) Parameters() string {
	return `{"type":"object","properties":{"content":{"type":"string","description":"The fact to remember"}},"required":["content"]}`
}

func (t *RememberFactTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "remember_fact requires store"}
	}
	content, _ := args["content"].(string)
	content = strings.TrimSpace(content)
	if content == "" {
		return "", ErrInvalidArgs{Msg: "content required"}
	}
	if len(content) > maxFactLen {
		return fmt.Sprintf("Error: Fact exceeds %d character limit", maxFactLen), nil
	}
	raw, _, _ := t.Store.GetKV(memoryFactsKey)
	var facts []memoryFact
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &facts)
	}
	if len(facts) >= maxFacts {
		return "Memory limit reached; forget some facts first.", nil
	}
	id := fmt.Sprintf("f%d", time.Now().UnixNano())
	facts = append(facts, memoryFact{ID: id, Content: content, At: time.Now().Format(time.RFC3339)})
	b, _ := json.Marshal(facts)
	if err := t.Store.SetKV(memoryFactsKey, string(b)); err != nil {
		return "", err
	}
	return fmt.Sprintf("Remembered fact (id=%s).", id), nil
}

// RecallFactsTool recalls facts matching a query.
type RecallFactsTool struct {
	Store ToolStore
}

func (RecallFactsTool) Name() string        { return "recall_facts" }
func (RecallFactsTool) Description() string { return "Recall facts matching a query." }
func (RecallFactsTool) Parameters() string {
	return `{"type":"object","properties":{"query":{"type":"string","description":"Search query (substring match)"}},"required":["query"]}`
}

func (t *RecallFactsTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "recall_facts requires store"}
	}
	query, _ := args["query"].(string)
	query = strings.TrimSpace(strings.ToLower(query))
	raw, _, _ := t.Store.GetKV(memoryFactsKey)
	if raw == "" {
		return "No facts stored.", nil
	}
	var facts []memoryFact
	if err := json.Unmarshal([]byte(raw), &facts); err != nil {
		return "Corrupt memory.", nil
	}
	var out []string
	for _, f := range facts {
		if query == "" || strings.Contains(strings.ToLower(f.Content), query) {
			out = append(out, fmt.Sprintf("[%s] %s", f.ID, f.Content))
		}
	}
	if len(out) == 0 {
		return "No matching facts.", nil
	}
	return strings.Join(out, "\n"), nil
}

// ForgetTool removes a fact by id.
type ForgetTool struct {
	Store ToolStore
}

func (ForgetTool) Name() string        { return "forget" }
func (ForgetTool) Description() string { return "Forget a memory by id." }
func (ForgetTool) Parameters() string {
	return `{"type":"object","properties":{"id":{"type":"string","description":"Memory id from recall_facts"}},"required":["id"]}`
}

func (t *ForgetTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "forget requires store"}
	}
	id, _ := args["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return "", ErrInvalidArgs{Msg: "id required"}
	}
	raw, _, _ := t.Store.GetKV(memoryFactsKey)
	if raw == "" {
		return "No facts to forget.", nil
	}
	var facts []memoryFact
	if err := json.Unmarshal([]byte(raw), &facts); err != nil {
		return "Corrupt memory.", nil
	}
	var kept []memoryFact
	for _, f := range facts {
		if f.ID != id {
			kept = append(kept, f)
		}
	}
	if len(kept) == len(facts) {
		return "No fact with that id.", nil
	}
	b, _ := json.Marshal(kept)
	if err := t.Store.SetKV(memoryFactsKey, string(b)); err != nil {
		return "", err
	}
	return "Forgotten.", nil
}
