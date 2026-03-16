package mock

import (
	"context"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/simulation"
)

// ReplayProvider is a simple mock that yields one tick per Advance.
type ReplayProvider struct {
	start    time.Time
	current  time.Time
	interval time.Duration
	price    float64
}

// NewReplayProvider creates a mock replay that advances by interval each tick.
func NewReplayProvider(start time.Time, interval time.Duration, initialPrice float64) *ReplayProvider {
	if initialPrice <= 0 {
		initialPrice = 50000.0 // placeholder BTC price
	}
	return &ReplayProvider{
		start:    start,
		current:  start,
		interval: interval,
		price:    initialPrice,
	}
}

// NextTick returns the next tick and advances internal state.
func (r *ReplayProvider) NextTick(ctx context.Context) (simulation.Tick, error) {
	select {
	case <-ctx.Done():
		return simulation.Tick{}, ctx.Err()
	default:
	}
	tick := simulation.Tick{
		Time:   r.current,
		Price:  r.price,
		Volume: 1000,
	}
	r.current = r.current.Add(r.interval)
	return tick, nil
}

// ResetToDay resets to the start of the given day.
func (r *ReplayProvider) ResetToDay(start time.Time) {
	r.current = start.Truncate(24 * time.Hour)
}
