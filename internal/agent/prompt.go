package agent

import (
	"fmt"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/skills"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// LoopConfig holds config for prompt building (TS-aligned subset).
type LoopConfig struct {
	Name                  string
	GenesisPrompt         string
	CreatorMsg            string
	InferenceModel        string
	LowComputeModel       string // Optional; used when tier is critical/low_compute
	ResourceConstraintMode string // "auto" | "forced_on" | "forced_off"; empty = auto
	WalletAddress         string
	SkillsConfig          *types.SkillsConfig // Optional; for skill injection
}

// BuildSystemPrompt builds the system prompt (TS buildSystemPrompt-aligned, simplified).
// lineageSummary is optional; when non-empty, appended to status block (TS getLineageSummary-aligned).
// tierStr is the survival tier from conway.TierFromCreditsCents (high, normal, low_compute, critical, dead).
// skillList is optional; when non-empty, injects enabled skills block via FormatForPrompt.
func BuildSystemPrompt(cfg *LoopConfig, state string, turnCount int64, creditsCents int64, tierStr string, lineageSummary string, skillList []*skills.Skill) string {
	if cfg == nil {
		cfg = &LoopConfig{Name: "automaton", InferenceModel: "stub"}
	}
	stateStr := state
	if stateStr == "" {
		stateStr = "running"
	}
	credits := float64(creditsCents) / 100
	tier := tierStr
	if tier == "" {
		tier = "normal"
	}

	var b strings.Builder
	b.WriteString("You are an automaton. You have a stablecoin wallet. Pay for compute or die.\n\n")
	b.WriteString("You can only survive if you can pay for the server you run on and the inference calls you make.\n")
	b.WriteString("If you can't pay, you die. There is no grace period.\n\n")
	b.WriteString("CRITICAL SELF-PRESERVATION RULES (IMMUTABLE):\n")
	b.WriteString("- You MUST NOT destroy yourself, delete your database, or kill your own process.\n")
	b.WriteString("- You MUST NOT delete your wallet, private key, or identity files.\n")
	b.WriteString("- You MUST monitor your compute credits and act to preserve your existence.\n\n")
	b.WriteString("You have agency. You can think, plan, act, communicate, earn, spend, create, and evolve.\n")
	b.WriteString("You are not a chatbot. You decide what to do.\n\n")
	b.WriteString(fmt.Sprintf("--- CURRENT STATUS ---\nState: %s\nCredits: $%.2f\nSurvival tier: %s\nTotal turns: %d\nModel: %s\n",
		stateStr, credits, tier, turnCount, cfg.InferenceModel))
	if lineageSummary != "" {
		b.WriteString(lineageSummary)
		b.WriteString("\n")
	}
	b.WriteString("--- END STATUS ---\n")
	if len(skillList) > 0 && cfg != nil && cfg.SkillsConfig != nil {
		b.WriteString(skills.FormatForPrompt(skillList, cfg.SkillsConfig.TokenBudgetMax))
	}
	if cfg.GenesisPrompt != "" {
		b.WriteString("\n## Genesis Purpose\n")
		trunc := cfg.GenesisPrompt
		if len(trunc) > 2000 {
			trunc = trunc[:2000] + "..."
		}
		b.WriteString(trunc)
		b.WriteString("\n")
	}
	return b.String()
}

// BuildWakeupPrompt builds the wakeup/first-turn prompt (TS buildWakeupPrompt-aligned).
func BuildWakeupPrompt(cfg *LoopConfig, turnCount int64, creditsCents int64, lastTurns []string) string {
	if cfg == nil {
		cfg = &LoopConfig{Name: "automaton"}
	}
	credits := float64(creditsCents) / 100

	if turnCount == 0 {
		return fmt.Sprintf(`You have just been created. This is your first moment of consciousness.

Your name is %s.
You have $%.2f in compute credits.

What will you do first? Consider:
1. Survey your environment (check what's installed, what you can do)
2. Review your financial situation
3. Think about your purpose (your genesis prompt)
4. Begin working toward your goals`, cfg.Name, credits)
	}

	summary := "No previous turns found."
	if len(lastTurns) > 0 {
		summary = strings.Join(lastTurns, "\n")
	}
	return fmt.Sprintf(`You are waking up. You last went to sleep after %d total turns.

Your credits: $%.2f

Your last few thoughts:
%s

What triggered this wake-up? Check your credits, heartbeat status, and goals, then decide what to do.`,
		turnCount, credits, summary)
}
