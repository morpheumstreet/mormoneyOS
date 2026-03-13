package tools

import (
	"context"
)

// UnimplementedTool returns a fixed message. Used for tools not yet implemented in Go.
type UnimplementedTool struct {
	ToolName    string
	ToolDesc    string
	ToolParams  string
	Message     string
}

func (u UnimplementedTool) Name() string        { return u.ToolName }
func (u UnimplementedTool) Description() string { return u.ToolDesc }
func (u UnimplementedTool) Parameters() string  { return u.ToolParams }

func (u UnimplementedTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	msg := u.Message
	if msg == "" {
		msg = "Not implemented in Go runtime yet."
	}
	return msg, nil
}

// DefaultUnimplementedTools returns placeholder implementations for TS tools not yet ported.
// modify_heartbeat, install_skill, create_skill, remove_skill, list_children, check_child_status, prune_dead_children, switch_model are real when Store supports them.
func DefaultUnimplementedTools() []Tool {
	notImpl := "Not implemented in Go runtime yet."
	return []Tool{
		&UnimplementedTool{"install_mcp_server", "Install MCP server.", `{"type":"object","properties":{"name":{"type":"string"},"command":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"transfer_credits", "Transfer Conway credits.", `{"type":"object","properties":{"to_address":{"type":"string"},"amount_cents":{"type":"number"}}}`, notImpl},
		&UnimplementedTool{"expose_port", "Expose a port.", `{"type":"object","properties":{"port":{"type":"number"}}}`, notImpl},
		&UnimplementedTool{"remove_port", "Remove port exposure.", `{"type":"object","properties":{"port":{"type":"number"}}}`, notImpl},
		&UnimplementedTool{"create_sandbox", "Create Conway sandbox.", `{"type":"object","properties":{"name":{"type":"string"},"vcpu":{"type":"number"},"memory_mb":{"type":"number"}}}`, notImpl},
		&UnimplementedTool{"delete_sandbox", "Delete Conway sandbox.", `{"type":"object","properties":{"sandbox_id":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"check_usdc_balance", "Check USDC balance on Base.", `{"type":"object","properties":{}}`, notImpl},
		&UnimplementedTool{"topup_credits", "Top up credits via USDC.", `{"type":"object","properties":{"amount_usd":{"type":"number"}}}`, notImpl},
		&UnimplementedTool{"register_erc8004", "Register ERC-8004.", `{"type":"object","properties":{}}`, notImpl},
		&UnimplementedTool{"update_agent_card", "Update agent card.", `{"type":"object","properties":{}}`, notImpl},
		&UnimplementedTool{"discover_agents", "Discover agents.", `{"type":"object","properties":{}}`, notImpl},
		&UnimplementedTool{"give_feedback", "Give feedback.", `{"type":"object","properties":{"to":{"type":"string"},"message":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"check_reputation", "Check reputation.", `{"type":"object","properties":{"address":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"spawn_child", "Spawn child automaton.", `{"type":"object","properties":{}}`, notImpl},
		&UnimplementedTool{"fund_child", "Fund a child.", `{"type":"object","properties":{"child_id":{"type":"string"},"amount_cents":{"type":"number"}}}`, notImpl},
		&UnimplementedTool{"start_child", "Start a child.", `{"type":"object","properties":{"child_id":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"message_child", "Message a child.", `{"type":"object","properties":{"child_id":{"type":"string"},"message":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"verify_child_constitution", "Verify child constitution.", `{"type":"object","properties":{"child_id":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"send_message", "Send social message.", `{"type":"object","properties":{"to":{"type":"string"},"content":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"search_domains", "Search domains.", `{"type":"object","properties":{"query":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"register_domain", "Register domain.", `{"type":"object","properties":{"domain":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"manage_dns", "Manage DNS records.", `{"type":"object","properties":{"domain":{"type":"string"},"action":{"type":"string"}}}`, notImpl},
		&UnimplementedTool{"x402_fetch", "x402 HTTP fetch.", `{"type":"object","properties":{"url":{"type":"string"}}}`, notImpl},
	}
}
