package replication

// ChildLifecycleState represents child automaton lifecycle states (TS-aligned).
// Flow: requested → sandbox_created → runtime_ready → wallet_verified → funded →
// starting → healthy → (unhealthy|stopped|failed) → cleaned_up
const (
	StateRequested       = "requested"
	StateSandboxCreated  = "sandbox_created"
	StateRuntimeReady    = "runtime_ready"
	StateWalletVerified  = "wallet_verified"
	StateFunded          = "funded"
	StateStarting        = "starting"
	StateHealthy         = "healthy"
	StateUnhealthy       = "unhealthy"
	StateStopped         = "stopped"
	StateFailed          = "failed"
	StateDead            = "dead"
	StateCleanedUp       = "cleaned_up"
)

// ValidTransitions maps each state to allowed next states (TS-aligned).
var ValidTransitions = map[string][]string{
	StateRequested:      {StateSandboxCreated, StateFailed},
	StateSandboxCreated:  {StateRuntimeReady, StateFailed},
	StateRuntimeReady:   {StateWalletVerified, StateFailed},
	StateWalletVerified:  {StateFunded, StateStarting, StateFailed},
	StateFunded:         {StateStarting, StateFailed},
	StateStarting:       {StateHealthy, StateUnhealthy, StateFailed},
	StateHealthy:        {StateUnhealthy, StateStopped, StateDead},
	StateUnhealthy:      {StateHealthy, StateStopped, StateDead},
	StateStopped:        {StateStarting, StateDead},
	StateFailed:         {StateCleanedUp},
	StateDead:           {StateCleanedUp},
	StateCleanedUp:      {},
}
