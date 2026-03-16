package agent

import (
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestTokenLimitsFromConfig(t *testing.T) {
	cfg := &types.AutomatonConfig{
		MaxInputTokens:   6000,
		MaxHistoryTurns:  15,
		WarnAtTokens:    5000,
	}
	limits := TokenLimitsFromConfig(cfg)
	if limits == nil {
		t.Fatal("expected non-nil limits")
	}
	if limits.MaxInputTokens != 6000 {
		t.Errorf("MaxInputTokens: got %d", limits.MaxInputTokens)
	}
	if limits.MaxHistoryTurns != 15 {
		t.Errorf("MaxHistoryTurns: got %d", limits.MaxHistoryTurns)
	}
	if limits.WarnAtTokens != 5000 {
		t.Errorf("WarnAtTokens: got %d", limits.WarnAtTokens)
	}
}

func TestTokenLimitsFromConfig_Nil(t *testing.T) {
	if TokenLimitsFromConfig(nil) != nil {
		t.Error("expected nil for nil config")
	}
}

func TestBuildLoopConfig(t *testing.T) {
	cfg := &types.AutomatonConfig{
		Name:           "test",
		GenesisPrompt:  "be helpful",
		InferenceModel: "gpt-4",
		WalletAddress:  "0xabc",
	}
	loopCfg := BuildLoopConfig(cfg, nil)
	if loopCfg.Name != "test" {
		t.Errorf("Name: got %s", loopCfg.Name)
	}
	if loopCfg.InferenceModel != "gpt-4" {
		t.Errorf("InferenceModel: got %s", loopCfg.InferenceModel)
	}
	if loopCfg.WalletAddress != "0xabc" {
		t.Errorf("WalletAddress: got %s", loopCfg.WalletAddress)
	}
}

func TestBuildLoopConfig_Overrides(t *testing.T) {
	cfg := &types.AutomatonConfig{
		Name:           "test",
		InferenceModel: "gpt-4",
		WalletAddress:  "0xabc",
	}
	opts := &BuildLoopConfigOpts{
		WalletAddress:  "0xoverride",
		InferenceModel: "sim-stub",
	}
	loopCfg := BuildLoopConfig(cfg, opts)
	if loopCfg.WalletAddress != "0xoverride" {
		t.Errorf("WalletAddress: got %s", loopCfg.WalletAddress)
	}
	if loopCfg.InferenceModel != "sim-stub" {
		t.Errorf("InferenceModel: got %s", loopCfg.InferenceModel)
	}
}
