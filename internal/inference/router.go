package inference

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// RoutingConfig holds routing parameters (from types.AutomatonConfig.Routing).
type RoutingConfig struct {
	DefaultTier            string // "fast", "normal", "strong"
	StrongThresholdTokens  int    // use strong when tokens > this
	ForceStrongOnMoneyMove bool
	ReflectionTier         string
}

// DefaultRoutingConfig returns sensible defaults.
func DefaultRoutingConfig() RoutingConfig {
	return RoutingConfig{
		DefaultTier:            "normal",
		StrongThresholdTokens:  3500,
		ForceStrongOnMoneyMove: true,
		ReflectionTier:         "fast",
	}
}

// RoutingMetricsRecorder records routing decisions (optional, avoids import cycles).
type RoutingMetricsRecorder interface {
	RecordTier(tier ModelTier)
}

// ModelRouter selects the best inference client for a turn based on DecisionContext.
type ModelRouter struct {
	cfg     *types.AutomatonConfig
	holder  *InferenceClientHolder
	routing RoutingConfig
	metrics RoutingMetricsRecorder
	// cached clients per tier (lazy, from cfg.Models)
	mu   sync.RWMutex
	fast Client
	strong Client
	log  *slog.Logger
}

// NewModelRouter creates a router. When cfg.Routing is nil, uses DefaultRoutingConfig.
func NewModelRouter(cfg *types.AutomatonConfig, holder *InferenceClientHolder, metrics RoutingMetricsRecorder, log *slog.Logger) *ModelRouter {
	routing := DefaultRoutingConfig()
	if cfg != nil && cfg.Routing != nil {
		if cfg.Routing.DefaultTier != "" {
			routing.DefaultTier = cfg.Routing.DefaultTier
		}
		if cfg.Routing.StrongThresholdTokens > 0 {
			routing.StrongThresholdTokens = cfg.Routing.StrongThresholdTokens
		}
		routing.ForceStrongOnMoneyMove = cfg.Routing.ForceStrongOnMoneyMove
		if cfg.Routing.ReflectionTier != "" {
			routing.ReflectionTier = cfg.Routing.ReflectionTier
		}
	}
	if log == nil {
		log = slog.Default()
	}
	return &ModelRouter{cfg: cfg, holder: holder, routing: routing, metrics: metrics, log: log}
}

// Select returns the best client for this turn. Thread-safe.
func (r *ModelRouter) Select(ctx context.Context, dc DecisionContext) (Client, error) {
	tier := r.decideTier(dc)
	if r.metrics != nil {
		r.metrics.RecordTier(tier)
	}
	client := r.clientForTier(tier)
	if client == nil {
		client = r.holder.Client()
	}
	return client, nil
}

// ClientForReflection returns the client for critique/reflection calls (always fast tier).
func (r *ModelRouter) ClientForReflection(ctx context.Context) (Client, error) {
	tier := r.parseTier(r.routing.ReflectionTier)
	client := r.clientForTier(tier)
	if client == nil {
		client = r.holder.Client()
	}
	return client, nil
}

func (r *ModelRouter) decideTier(dc DecisionContext) ModelTier {
	// High token usage → strong model
	if r.routing.StrongThresholdTokens > 0 && dc.TokensUsed >= r.routing.StrongThresholdTokens {
		r.log.Debug("routing: strong (tokens)", "tokens", dc.TokensUsed, "threshold", r.routing.StrongThresholdTokens)
		return TierStrong
	}
	// Money impact → strong model
	if r.routing.ForceStrongOnMoneyMove && dc.HasMoneyImpact {
		r.log.Debug("routing: strong (money impact)")
		return TierStrong
	}
	// High risk → strong model
	if dc.RiskLevel == RiskHigh {
		r.log.Debug("routing: strong (risk)")
		return TierStrong
	}
	return r.parseTier(r.routing.DefaultTier)
}

func (r *ModelRouter) parseTier(s string) ModelTier {
	switch s {
	case "fast":
		return TierFast
	case "strong":
		return TierStrong
	default:
		return TierNormal
	}
}

func (r *ModelRouter) clientForTier(tier ModelTier) Client {
	if r.cfg == nil || len(r.cfg.Models) == 0 {
		return nil
	}
	r.mu.RLock()
	var c Client
	switch tier {
	case TierFast:
		c = r.fast
	case TierStrong:
		c = r.strong
	default:
		r.mu.RUnlock()
		return r.holder.Client()
	}
	r.mu.RUnlock()
	if c != nil {
		return c
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// Double-check after lock
	switch tier {
	case TierFast:
		if r.fast != nil {
			return r.fast
		}
		r.fast = r.clientForModelIndex(0)
		return r.fast
	case TierStrong:
		if r.strong != nil {
			return r.strong
		}
		r.strong = r.clientForModelIndex(len(r.cfg.Models) - 1)
		return r.strong
	}
	return r.holder.Client()
}

// clientForModelIndex returns client for the model at index (sorted by priority).
func (r *ModelRouter) clientForModelIndex(idx int) Client {
	models := make([]types.LLMModelEntry, len(r.cfg.Models))
	copy(models, r.cfg.Models)
	sort.Slice(models, func(i, j int) bool { return models[i].Priority < models[j].Priority })
	if idx < 0 || idx >= len(models) {
		return nil
	}
	entry := &models[idx]
	client := NewClientForModelEntry(r.cfg, entry)
	if client == nil {
		return nil
	}
	return client
}

// Reload clears cached clients so they are recreated on next Select (e.g. after config change).
func (r *ModelRouter) Reload() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fast = nil
	r.strong = nil
}
