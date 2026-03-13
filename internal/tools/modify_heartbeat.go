package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// HeartbeatScheduleStore provides DB access for heartbeat schedule (TS-aligned).
type HeartbeatScheduleStore interface {
	GetHeartbeatSchedule() ([]state.HeartbeatScheduleRow, error)
	UpsertHeartbeatSchedule(row state.HeartbeatScheduleRow) error
}

// ModifyHeartbeatTool modifies a heartbeat task's schedule or enabled state.
type ModifyHeartbeatTool struct {
	Store interface {
		GetHeartbeatSchedule() ([]state.HeartbeatScheduleRow, error)
		UpsertHeartbeatSchedule(row state.HeartbeatScheduleRow) error
	}
}

func (ModifyHeartbeatTool) Name() string        { return "modify_heartbeat" }
func (ModifyHeartbeatTool) Description() string { return "Modify heartbeat task schedule or enabled state." }
func (ModifyHeartbeatTool) Parameters() string {
	return `{"type":"object","properties":{"task":{"type":"string","description":"Task name"},"schedule":{"type":"string","description":"Cron expression"},"enabled":{"type":"boolean","description":"Enable or disable"}},"required":["task"]}`
}

func (t *ModifyHeartbeatTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "modify_heartbeat requires store with heartbeat_schedule"}
	}
	task, _ := args["task"].(string)
	task = strings.TrimSpace(task)
	if task == "" {
		return "", ErrInvalidArgs{Msg: "task required"}
	}
	rows, err := t.Store.GetHeartbeatSchedule()
	if err != nil {
		return "", fmt.Errorf("get schedule: %w", err)
	}
	var found *state.HeartbeatScheduleRow
	for i := range rows {
		if rows[i].Name == task || rows[i].Task == task {
			found = &rows[i]
			break
		}
	}
	if found == nil {
		return fmt.Sprintf("Task %q not found in heartbeat schedule. Available: %s",
			task, listTaskNames(rows)), nil
	}
	row := *found
	if s, ok := args["schedule"].(string); ok && s != "" {
		row.Schedule = strings.TrimSpace(s)
	}
	if b, ok := args["enabled"].(bool); ok {
		if b {
			row.Enabled = 1
		} else {
			row.Enabled = 0
		}
	}
	if err := t.Store.UpsertHeartbeatSchedule(row); err != nil {
		return "", fmt.Errorf("upsert: %w", err)
	}
	return fmt.Sprintf("Updated %s: schedule=%s enabled=%v", task, row.Schedule, row.Enabled == 1), nil
}

func listTaskNames(rows []state.HeartbeatScheduleRow) string {
	var names []string
	for _, r := range rows {
		names = append(names, r.Name)
	}
	return strings.Join(names, ", ")
}
