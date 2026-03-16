package simulation

import (
	"context"
	"time"
)

// ConstantTickReplay yields a fixed number of ticks at a fixed interval.
// Used when no CSV/market data is configured.
type ConstantTickReplay struct {
	Start     time.Time
	Interval  time.Duration
	TicksPerDay int
	Price     float64
}

// NewConstantTickReplay creates a replay with N ticks per simulated day.
func NewConstantTickReplay(start time.Time, interval time.Duration, ticksPerDay int, price float64) *ConstantTickReplay {
	if ticksPerDay <= 0 {
		ticksPerDay = 24 // one per hour
	}
	if interval <= 0 {
		interval = time.Hour
	}
	return &ConstantTickReplay{
		Start:       start.Truncate(24 * time.Hour),
		Interval:    interval,
		TicksPerDay: ticksPerDay,
		Price:       price,
	}
}

// NextTick returns the next tick.
func (r *ConstantTickReplay) NextTick(ctx context.Context) (Tick, error) {
	select {
	case <-ctx.Done():
		return Tick{}, ctx.Err()
	default:
	}
	tick := Tick{
		Time:   r.Start,
		Price:  r.Price,
		Volume: 1000,
	}
	r.Start = r.Start.Add(r.Interval)
	return tick, nil
}

// ResetToDay resets to the start of the given day.
func (r *ConstantTickReplay) ResetToDay(start time.Time) {
	r.Start = start.Truncate(24 * time.Hour)
}
