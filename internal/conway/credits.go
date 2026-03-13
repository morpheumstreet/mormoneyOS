package conway

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// TierFromCreditsCents returns survival tier from credit balance.
func TierFromCreditsCents(cents int64) types.SurvivalTier {
	switch {
	case cents > 500:
		return types.SurvivalTierHigh
	case cents > 50:
		return types.SurvivalTierNormal
	case cents > 10:
		return types.SurvivalTierLowCompute
	case cents >= 0:
		return types.SurvivalTierCritical
	default:
		return types.SurvivalTierDead
	}
}
