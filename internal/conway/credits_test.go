package conway

import (
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestTierFromCreditsCents_High(t *testing.T) {
	got := TierFromCreditsCents(501)
	if got != types.SurvivalTierHigh {
		t.Errorf("TierFromCreditsCents(501) = %q, want high", got)
	}
}

func TestTierFromCreditsCents_Normal(t *testing.T) {
	got := TierFromCreditsCents(51)
	if got != types.SurvivalTierNormal {
		t.Errorf("TierFromCreditsCents(51) = %q, want normal", got)
	}
}

func TestTierFromCreditsCents_LowCompute(t *testing.T) {
	got := TierFromCreditsCents(11)
	if got != types.SurvivalTierLowCompute {
		t.Errorf("TierFromCreditsCents(11) = %q, want low_compute", got)
	}
}

func TestTierFromCreditsCents_Critical(t *testing.T) {
	got := TierFromCreditsCents(0)
	if got != types.SurvivalTierCritical {
		t.Errorf("TierFromCreditsCents(0) = %q, want critical", got)
	}
}

func TestTierFromCreditsCents_Dead(t *testing.T) {
	got := TierFromCreditsCents(-1)
	if got != types.SurvivalTierDead {
		t.Errorf("TierFromCreditsCents(-1) = %q, want dead", got)
	}
}

func TestTierFromCreditsCents_BoundaryHighNormal(t *testing.T) {
	got := TierFromCreditsCents(500)
	if got != types.SurvivalTierNormal {
		t.Errorf("TierFromCreditsCents(500) = %q, want normal", got)
	}
}

func TestTierFromCreditsCents_BoundaryNormalLow(t *testing.T) {
	got := TierFromCreditsCents(50)
	if got != types.SurvivalTierLowCompute {
		t.Errorf("TierFromCreditsCents(50) = %q, want low_compute", got)
	}
}

func TestTierFromCreditsCents_BoundaryLowCritical(t *testing.T) {
	got := TierFromCreditsCents(10)
	if got != types.SurvivalTierCritical {
		t.Errorf("TierFromCreditsCents(10) = %q, want critical", got)
	}
}
