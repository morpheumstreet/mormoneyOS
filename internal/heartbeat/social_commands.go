package heartbeat

import (
	"context"
	"fmt"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/morpheumlabs/mormoneyos-go/internal/social"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// HandleSocialCommand processes slash commands (Type 2 / programmatic replies).
// Returns response text for immediate send via ch.Send in check_social_inbox.
// No LLM, no agent wake — always on time. Must NOT be saved to inbox_messages.
// OpenClaw-aligned: /status, /help, /pause, /resume, /reset.
func HandleSocialCommand(ctx context.Context, tc *TaskContext, m social.InboxMessage) (response string, handled bool) {
	cmd, args := social.ParseCommand(m.Content)
	if cmd == "" {
		return "", false
	}

	db, _ := tc.DB.(*state.Database)

	switch cmd {
	case "ping":
		return "pong", true

	case "status":
		agentState := "unknown"
		if tc.DB != nil {
			if s, ok, _ := tc.DB.GetAgentState(); ok {
				agentState = s
			}
		}
		turnCount := int64(0)
		if db != nil {
			turnCount, _ = db.GetTurnCount()
		}
		credits := int64(0)
		if tc.Tick != nil {
			credits = tc.Tick.CreditBalance
		}
		tier := "unknown"
		if tc.Tick != nil {
			tier = string(tc.Tick.SurvivalTier)
		}
		return fmt.Sprintf("Status: %s | Turns: %d | Credits: %d¢ | Tier: %s",
			strings.ToUpper(agentState), turnCount, credits, tier), true

	case "help":
		return `Commands (also: ping, status, credits?, !cmd <x>):
/status — agent state, turns, credits, tier
/balance — economic status, USDC by wallet
/skill — list all skills (markdown)
/help — this message
/pause — pause agent (dashboard-style)
/resume — resume agent
/reset — request context reset (wake agent)`, true

	case "skill", "skills":
		if db == nil {
			return "Skills not available.", true
		}
		skills, err := db.GetAllSkills()
		if err != nil || len(skills) == 0 {
			return "No skills installed.", true
		}
		var sb strings.Builder
		sb.WriteString("## Skills\n\n")
		sb.WriteString("| Name | Description | Status |\n")
		sb.WriteString("|------|-------------|--------|\n")
		for _, s := range skills {
			status := "disabled"
			if s.Enabled {
				status = "✓ enabled"
			}
			desc := s.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			// Escape pipe in markdown table
			name := strings.ReplaceAll(s.Name, "|", "\\|")
			desc = strings.ReplaceAll(desc, "|", "\\|")
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", name, desc, status))
		}
		return sb.String(), true

	case "balance", "balances":
		return formatBalanceCommand(ctx, tc, db), true

	case "pause":
		if tc.DB == nil {
			return "Pause not available.", true
		}
		_ = tc.DB.SetAgentState("sleeping")
		_ = tc.DB.SetKV("sleep_until", "2099-12-31T23:59:59.000Z")
		return "Agent paused.", true

	case "resume":
		if tc.DB == nil {
			return "Resume not available.", true
		}
		_ = tc.DB.DeleteKV("sleep_until")
		_ = tc.DB.InsertWakeEvent("social", "resume from "+m.Channel)
		return "Agent resumed.", true

	case "reset":
		if tc.DB == nil {
			return "Reset not available.", true
		}
		_ = tc.DB.InsertWakeEvent("social", "reset requested from "+m.Channel)
		return "Reset requested. Agent will clear context on next turn.", true

	case "compact":
		return "Compact is a dashboard feature. Use the web UI.", true

	case "think":
		if args != "" {
			return fmt.Sprintf("Think level '%s' — set via dashboard or config.", args), true
		}
		return "Usage: /think off|minimal|low|medium|high", true

	case "verbose":
		return "Verbose — set via dashboard.", true

	case "usage":
		return "Usage — set via dashboard.", true

	case "activation":
		if args != "" {
			return fmt.Sprintf("Activation '%s' — group-only; set via config.", args), true
		}
		return "Usage: /activation mention|always (groups only)", true

	case "restart":
		return "Restart requires dashboard or process restart.", true

	default:
		// Unknown command — don't claim as handled so it goes to inbox
		return "", false
	}
}

// formatBalanceCommand builds markdown for /balance: credits, USDC by wallet.
func formatBalanceCommand(ctx context.Context, tc *TaskContext, db *state.Database) string {
	var sb strings.Builder

	// Credits (Conway)
	credits := int64(0)
	if tc.Tick != nil {
		credits = tc.Tick.CreditBalance
	}
	tier := "unknown"
	if tc.Tick != nil {
		tier = string(tc.Tick.SurvivalTier)
	}
	sb.WriteString("## Economic Status\n\n")
	sb.WriteString(fmt.Sprintf("- **Credits:** $%.2f (%d¢)\n", float64(credits)/100, credits))
	sb.WriteString(fmt.Sprintf("- **Tier:** %s\n\n", tier))

	// Collect addresses: config, identity, children
	type addrEntry struct {
		Address string
		Chain   string
		Source  string
	}
	seen := make(map[string]bool)
	var entries []addrEntry
	add := func(addr, chain, source string) {
		if addr == "" || !strings.HasPrefix(addr, "0x") {
			return
		}
		key := strings.ToLower(addr) + "|" + chain
		if seen[key] {
			return
		}
		seen[key] = true
		entries = append(entries, addrEntry{Address: addr, Chain: chain, Source: source})
	}

	chains := []string{"eip155:8453", "eip155:84532", "eip155:1", "eip155:137", "eip155:42161"}
	var providers map[string]conway.USDCChainProvider
	if tc.Config != nil && len(tc.Config.ChainProviders) > 0 {
		chains = make([]string, 0, len(tc.Config.ChainProviders))
		providers = make(map[string]conway.USDCChainProvider)
		for ch, cp := range tc.Config.ChainProviders {
			if cp.RPCURL != "" && cp.USDCAddress != "" {
				chains = append(chains, ch)
				providers[ch] = conway.USDCChainProvider{RPCURL: cp.RPCURL, USDCAddress: cp.USDCAddress}
			}
		}
	}
	defaultChain := identity.DefaultChainBase
	if tc.Config != nil && tc.Config.DefaultChain != "" {
		defaultChain = tc.Config.DefaultChain
	}

	if tc.Config != nil {
		if tc.Config.WalletAddress != "" {
			add(tc.Config.WalletAddress, defaultChain, "wallet")
		}
		if tc.Config.CreatorAddress != "" && tc.Config.CreatorAddress != tc.Config.WalletAddress {
			add(tc.Config.CreatorAddress, defaultChain, "creator")
		}
	}
	if tc.Address != "" {
		add(tc.Address, defaultChain, "identity")
	}
	if db != nil {
		if a, ok, _ := db.GetIdentity("address"); ok && a != "" {
			add(a, defaultChain, "identity")
		}
		for _, ch := range chains {
			if a, ok, _ := db.GetIdentity(identity.AddressKeyForChain(ch)); ok && a != "" {
				add(a, ch, "identity")
			}
		}
		children, _ := db.GetAllChildren()
		for _, c := range children {
			if c.Address != "" {
				ch := c.Chain
				if ch == "" {
					ch = defaultChain
				}
				add(c.Address, ch, "child")
			}
		}
	}

	if len(entries) == 0 {
		sb.WriteString("No wallets configured.")
		return sb.String()
	}

	sb.WriteString("## Wallets (USDC)\n\n")
	sb.WriteString("| Address | Chain | Source | USDC |\n")
	sb.WriteString("|---------|-------|--------|------|\n")

	totalUSDC := 0.0
	for _, e := range entries {
		if !identity.IsEVM(e.Chain) {
			continue
		}
		results, err := conway.GetUSDCBalanceMulti(ctx, e.Address, []string{e.Chain}, providers)
		bal := 0.0
		errStr := ""
		if err != nil {
			errStr = err.Error()
		} else if len(results) > 0 {
			bal = results[0].Balance
			totalUSDC += bal
		}
		shortAddr := e.Address
		if len(shortAddr) > 14 {
			shortAddr = shortAddr[:6] + "…" + shortAddr[len(shortAddr)-4:]
		}
		chainShort := strings.TrimPrefix(e.Chain, "eip155:")
		if errStr != "" {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | error |\n", shortAddr, chainShort, e.Source))
		} else {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | $%.2f |\n", shortAddr, chainShort, e.Source, bal))
		}
	}
	sb.WriteString(fmt.Sprintf("\n**Total USDC:** $%.2f", totalUSDC))
	return sb.String()
}
