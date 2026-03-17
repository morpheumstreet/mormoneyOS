package agent

import (
	"expvar"
	"sync/atomic"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
)

var (
	agentInputTokensTotal atomic.Int64
	agentTruncationsTotal atomic.Int64
	// Memory ingestion metrics (automatic memory pipeline)
	memoryIngestTurnsTotal    atomic.Int64
	memoryConsolidatedItems   atomic.Int64
	memoryPrunedCount         atomic.Int64
	memoryExtractionLatencyMs atomic.Int64
	// Model routing and critique
	routingStrongTotal atomic.Int64
	routingFastTotal   atomic.Int64
	routingStrongReasonTokens atomic.Int64
	routingStrongReasonMoney  atomic.Int64
	routingStrongReasonRisk   atomic.Int64
	critiqueTotal      atomic.Int64
	// critique_success_score_avg: rolling average (sum/count)
	critiqueSuccessScoreSum   atomic.Int64
	critiqueSuccessScoreCount  atomic.Int64
)

func init() {
	expvar.Publish("agent_input_tokens_total", expvar.Func(func() any { return agentInputTokensTotal.Load() }))
	expvar.Publish("agent_truncations_total", expvar.Func(func() any { return agentTruncationsTotal.Load() }))
	expvar.Publish("memory_ingest_turns_total", expvar.Func(func() any { return memoryIngestTurnsTotal.Load() }))
	expvar.Publish("memory_consolidated_items", expvar.Func(func() any { return memoryConsolidatedItems.Load() }))
	expvar.Publish("memory_pruned_count", expvar.Func(func() any { return memoryPrunedCount.Load() }))
	expvar.Publish("memory_extraction_latency_ms", expvar.Func(func() any { return memoryExtractionLatencyMs.Load() }))
	expvar.Publish("routing_strong_total", expvar.Func(func() any { return routingStrongTotal.Load() }))
	expvar.Publish("routing_fast_total", expvar.Func(func() any { return routingFastTotal.Load() }))
	expvar.Publish("routing_strong_reason_tokens", expvar.Func(func() any { return routingStrongReasonTokens.Load() }))
	expvar.Publish("routing_strong_reason_money", expvar.Func(func() any { return routingStrongReasonMoney.Load() }))
	expvar.Publish("routing_strong_reason_risk", expvar.Func(func() any { return routingStrongReasonRisk.Load() }))
	expvar.Publish("critique_total", expvar.Func(func() any { return critiqueTotal.Load() }))
	expvar.Publish("critique_success_score_avg", expvar.Func(func() any {
		if count := critiqueSuccessScoreCount.Load(); count > 0 {
			return float64(critiqueSuccessScoreSum.Load()) / 100.0 / float64(count) // sum stored as int (score*100)
		}
		return float64(0)
	}))
}

// RecordRoutingStrong increments the strong-tier routing counter.
func RecordRoutingStrong() {
	routingStrongTotal.Add(1)
}

// RecordRoutingFast increments the fast-tier routing counter.
func RecordRoutingFast() {
	routingFastTotal.Add(1)
}

// RecordCritique increments the critique counter.
func RecordCritique() {
	critiqueTotal.Add(1)
}

// RoutingMetrics implements inference.RoutingMetricsRecorder for the model router.
var RoutingMetrics inference.RoutingMetricsRecorder = &routingMetricsImpl{}

type routingMetricsImpl struct{}

func (*routingMetricsImpl) RecordTier(tier inference.ModelTier) {
	switch tier {
	case inference.TierStrong:
		routingStrongTotal.Add(1)
	case inference.TierFast:
		routingFastTotal.Add(1)
	}
}

func (*routingMetricsImpl) RecordTierWithReason(tier inference.ModelTier, reason inference.RoutingEscalationReason) {
	switch tier {
	case inference.TierStrong:
		switch reason {
		case inference.EscalationReasonTokens:
			routingStrongReasonTokens.Add(1)
		case inference.EscalationReasonMoney:
			routingStrongReasonMoney.Add(1)
		case inference.EscalationReasonRisk:
			routingStrongReasonRisk.Add(1)
		}
	}
}

// RecordCritiqueSuccessScore records a success_score for rolling average (0–1).
func RecordCritiqueSuccessScore(score float64) {
	if score < 0 || score > 1 {
		return
	}
	critiqueSuccessScoreSum.Add(int64(score * 100))
	critiqueSuccessScoreCount.Add(1)
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
