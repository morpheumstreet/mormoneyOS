package heartbeat

import (
	"context"
	"testing"
	"time"
)

func TestDefaultTasks(t *testing.T) {
	tasks := DefaultTasks()
	if len(tasks) < 3 {
		t.Errorf("DefaultTasks() len = %d, want >= 3", len(tasks))
	}
	names := make(map[string]bool)
	for _, t := range tasks {
		names[t.Name] = true
	}
	for _, want := range []string{"heartbeat_ping", "check_credits", "check_usdc_balance"} {
		if !names[want] {
			t.Errorf("DefaultTasks() missing %q", want)
		}
	}
}

func TestDaemon_StartStop(t *testing.T) {
	daemon := NewDaemon(50*time.Millisecond, 30*time.Second, DefaultTasks(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	daemon.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	cancel()
	daemon.Stop()
}

func TestDaemon_ContextCancelStops(t *testing.T) {
	daemon := NewDaemon(100*time.Millisecond, 30*time.Second, DefaultTasks(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	daemon.Start(ctx)
	cancel()
	daemon.Stop()
}
