package heartbeat

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// TickContext holds shared data for a single tick (TS buildTickContext-aligned).
// Credits are fetched once per tick and shared across all tasks.
type TickContext struct {
	CreditBalance int64
	SurvivalTier  types.SurvivalTier
	USDCBalance   float64 // optional; 0 when not available
}

// TaskContext holds tick context plus dependencies for task execution (TS HeartbeatLegacyContext-aligned).
type TaskContext struct {
	Tick    *TickContext
	DB      TaskStore
	Conway  conway.Client // nil when Conway not configured
	Config  *types.AutomatonConfig
	Address string // wallet address from identity
}

// TaskStore is the minimal DB interface for heartbeat tasks.
type TaskStore interface {
	GetKV(key string) (string, bool, error)
	SetKV(key, value string) error
	DeleteKV(key string) error
	GetAgentState() (string, bool, error)
	SetAgentState(state string) error
	InsertWakeEvent(source, reason string) error
}

// ChildStore is an optional extension for tasks that need children operations.
// *state.Database implements this when children table exists.
type ChildStore interface {
	GetAllChildren() ([]state.Child, bool)
	UpdateChildStatus(id, status string) error
}

// Ensure *state.Database implements TaskStore and ChildStore.
var _ TaskStore = (*state.Database)(nil)
var _ ChildStore = (*state.Database)(nil)

// BuildTickContext fetches credits once and builds TickContext (TS buildTickContext-aligned).
func BuildTickContext(creditsFn func() int64) *TickContext {
	credits := int64(0)
	if creditsFn != nil {
		credits = creditsFn()
	}
	tier := conway.TierFromCreditsCents(credits)
	return &TickContext{
		CreditBalance: credits,
		SurvivalTier:  tier,
		USDCBalance:   0, // Go: no USDC balance API yet
	}
}
