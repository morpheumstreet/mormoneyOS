package inference

import (
	"context"
	"log/slog"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

func TestModelRouter_Select(t *testing.T) {
	cfg := &types.AutomatonConfig{
		Provider:         "chatjimmy",
		InferenceModel:   "llama3.1-8B",
		MaxTokensPerTurn: 4096,
	}
	holder := NewInferenceClientHolder(cfg)
	router := NewModelRouter(cfg, holder, nil, slog.Default())

	dc := DecisionContext{
		TokensUsed:     1000,
		RiskLevel:     RiskLow,
		HasMoneyImpact: false,
		TurnPhase:      "action",
	}
	client, err := router.Select(context.Background(), dc)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}
