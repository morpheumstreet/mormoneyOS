package replication

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// ConwayExec executes commands in Conway sandboxes.
type ConwayExec interface {
	ExecInSandbox(ctx context.Context, sandboxID, command string, timeoutMs int) (conway.ExecResult, error)
}

// ChildHealthStore provides children and status updates.
type ChildHealthStore interface {
	GetAllChildren() ([]state.Child, bool)
	UpdateChildStatus(id, status string) error
}

// ChildHealthMonitor checks child health via Conway exec (TS ChildHealthMonitor-aligned).
// Runs health check command in each child sandbox; parses JSON response; updates status.
type ChildHealthMonitor struct {
	Conway ConwayExec
	Store  ChildHealthStore
	// MaxConcurrent limits parallel health checks (default 3).
	MaxConcurrent int
}

// healthCheckResult is the expected JSON shape from child health endpoint/command.
type healthCheckResult struct {
	Status  string `json:"status"`  // "healthy", "unhealthy", etc.
	Credits int64  `json:"credits"`  // optional
	State   string `json:"state"`   // optional
}

// Check runs health checks for all non-dead children with sandbox_id.
func (m *ChildHealthMonitor) Check(ctx context.Context) (checked int, needsAttention []string) {
	if m.Conway == nil || m.Store == nil {
		return 0, nil
	}
	children, ok := m.Store.GetAllChildren()
	if !ok || len(children) == 0 {
		return 0, nil
	}
	maxConcurrent := m.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, c := range children {
		if c.Status == StateDead || c.Status == StateCleanedUp || c.Status == StateFailed {
			continue
		}
		if c.SandboxID == "" {
			mu.Lock()
			needsAttention = append(needsAttention, c.Name+" (no sandbox)")
			mu.Unlock()
			continue
		}
		wg.Add(1)
		go func(child state.Child) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			res, err := m.Conway.ExecInSandbox(ctx, child.SandboxID, "automaton --status 2>/dev/null || echo '{}'", 15_000)
			mu.Lock()
			checked++
			mu.Unlock()
			if err != nil {
				_ = m.Store.UpdateChildStatus(child.ID, StateUnhealthy)
				mu.Lock()
				needsAttention = append(needsAttention, child.Name+" (exec failed)")
				mu.Unlock()
				return
			}
			var h healthCheckResult
			if json.Unmarshal([]byte(res.Stdout), &h) != nil {
				// Fallback: non-JSON output; treat as unhealthy if exit code non-zero
				if res.ExitCode != 0 {
					_ = m.Store.UpdateChildStatus(child.ID, StateUnhealthy)
					mu.Lock()
					needsAttention = append(needsAttention, child.Name+" (bad response)")
					mu.Unlock()
				}
				return
			}
			switch h.Status {
			case "healthy", "running":
				_ = m.Store.UpdateChildStatus(child.ID, StateHealthy)
			case "unhealthy", "critical", "dead":
				_ = m.Store.UpdateChildStatus(child.ID, StateUnhealthy)
				mu.Lock()
				needsAttention = append(needsAttention, child.Name)
				mu.Unlock()
			default:
				_ = m.Store.UpdateChildStatus(child.ID, StateUnhealthy)
			}
		}(c)
	}
	wg.Wait()
	return checked, needsAttention
}

// StaleThreshold is the duration after which a child is considered stale (TS-aligned: 7 days).
const StaleThreshold = 7 * 24 * time.Hour
