package heartbeat

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

const defaultLeaseTTLMs = 60_000

var tierOrder = map[string]int{
	"dead": 0, "critical": 1, "low_compute": 2, "normal": 3, "high": 4,
}

func tierMeetsMinimum(current types.SurvivalTier, minimum string) bool {
	c := tierOrder[string(current)]
	m := tierOrder[minimum]
	return c >= m
}

// Scheduler is a DB-backed cron scheduler (TS DurableScheduler-aligned).
type Scheduler struct {
	store     *state.Database
	tasks     map[string]Task
	ownerID   string
	onWake    func(reason string)
	log       *slog.Logger
	tickInProgress bool
}

// NewScheduler creates a DB-backed scheduler.
func NewScheduler(store *state.Database, taskList []Task, onWake func(reason string), log *slog.Logger) *Scheduler {
	if log == nil {
		log = slog.Default()
	}
	tasks := make(map[string]Task)
	for _, t := range taskList {
		tasks[t.Name] = t
	}
	return &Scheduler{
		store:   store,
		tasks:   tasks,
		ownerID: fmt.Sprintf("scheduler-%d", time.Now().UnixNano()%100000),
		onWake:  onWake,
		log:     log,
	}
}

// Tick runs one scheduler cycle: clear expired leases, get due tasks, execute each.
func (s *Scheduler) Tick(ctx context.Context, taskCtx *TaskContext) {
	if s.tickInProgress {
		return
	}
	s.tickInProgress = true
	defer func() { s.tickInProgress = false }()

	_, _ = s.store.ClearExpiredLeases()

	schedule, err := s.store.GetHeartbeatSchedule()
	if err != nil {
		s.log.Warn("get heartbeat schedule failed", "err", err)
		return
	}

	due := s.getDueTasks(schedule, taskCtx)
	for _, row := range due {
		s.executeTask(ctx, row, taskCtx)
	}
}

func (s *Scheduler) getDueTasks(schedule []state.HeartbeatScheduleRow, tc *TaskContext) []state.HeartbeatScheduleRow {
	now := time.Now()
	var out []state.HeartbeatScheduleRow
	for _, row := range schedule {
		if row.Enabled == 0 {
			continue
		}
		if tc != nil && tc.Tick != nil {
			if !tierMeetsMinimum(tc.Tick.SurvivalTier, row.TierMinimum) {
				continue
			}
		}
		if row.LeaseUntil != "" {
			exp, err := time.Parse(time.RFC3339, row.LeaseUntil)
			if err == nil && exp.After(now) && row.LeaseOwner != "" && row.LeaseOwner != s.ownerID {
				continue
			}
		}
		if row.NextRun != "" {
			next, err := time.Parse(time.RFC3339, row.NextRun)
			if err == nil && next.After(now) {
				continue
			}
		}
		due := s.isCronDue(row.Schedule, row.LastRun)
		if due {
			out = append(out, row)
		}
	}
	return out
}

func (s *Scheduler) isCronDue(cronExpr, lastRun string) bool {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(cronExpr)
	if err != nil {
		return false
	}
	var from time.Time
	if lastRun != "" {
		from, _ = time.Parse(time.RFC3339, lastRun)
	}
	if from.IsZero() {
		from = time.Now().Add(-24 * time.Hour)
	}
	next := sched.Next(from)
	return !next.After(time.Now())
}

func (s *Scheduler) executeTask(ctx context.Context, row state.HeartbeatScheduleRow, tc *TaskContext) {
	task, ok := s.tasks[row.Name]
	if !ok {
		task, ok = s.tasks[row.Task]
		if !ok {
			return
		}
	}
	acquired, err := s.store.AcquireTaskLease(row.Name, s.ownerID, defaultLeaseTTLMs)
	if err != nil || !acquired {
		return
	}
	defer func() { _ = s.store.ReleaseTaskLease(row.Name, s.ownerID) }()

	startedAt := time.Now().UTC().Format(time.RFC3339)
	historyID := uuid.New().String()

	wake, msg, err := task.Run(ctx, tc)
	finishedAt := time.Now().UTC().Format(time.RFC3339)
	success := 0
	result := "success"
	if err != nil {
		success = 0
		result = "failure"
		s.log.Warn("heartbeat task failed", "task", row.Name, "err", err)
	} else {
		success = 1
	}
	shouldWake := 0
	if wake {
		shouldWake = 1
	}
	_ = s.store.InsertHeartbeatHistory(historyID, row.Name, startedAt, finishedAt, success, result, shouldWake)
	_ = s.store.UpdateHeartbeatSchedule(row.Name, finishedAt, "", "")

	if wake && s.onWake != nil {
		reason := msg
		if reason == "" {
			reason = "heartbeat " + row.Name
		}
		s.onWake(reason)
		_ = s.store.InsertWakeEvent("heartbeat", reason)
	}
}
