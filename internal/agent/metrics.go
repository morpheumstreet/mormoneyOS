package agent

import (
	"expvar"
	"sync/atomic"
)

var (
	agentInputTokensTotal atomic.Int64
	agentTruncationsTotal atomic.Int64
	// Memory ingestion metrics (automatic memory pipeline)
	memoryIngestTurnsTotal    atomic.Int64
	memoryConsolidatedItems   atomic.Int64
	memoryPrunedCount         atomic.Int64
	memoryExtractionLatencyMs atomic.Int64
)

func init() {
	expvar.Publish("agent_input_tokens_total", expvar.Func(func() any { return agentInputTokensTotal.Load() }))
	expvar.Publish("agent_truncations_total", expvar.Func(func() any { return agentTruncationsTotal.Load() }))
	expvar.Publish("memory_ingest_turns_total", expvar.Func(func() any { return memoryIngestTurnsTotal.Load() }))
	expvar.Publish("memory_consolidated_items", expvar.Func(func() any { return memoryConsolidatedItems.Load() }))
	expvar.Publish("memory_pruned_count", expvar.Func(func() any { return memoryPrunedCount.Load() }))
	expvar.Publish("memory_extraction_latency_ms", expvar.Func(func() any { return memoryExtractionLatencyMs.Load() }))
}

// RecordInputTokens adds to the input tokens counter (call before each inference).
func RecordInputTokens(tokens int64) {
	agentInputTokensTotal.Add(tokens)
}

// RecordTruncation increments the truncation counter (call when BuildMessagesSafe truncates).
func RecordTruncation() {
	agentTruncationsTotal.Add(1)
}

// RecordMemoryIngestTurn increments the ingest turns counter.
func RecordMemoryIngestTurn() {
	memoryIngestTurnsTotal.Add(1)
}

// RecordMemoryConsolidated adds to the consolidated items counter.
func RecordMemoryConsolidated(n int64) {
	memoryConsolidatedItems.Add(n)
}

// RecordMemoryPruned adds to the pruned count.
func RecordMemoryPruned(n int64) {
	memoryPrunedCount.Add(n)
}

// RecordMemoryExtractionLatency records extraction latency in ms.
func RecordMemoryExtractionLatency(ms int64) {
	memoryExtractionLatencyMs.Add(ms)
}
