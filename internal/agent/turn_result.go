package agent

import "github.com/morpheumlabs/mormoneyos-go/internal/types"

// TurnResult holds the outcome of one agent turn (TS step 13 aligned).
// State is Sleeping when the agent chose to sleep (sleep tool or finishReason stop).
// WasIdle is true when the turn had no mutating tools and turnCount > 0.
type TurnResult struct {
	State   types.AgentState
	WasIdle bool
	Err     error
}
