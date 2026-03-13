package tools

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
)

// NewConwayTools returns Conway-specific tools when Conway client is configured.
// Register these via Registry.RegisterMany() when Conway API is available.
func NewConwayTools(c conway.Client) []Tool {
	if c == nil {
		return nil
	}
	return []Tool{
		&CheckCreditsTool{Conway: c},
		&ListSandboxesTool{Conway: c},
		&TransferCreditsTool{Conway: c},
		&CreateSandboxTool{Conway: c},
		&DeleteSandboxTool{Conway: c},
	}
}
