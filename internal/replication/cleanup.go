package replication

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// SandboxDeleter deletes Conway sandboxes.
type SandboxDeleter interface {
	DeleteSandbox(ctx context.Context, sandboxID string) error
}

// CleanupStore provides children and deletion.
type CleanupStore interface {
	GetAllChildren() ([]state.Child, bool)
	UpdateChildStatus(id, status string) error
	DeleteChild(childID string) error
}

// SandboxCleanup deletes Conway sandboxes for dead/cleaned children (TS SandboxCleanup-aligned).
type SandboxCleanup struct {
	Conway SandboxDeleter
	Store  CleanupStore
}

// PruneDead deletes sandboxes for children in dead/failed/cleaned_up and removes them from DB.
func (c *SandboxCleanup) PruneDead(ctx context.Context) (pruned int, err error) {
	if c.Conway == nil || c.Store == nil {
		return 0, nil
	}
	children, ok := c.Store.GetAllChildren()
	if !ok {
		return 0, nil
	}
	for _, child := range children {
		if child.Status != StateDead && child.Status != StateFailed && child.Status != StateCleanedUp {
			continue
		}
		if child.SandboxID != "" {
			if e := c.Conway.DeleteSandbox(ctx, child.SandboxID); e != nil {
				err = e
				continue
			}
		}
		if e := c.Store.DeleteChild(child.ID); e != nil {
			err = e
			continue
		}
		pruned++
	}
	return pruned, err
}
