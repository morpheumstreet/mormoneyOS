package simulation

import (
	"expvar"
	"sync/atomic"
)

var (
	simTurnsTotal       atomic.Int64
	simCrashesTotal     atomic.Int64
	simTokenUsagePeak   atomic.Int64
	simMemoryGrowth     atomic.Int64
	simTrimEventsTotal  atomic.Int64
	simIngestionCandidates atomic.Int64
)

func init() {
	expvar.Publish("sim_turns_total", expvar.Func(func() any { return simTurnsTotal.Load() }))
	expvar.Publish("sim_crashes_total", expvar.Func(func() any { return simCrashesTotal.Load() }))
	expvar.Publish("sim_token_usage_peak", expvar.Func(func() any { return simTokenUsagePeak.Load() }))
	expvar.Publish("sim_memory_growth", expvar.Func(func() any { return simMemoryGrowth.Load() }))
	expvar.Publish("sim_trim_events_total", expvar.Func(func() any { return simTrimEventsTotal.Load() }))
	expvar.Publish("sim_ingestion_candidates_processed", expvar.Func(func() any { return simIngestionCandidates.Load() }))
}

// RecordTurn increments turn count and optionally updates peak token usage.
func RecordTurn(tokenUsage int) {
	simTurnsTotal.Add(1)
	for {
		peak := simTokenUsagePeak.Load()
		if int64(tokenUsage) <= peak || simTokenUsagePeak.CompareAndSwap(peak, int64(tokenUsage)) {
			break
		}
	}
}

// RecordCrash increments the crash counter.
func RecordCrash() {
	simCrashesTotal.Add(1)
}

// RecordMemoryGrowth adds to memory growth metric.
func RecordMemoryGrowth(delta int64) {
	simMemoryGrowth.Add(delta)
}

// RecordTrimEvent increments trim events.
func RecordTrimEvent() {
	simTrimEventsTotal.Add(1)
}

// RecordIngestionCandidate increments ingestion candidates processed.
func RecordIngestionCandidate() {
	simIngestionCandidates.Add(1)
}

// ResetMetrics zeros all sim metrics (for test isolation).
func ResetMetrics() {
	simTurnsTotal.Store(0)
	simCrashesTotal.Store(0)
	simTokenUsagePeak.Store(0)
	simMemoryGrowth.Store(0)
	simTrimEventsTotal.Store(0)
	simIngestionCandidates.Store(0)
}
