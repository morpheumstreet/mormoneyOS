package heartbeat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/replication"
	"github.com/morpheumlabs/mormoneyos-go/internal/social"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// WakeInserter inserts wake events into the database (e.g. when heartbeat tasks request wake).
type WakeInserter interface {
	InsertWakeEvent(source, reason string) error
}

// Daemon runs background tasks per mormoneyOS design.
type Daemon struct {
	tickInterval   time.Duration
	wakeCheck      time.Duration
	tasks          []Task
	wakeInserter   WakeInserter
	store         TaskStore
	creditsFn     func(context.Context) int64
	config        *types.AutomatonConfig
	conway        conway.Client
	channels      map[string]social.SocialChannel
	address       string
	healthMonitor *replication.ChildHealthMonitor
	sandboxCleanup *replication.SandboxCleanup
	scheduler      *Scheduler // when store is *state.Database
	log            *slog.Logger
	stop           chan struct{}
	wg             sync.WaitGroup
}

// Task is a heartbeat task that may emit wake events (TS HeartbeatTaskFn-aligned).
type Task struct {
	Name     string
	Schedule string // cron expression
	Run      func(ctx context.Context, tc *TaskContext) (shouldWake bool, message string, err error)
}

// NewDaemon creates a heartbeat daemon (minimal; no DB/Conway).
func NewDaemon(tickInterval, wakeCheck time.Duration, tasks []Task, log *slog.Logger) *Daemon {
	return NewDaemonWithWakeInserter(tickInterval, wakeCheck, tasks, nil, log)
}

// NewDaemonWithWakeInserter creates a daemon that inserts wake events when tasks request wake.
func NewDaemonWithWakeInserter(tickInterval, wakeCheck time.Duration, tasks []Task, wake WakeInserter, log *slog.Logger) *Daemon {
	if log == nil {
		log = slog.Default()
	}
	return &Daemon{
		tickInterval: tickInterval,
		wakeCheck:    wakeCheck,
		tasks:        tasks,
		wakeInserter: wake,
		log:          log,
		stop:         make(chan struct{}),
	}
}

// DaemonOptions configures the full heartbeat daemon (TS-aligned).
type DaemonOptions struct {
	TickInterval   time.Duration
	WakeCheck      time.Duration
	Tasks          []Task
	Store          TaskStore
	WakeInserter   WakeInserter
	CreditsFn      func(context.Context) int64
	Config         *types.AutomatonConfig
	Conway         conway.Client
	Channels       map[string]social.SocialChannel
	Address        string
	HealthMonitor  *replication.ChildHealthMonitor // optional; for check_child_health
	SandboxCleanup *replication.SandboxCleanup     // optional; for prune_dead_children
	Log            *slog.Logger
}

// NewDaemonWithOptions creates a daemon with full task context (TS-aligned).
// When Store is *state.Database, uses DB-backed cron scheduler.
func NewDaemonWithOptions(opts DaemonOptions) *Daemon {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	d := &Daemon{
		tickInterval:   opts.TickInterval,
		wakeCheck:      opts.WakeCheck,
		tasks:          opts.Tasks,
		wakeInserter:   opts.WakeInserter,
		store:          opts.Store,
		creditsFn:      opts.CreditsFn,
		config:         opts.Config,
		conway:         opts.Conway,
		channels:       opts.Channels,
		address:        opts.Address,
		healthMonitor:  opts.HealthMonitor,
		sandboxCleanup: opts.SandboxCleanup,
		log:            opts.Log,
		stop:           make(chan struct{}),
	}
	if db, ok := opts.Store.(*state.Database); ok {
		seedHeartbeatSchedule(db, opts.Tasks)
		d.scheduler = NewScheduler(db, opts.Tasks, func(reason string) {
			if d.wakeInserter != nil {
				_ = d.wakeInserter.InsertWakeEvent("heartbeat", reason)
			}
		}, opts.Log)
	}
	return d
}

func seedHeartbeatSchedule(db *state.Database, tasks []Task) {
	for _, t := range tasks {
		_ = db.UpsertHeartbeatSchedule(state.HeartbeatScheduleRow{
			Name:        t.Name,
			Schedule:    t.Schedule,
			Task:        t.Name,
			Enabled:     1,
			TierMinimum: "dead",
		})
	}
}

// Start begins the tick loop.
func (d *Daemon) Start(ctx context.Context) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		ticker := time.NewTicker(d.tickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-d.stop:
				return
			case <-ticker.C:
				d.tick(ctx)
			}
		}
	}()
	d.log.Info("heartbeat daemon started", "tick_interval", d.tickInterval)
}

// Stop stops the daemon.
func (d *Daemon) Stop() {
	close(d.stop)
	d.wg.Wait()
}

func (d *Daemon) tick(ctx context.Context) {
	var taskCtx *TaskContext
	if d.store != nil {
		creditsFn := func() int64 { return 0 }
		if d.creditsFn != nil {
			creditsFn = func() int64 { return d.creditsFn(ctx) }
		}
		tick := BuildTickContext(creditsFn)
		taskCtx = &TaskContext{
			Tick:           tick,
			DB:             d.store,
			Conway:         d.conway,
			Channels:       d.channels,
			Config:         d.config,
			Address:        d.address,
			HealthMonitor:  d.healthMonitor,
			SandboxCleanup: d.sandboxCleanup,
		}
	}

	if d.scheduler != nil {
		d.scheduler.Tick(ctx, taskCtx)
		return
	}

	for _, t := range d.tasks {
		select {
		case <-ctx.Done():
			return
		default:
			wake, msg, err := t.Run(ctx, taskCtx)
			if err != nil {
				d.log.Warn("heartbeat task failed", "task", t.Name, "err", err)
			} else if wake && d.wakeInserter != nil {
				reason := msg
				if reason == "" {
					reason = t.Name
				}
				if err := d.wakeInserter.InsertWakeEvent("heartbeat", reason); err != nil {
					d.log.Warn("insert wake event failed", "task", t.Name, "err", err)
				} else {
					d.log.Info("heartbeat task requested wake", "task", t.Name, "reason", reason)
				}
			}
		}
	}
}
