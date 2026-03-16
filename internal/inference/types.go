package inference

// ModelTier selects model capability for routing (cheapest→strongest).
type ModelTier int

const (
	TierFast   ModelTier = iota // cheapest & fastest (default for normal turns)
	TierNormal                  // balanced
	TierStrong                  // highest reasoning power
)

// String returns the tier name for config/logging.
func (t ModelTier) String() string {
	switch t {
	case TierFast:
		return "fast"
	case TierNormal:
		return "normal"
	case TierStrong:
		return "strong"
	default:
		return "normal"
	}
}

// RoutingRiskLevel is used by the router (policy or heuristic).
type RoutingRiskLevel int

const (
	RiskLow RoutingRiskLevel = iota
	RiskMedium
	RiskHigh
)

// DecisionContext is passed to the model router for tier selection.
type DecisionContext struct {
	TokensUsed     int           // estimated input tokens for this turn
	RiskLevel      RoutingRiskLevel
	HasMoneyImpact bool          // transfer_credits, fund_child, etc.
	TurnPhase      string        // "planning", "action", "reflection"
	Uncertainty    float64       // optional, from prompt analysis (0–1)
}
