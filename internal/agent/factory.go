package agent

import (
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// BuildLoopConfigOpts overrides specific LoopConfig fields (e.g. for sim mode).
type BuildLoopConfigOpts struct {
	WalletAddress  string // override cfg.WalletAddress
	InferenceModel string // override cfg.InferenceModel (e.g. "sim-stub")
}

// TokenLimitsFromConfig builds TokenLimits from AutomatonConfig.
func TokenLimitsFromConfig(cfg *types.AutomatonConfig) *TokenLimits {
	if cfg == nil {
		return nil
	}
	limits := DefaultTokenLimits().WithOverrides(cfg.MaxInputTokens, cfg.MaxHistoryTurns, cfg.WarnAtTokens)
	compressCfg := DefaultHistoryTrimmerConfig()
	compressCfg.HistoryBudget = limits.MaxInputTokens - 800
	limits.HistoryCompress = &compressCfg
	return &limits
}

// ContextLimitForModelFromConfig builds per-model context limit from AutomatonConfig.
func ContextLimitForModelFromConfig(cfg *types.AutomatonConfig) ContextLimitForModel {
	if cfg == nil || len(cfg.Models) == 0 {
		return nil
	}
	models := cfg.Models
	return func(modelID string) int {
		for _, m := range models {
			if m.ModelID == modelID || strings.HasSuffix(modelID, "/"+m.ModelID) || strings.HasSuffix(modelID, m.ModelID) {
				if m.ContextLimit > 0 {
					return m.ContextLimit
				}
				break
			}
		}
		return 0
	}
}

// BuildLoopConfig builds LoopConfig from AutomatonConfig with optional overrides.
func BuildLoopConfig(cfg *types.AutomatonConfig, opts *BuildLoopConfigOpts) *LoopConfig {
	if cfg == nil {
		return &LoopConfig{Name: "automaton", InferenceModel: "stub"}
	}
	walletAddr := cfg.WalletAddress
	inferenceModel := cfg.InferenceModel
	if opts != nil {
		if opts.WalletAddress != "" {
			walletAddr = opts.WalletAddress
		}
		if opts.InferenceModel != "" {
			inferenceModel = opts.InferenceModel
		}
	}
	return &LoopConfig{
		Name:                   cfg.Name,
		GenesisPrompt:          cfg.GenesisPrompt,
		CreatorMsg:             cfg.CreatorAddress,
		InferenceModel:         inferenceModel,
		LowComputeModel:        cfg.LowComputeModel,
		ResourceConstraintMode:  cfg.ResourceConstraintMode,
		WalletAddress:          walletAddr,
		SkillsConfig:           cfg.Skills,
		TokenLimits:            TokenLimitsFromConfig(cfg),
		ContextLimitForModel:   ContextLimitForModelFromConfig(cfg),
		PromptVersion:          cfg.PromptVersion,
		Routing:                cfg.Routing,
	}
}
