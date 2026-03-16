package agent

import (
	"expvar"
	"sync/atomic"
)

var (
	agentInputTokensTotal atomic.Int64
	agentTruncationsTotal atomic.Int64
)

func init() {
	expvar.Publish("agent_input_tokens_total", expvar.Func(func() any { return agentInputTokensTotal.Load() }))
	expvar.Publish("agent_truncations_total", expvar.Func(func() any { return agentTruncationsTotal.Load() }))
}

// RecordInputTokens adds to the input tokens counter (call before each inference).
func RecordInputTokens(tokens int64) {
	agentInputTokensTotal.Add(tokens)
}

// RecordTruncation increments the truncation counter (call when BuildMessagesSafe truncates).
func RecordTruncation() {
	agentTruncationsTotal.Add(1)
}
