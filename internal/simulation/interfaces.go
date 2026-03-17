package simulation

import (
	"context"
	"time"
)

// Tick holds market/time data for one simulation step.
type Tick struct {
	Time   time.Time
	Price  float64   // optional; for price-aware strategies
	Volume float64   // optional
	News   string    // optional; headline or context
}

// MarketReplayProvider supplies ticks for replay (CSV, Conway, or mock).
type MarketReplayProvider interface {
	NextTick(ctx context.Context) (Tick, error)
	ResetToDay(start time.Time)
}

// Clock provides virtual time for deterministic simulation.
type Clock interface {
	Now() time.Time
	Advance(d time.Duration)
}
