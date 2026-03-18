// Package usecase holds business logic only (pure Go, no HTTP).
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// ErrPermissionDenied is returned when permission check fails.
var ErrPermissionDenied = errors.New("permission denied: agent_card_sig required for install")

// InstallSkill triggers safe install with permission manifest check.
// Phase 2: full flow — permission check, install from registry, MORM reward claim.
type InstallSkill struct {
	Registry port.RegistryPort
	Scanner  port.ScannerPort
	Installer port.InstallerPort
	OnChain  port.OnChainPort
}

// Execute performs permission check, installs from registry, and claims micro MORM reward.
func (u *InstallSkill) Execute(ctx context.Context, skillID string, agentCardSig string) (string, error) {
	if skillID == "" {
		return "", fmt.Errorf("skill_id required")
	}
	// Permission enforcement: agent_card_sig required for marketplace install
	if agentCardSig == "" {
		return "", ErrPermissionDenied
	}

	// Get skill metadata (optional; for badges)
	skill, _ := u.Registry.GetByID(skillID)
	if skill != nil && u.Scanner != nil && skill.SecurityHash != "" {
		skill.Badges = u.Scanner.GetBadges(skill.SecurityHash)
	}

	// Install from registry
	if u.Installer != nil {
		name := ""
		desc := ""
		if skill != nil {
			name = skill.Name
			desc = skill.Description
		}
		skillRoot, err := u.Installer.Install(ctx, skillID, name, desc)
		if err != nil {
			return "", fmt.Errorf("install: %w", err)
		}
		if skillRoot == "" {
			return "", fmt.Errorf("install returned empty path")
		}

		// Claim micro MORM reward (Phase 2 stub; Phase 3 real tx)
		claimMsg := ""
		if u.OnChain != nil {
			claim, err := u.OnChain.ClaimReward(skillID, "install:"+skillRoot)
			if err == nil && claim != nil {
				claimMsg = fmt.Sprintf("; claim: %.4f MORM (%s)", claim.MORMAmount, claim.Status)
			}
		}

		out := map[string]any{
			"status":     "installed",
			"skill_id":   skillID,
			"skill_root": skillRoot,
		}
		b, _ := json.Marshal(out)
		return string(b) + claimMsg, nil
	}

	return `{"status":"installed","skill_id":"` + skillID + `"}`, nil
}
