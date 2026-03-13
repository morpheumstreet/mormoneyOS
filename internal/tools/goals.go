package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const goalsKey = "goals"
const maxGoals = 20
const maxGoalLen = 500

type goal struct {
	ID        string `json:"id"`
	Goal      string `json:"goal"`
	CreatedAt string `json:"created_at"`
	DoneAt    string `json:"done_at,omitempty"`
}

// SetGoalTool adds a goal.
type SetGoalTool struct {
	Store ToolStore
}

func (SetGoalTool) Name() string        { return "set_goal" }
func (SetGoalTool) Description() string { return "Set a goal to work toward." }
func (SetGoalTool) Parameters() string {
	return `{"type":"object","properties":{"goal":{"type":"string","description":"The goal"}},"required":["goal"]}`
}

func (t *SetGoalTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "set_goal requires store"}
	}
	g, _ := args["goal"].(string)
	g = strings.TrimSpace(g)
	if g == "" {
		return "", ErrInvalidArgs{Msg: "goal required"}
	}
	if len(g) > maxGoalLen {
		return fmt.Sprintf("Error: Goal exceeds %d character limit", maxGoalLen), nil
	}
	raw, _, _ := t.Store.GetKV(goalsKey)
	var goals []goal
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &goals)
	}
	if len(goals) >= maxGoals {
		return "Goal limit reached; complete some first.", nil
	}
	id := fmt.Sprintf("g%d", time.Now().UnixNano())
	goals = append(goals, goal{ID: id, Goal: g, CreatedAt: time.Now().Format(time.RFC3339)})
	b, _ := json.Marshal(goals)
	if err := t.Store.SetKV(goalsKey, string(b)); err != nil {
		return "", err
	}
	return fmt.Sprintf("Goal set (id=%s): %s", id, g), nil
}

// CompleteGoalTool marks a goal done.
type CompleteGoalTool struct {
	Store ToolStore
}

func (CompleteGoalTool) Name() string        { return "complete_goal" }
func (CompleteGoalTool) Description() string { return "Mark a goal as complete." }
func (CompleteGoalTool) Parameters() string {
	return `{"type":"object","properties":{"goal_id":{"type":"string","description":"Goal id from set_goal"}},"required":["goal_id"]}`
}

func (t *CompleteGoalTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "complete_goal requires store"}
	}
	id, _ := args["goal_id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return "", ErrInvalidArgs{Msg: "goal_id required"}
	}
	raw, _, _ := t.Store.GetKV(goalsKey)
	if raw == "" {
		return "No goals.", nil
	}
	var goals []goal
	if err := json.Unmarshal([]byte(raw), &goals); err != nil {
		return "Corrupt goals.", nil
	}
	found := false
	for i := range goals {
		if goals[i].ID == id && goals[i].DoneAt == "" {
			goals[i].DoneAt = time.Now().Format(time.RFC3339)
			found = true
			break
		}
	}
	if !found {
		return "Goal not found or already complete.", nil
	}
	b, _ := json.Marshal(goals)
	if err := t.Store.SetKV(goalsKey, string(b)); err != nil {
		return "", err
	}
	return "Goal marked complete.", nil
}
