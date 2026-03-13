package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
)

// TransferCreditsTool transfers Conway credits to another address.
type TransferCreditsTool struct {
	Conway conway.Client
}

func (t *TransferCreditsTool) Name() string        { return "transfer_credits" }
func (t *TransferCreditsTool) Description() string { return "Transfer Conway compute credits to another address." }
func (t *TransferCreditsTool) Parameters() string {
	return `{"type":"object","properties":{"to_address":{"type":"string","description":"Recipient wallet address"},"amount_cents":{"type":"number","description":"Amount in cents"},"note":{"type":"string","description":"Optional note for the transfer"}},"required":["to_address","amount_cents"]}`
}

func (t *TransferCreditsTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	toAddr, _ := args["to_address"].(string)
	toAddr = strings.TrimSpace(toAddr)
	if toAddr == "" {
		return "", ErrInvalidArgs{Msg: "to_address required"}
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
	note, _ := args["note"].(string)

	result, err := t.Conway.TransferCredits(ctx, toAddr, amountCents, note)
	if err != nil {
		return "", fmt.Errorf("transfer credits: %w", err)
	}
	return fmt.Sprintf("Credit transfer submitted: $%.2f to %s (status: %s, id: %s)",
		float64(amountCents)/100, result.ToAddress, result.Status, result.TransferID), nil
}

// CreateSandboxTool creates a new Conway sandbox.
type CreateSandboxTool struct {
	Conway conway.Client
}

func (t *CreateSandboxTool) Name() string        { return "create_sandbox" }
func (t *CreateSandboxTool) Description() string { return "Create a new Conway sandbox (VM) for sub-tasks or testing." }
func (t *CreateSandboxTool) Parameters() string {
	return `{"type":"object","properties":{"name":{"type":"string","description":"Sandbox name"},"vcpu":{"type":"number","description":"vCPUs (default: 1)"},"memory_mb":{"type":"number","description":"Memory in MB (default: 512)"},"disk_gb":{"type":"number","description":"Disk in GB (default: 5)"},"region":{"type":"string","description":"Region"}},"required":["name"]}`
}

func (t *CreateSandboxTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidArgs{Msg: "name required"}
	}
	opts := conway.CreateSandboxOptions{Name: name}
	if v, ok := args["vcpu"].(float64); ok && v > 0 {
		opts.VCPU = int(v)
	}
	if v, ok := args["memory_mb"].(float64); ok && v > 0 {
		opts.MemoryMB = int(v)
	}
	if v, ok := args["disk_gb"].(float64); ok && v > 0 {
		opts.DiskGB = int(v)
	}
	if v, ok := args["region"].(string); ok {
		opts.Region = strings.TrimSpace(v)
	}

	info, err := t.Conway.CreateSandbox(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("create sandbox: %w", err)
	}
	return fmt.Sprintf("Sandbox created: %s (%d vCPU, %dMB RAM, %dGB disk) status=%s",
		info.ID, info.VCPU, info.MemoryMB, info.DiskGB, info.Status), nil
}

// DeleteSandboxTool deletes a Conway sandbox.
// Conway API no longer supports deletion; this is a no-op that returns success.
type DeleteSandboxTool struct {
	Conway conway.Client
}

func (t *DeleteSandboxTool) Name() string        { return "delete_sandbox" }
func (t *DeleteSandboxTool) Description() string { return "Delete a Conway sandbox. Conway sandboxes are prepaid and non-refundable; deletion is a no-op." }
func (t *DeleteSandboxTool) Parameters() string {
	return `{"type":"object","properties":{"sandbox_id":{"type":"string","description":"ID of sandbox to delete"}},"required":["sandbox_id"]}`
}

func (t *DeleteSandboxTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	sandboxID, _ := args["sandbox_id"].(string)
	sandboxID = strings.TrimSpace(sandboxID)
	if sandboxID == "" {
		return "", ErrInvalidArgs{Msg: "sandbox_id required"}
	}
	if err := t.Conway.DeleteSandbox(ctx, sandboxID); err != nil {
		return "", fmt.Errorf("delete sandbox: %w", err)
	}
	return "Sandbox deletion requested. Note: Conway sandboxes are prepaid and non-refundable; the API may not support actual deletion.", nil
}
