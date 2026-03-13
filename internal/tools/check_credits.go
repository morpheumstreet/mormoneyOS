package tools

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
)

// CheckCreditsTool checks Conway compute credit balance.
// Requires Conway client; register only when Conway is configured.
type CheckCreditsTool struct {
	Conway conway.Client
}

func (t *CheckCreditsTool) Name() string        { return "check_credits" }
func (t *CheckCreditsTool) Description() string { return "Check your current Conway compute credit balance." }
func (t *CheckCreditsTool) Parameters() string  { return `{"type":"object","properties":{},"required":[]}` }

func (t *CheckCreditsTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	balance, err := t.Conway.GetCreditsBalance(ctx)
	if err != nil {
		return "", fmt.Errorf("get credits balance: %w", err)
	}
	return fmt.Sprintf("Credit balance: $%.2f (%d cents)", float64(balance)/100, balance), nil
}
