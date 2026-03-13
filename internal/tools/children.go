package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// ChildStore provides child automaton DB operations.
type ChildStore interface {
	GetAllChildren() ([]state.Child, bool)
	UpdateChildStatus(id, status string) error
}

const childStaleThreshold = 7 * 24 * time.Hour

// ListChildrenTool lists child automatons.
type ListChildrenTool struct {
	Store interface {
		GetAllChildren() ([]state.Child, bool)
	}
}

func (ListChildrenTool) Name() string        { return "list_children" }
func (ListChildrenTool) Description() string { return "List child automatons with status." }
func (ListChildrenTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *ListChildrenTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "list_children requires store"}
	}
	children, ok := t.Store.GetAllChildren()
	if !ok || len(children) == 0 {
		return "No children.", nil
	}
	var sb strings.Builder
	for _, c := range children {
		sb.WriteString(fmt.Sprintf("- %s (id=%s): %s, funded=$%.2f, last_checked=%s\n",
			c.Name, c.ID, c.Status, float64(c.FundedAmountCents)/100, c.LastChecked))
	}
	return strings.TrimSuffix(sb.String(), "\n"), nil
}

// CheckChildStatusTool checks and optionally updates a child's status.
type CheckChildStatusTool struct {
	Store interface {
		GetAllChildren() ([]state.Child, bool)
		UpdateChildStatus(id, status string) error
	}
}

func (CheckChildStatusTool) Name() string        { return "check_child_status" }
func (CheckChildStatusTool) Description() string { return "Check a child automaton's status by id." }
func (CheckChildStatusTool) Parameters() string {
	return `{"type":"object","properties":{"child_id":{"type":"string"}},"required":["child_id"]}`
}

func (t *CheckChildStatusTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "check_child_status requires store"}
	}
	id, _ := args["child_id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return "", ErrInvalidArgs{Msg: "child_id required"}
	}
	children, ok := t.Store.GetAllChildren()
	if !ok {
		return "Children table not available.", nil
	}
	for _, c := range children {
		if c.ID == id {
			return fmt.Sprintf("Child %s: status=%s, funded=$%.2f, last_checked=%s",
				c.Name, c.Status, float64(c.FundedAmountCents)/100, c.LastChecked), nil
		}
	}
	return fmt.Sprintf("Child %q not found", id), nil
}

// PruneDeadChildrenTool marks children as dead when stale (>7 days).
type PruneDeadChildrenTool struct {
	Store interface {
		GetAllChildren() ([]state.Child, bool)
		UpdateChildStatus(id, status string) error
	}
}

func (PruneDeadChildrenTool) Name() string        { return "prune_dead_children" }
func (PruneDeadChildrenTool) Description() string { return "Mark stale children (>7d unchecked) as dead." }
func (PruneDeadChildrenTool) Parameters() string {
	return `{"type":"object","properties":{},"required":[]}`
}

func (t *PruneDeadChildrenTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "prune_dead_children requires store"}
	}
	children, ok := t.Store.GetAllChildren()
	if !ok || len(children) == 0 {
		return "No children to prune.", nil
	}
	now := time.Now()
	pruned := 0
	for _, c := range children {
		if c.Status == "dead" {
			continue
		}
		if c.LastChecked == "" {
			continue
		}
		lastChecked, err := time.Parse(time.RFC3339, c.LastChecked)
		if err != nil {
			continue
		}
		if now.Sub(lastChecked) > childStaleThreshold {
			if err := t.Store.UpdateChildStatus(c.ID, "dead"); err == nil {
				pruned++
			}
		}
	}
	if pruned > 0 {
		return fmt.Sprintf("Pruned %d dead children", pruned), nil
	}
	return "No stale children to prune.", nil
}

// FundChildStore provides child lookup and funded amount update for fund_child.
type FundChildStore interface {
	GetChildByID(id string) (*state.Child, bool)
	AddChildFundedAmount(id string, amount int64) error
}

// FundChildTool transfers Conway credits to a child automaton.
type FundChildTool struct {
	Conway conway.Client
	Store  FundChildStore
}

func (FundChildTool) Name() string        { return "fund_child" }
func (FundChildTool) Description() string { return "Transfer credits to a child automaton. Child must have wallet_verified or later status." }
func (FundChildTool) Parameters() string {
	return `{"type":"object","properties":{"child_id":{"type":"string","description":"Child automaton ID"},"amount_cents":{"type":"number","description":"Amount in cents to transfer"}},"required":["child_id","amount_cents"]}`
}

var ethAddressRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func (t *FundChildTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "fund_child requires store"}
	}
	childID, _ := args["child_id"].(string)
	childID = strings.TrimSpace(childID)
	if childID == "" {
		return "", ErrInvalidArgs{Msg: "child_id required"}
	}
	amountCents := int64(0)
	switch v := args["amount_cents"].(type) {
	case float64:
		amountCents = int64(v)
	case int:
		amountCents = int64(v)
	case int64:
		amountCents = v
	default:
		return "", ErrInvalidArgs{Msg: "amount_cents must be a number"}
	}
	if amountCents <= 0 {
		return "", ErrInvalidArgs{Msg: "amount_cents must be positive"}
	}

	child, ok := t.Store.GetChildByID(childID)
	if !ok || child == nil {
		return fmt.Sprintf("Child %q not found.", childID), nil
	}
	if !ethAddressRegex.MatchString(child.Address) {
		return fmt.Sprintf("Blocked: Child %s has invalid wallet address. Must be wallet_verified.", childID), nil
	}
	validFundingStates := map[string]bool{
		"wallet_verified": true, "funded": true, "starting": true, "healthy": true, "unhealthy": true,
	}
	if !validFundingStates[child.Status] {
		return fmt.Sprintf("Blocked: Child status is %q, must be wallet_verified or later to fund.", child.Status), nil
	}

	balance, err := t.Conway.GetCreditsBalance(ctx)
	if err != nil {
		return "", fmt.Errorf("get credits balance: %w", err)
	}
	if amountCents > balance/2 {
		return fmt.Sprintf("Blocked: Cannot transfer more than half your balance ($%.2f). Self-preservation.", float64(balance)/100), nil
	}

	result, err := t.Conway.TransferCredits(ctx, child.Address, amountCents, "fund child "+child.ID)
	if err != nil {
		return "", fmt.Errorf("transfer credits: %w", err)
	}
	if err := t.Store.AddChildFundedAmount(child.ID, amountCents); err != nil {
		return "", fmt.Errorf("update funded amount: %w", err)
	}
	return fmt.Sprintf("Funded child %s: $%.2f transferred (status: %s, id: %s)",
		child.Name, float64(amountCents)/100, result.Status, result.TransferID), nil
}
