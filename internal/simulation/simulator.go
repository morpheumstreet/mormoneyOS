package simulation

import (
	"context"
	"log/slog"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/agent"
	"github.com/morpheumlabs/mormoneyos-go/internal/memory"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// Simulator orchestrates a simulation run using the real agent loop.
type Simulator struct {
	cfg       SimulationConfig
	db        *state.Database
	loop      *agent.Loop
	memSvc    *memory.MemoryService
	replay    MarketReplayProvider
	chaos     *ChaosInjector
	clock     Clock
	startDate time.Time
	log       *slog.Logger
}

// SimulatorOptions configures the simulator.
type SimulatorOptions struct {
	Config   SimulationConfig
	DB       *state.Database
	Loop     *agent.Loop
	MemSvc   *memory.MemoryService
	Replay   MarketReplayProvider
	Chaos    *ChaosInjector
	Clock    Clock
	Log      *slog.Logger
}

// NewSimulator creates a simulator from options.
func NewSimulator(opts SimulatorOptions) *Simulator {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.Clock == nil {
		opts.Clock = &simpleClock{now: opts.Config.StartDate}
	}
	if opts.Chaos == nil {
		opts.Chaos = NewChaosInjector(opts.Config.ChaosLevel, opts.Config.Seed)
	}
	return &Simulator{
		cfg:       opts.Config,
		db:        opts.DB,
		loop:      opts.Loop,
		memSvc:    opts.MemSvc,
		replay:    opts.Replay,
		chaos:     opts.Chaos,
		clock:     opts.Clock,
		startDate: opts.Config.StartDate,
		log:       opts.Log,
	}
}

// simpleClock is a minimal in-package clock for default use.
type simpleClock struct {
	now time.Time
}

func (m *simpleClock) Now() time.Time          { return m.now }
func (m *simpleClock) Advance(d time.Duration) { m.now = m.now.Add(d) }

// Run executes the simulation for the configured number of days.
func (s *Simulator) Run(ctx context.Context) (*RunResult, error) {
	ResetMetrics()
	result := &RunResult{
		StartTime: s.startDate,
		Turns:     make([]TurnRecord, 0),
	}

	for day := 0; day < s.cfg.Days && ctx.Err() == nil; day++ {
		dayStart := s.startDate.AddDate(0, 0, day)
		s.replay.ResetToDay(dayStart)

		for i := 0; i < 24; i++ { // placeholder: 24 ticks per day
			tick, err := s.replay.NextTick(ctx)
			if err != nil {
				return result, err
			}
			s.clock.Advance(time.Hour)

			res := s.loop.RunOneTurn(ctx, types.AgentStateRunning)
			if res.Err != nil {
				RecordCrash()
				s.log.Warn("sim turn crashed", "day", day, "tick", i, "err", res.Err)
				continue
			}

			RecordTurn(0) // token usage from turn would be passed when available
			result.Turns = append(result.Turns, TurnRecord{
				Day:   day,
				Tick:  i,
				Time:  tick.Time,
				State: string(res.State),
			})

			if s.memSvc != nil {
				// IngestTurn is called inside the loop; we could also record here for metrics
				RecordIngestionCandidate()
			}
		}

		if s.memSvc != nil {
			// Consolidator tick would run here in full implementation
		}
	}

	result.EndTime = s.clock.Now()
	result.TotalTurns = len(result.Turns)
	return result, nil
}

// RunResult holds the outcome of a simulation run.
type RunResult struct {
	StartTime  time.Time
	EndTime    time.Time
	TotalTurns int
	Turns      []TurnRecord
}

// TurnRecord is a single turn in the trace.
type TurnRecord struct {
	Day   int
	Tick  int
	Time  time.Time
	State string
}
