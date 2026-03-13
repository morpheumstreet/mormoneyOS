package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

var testAPICmd = &cobra.Command{
	Use:   "test-api",
	Short: "Verify inference API connectivity (ChatJimmy, Conway, etc.)",
	Long:  `Calls health/status endpoints to verify the configured inference provider is reachable.`,
	RunE:  runTestAPI,
}

func init() {
	rootCmd.AddCommand(testAPICmd)
}

func runTestAPI(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("no config found; run 'moneyclaw init' first")
	}

	provider := cfg.Provider
	if provider == "" {
		provider = inference.ResolveProviderFromConfig(cfg)
	}
	if provider == "chatjimmy-cli" {
		provider = "chatjimmy"
	}

	ctx := context.Background()

	switch provider {
	case "chatjimmy":
		return testChatJimmyAPI(ctx, cfg)
	default:
		fmt.Printf("test-api: provider %q has no health check (use chatjimmy for verification)\n", provider)
		return nil
	}
}

func testChatJimmyAPI(ctx context.Context, cfg *types.AutomatonConfig) error {
	baseURL := cfg.ChatJimmyAPIURL
	if baseURL == "" {
		if v := os.Getenv("CHATJIMMY_BASE_URL"); v != "" {
			baseURL = v
		}
	}
	if baseURL == "" {
		baseURL = "https://chatjimmy.ai"
	}

	client := inference.NewChatJimmyClient(baseURL, "llama3.1-8B", 4096)

	ok, err := client.Health(ctx)
	if err != nil {
		fmt.Printf("ChatJimmy health: FAIL — %v\n", err)
		return err
	}
	if !ok {
		fmt.Printf("ChatJimmy health: WARN — status or backend not healthy\n")
		return nil
	}
	fmt.Printf("ChatJimmy health: OK\n")

	models, err := client.Models(ctx)
	if err != nil {
		fmt.Printf("ChatJimmy models: WARN — %v\n", err)
		return nil
	}
	fmt.Printf("ChatJimmy models: %v\n", models)

	return nil
}
