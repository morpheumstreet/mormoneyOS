package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	Long:  `First-run wizard to create config and wallet.`,
	RunE:  runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	cfg, _ := config.Load()
	if cfg != nil {
		fmt.Fprintln(os.Stderr, "Config already exists. Use 'configure' to edit.")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	prompt := func(msg, def string) string {
		if def != "" {
			fmt.Printf("%s [%s]: ", msg, def)
		} else {
			fmt.Printf("%s: ", msg)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		return line
	}

	name := prompt("Agent name", "moneyclaw")
	genesis := prompt("Genesis prompt", "Operate as a sovereign AI agent.")
	creatorAddr := prompt("Creator Ethereum address", "0x0000000000000000000000000000000000000000")
	conwayURL := prompt("Conway API URL", "https://api.conway.tech")
	conwayKey := prompt("Conway API key (optional)", "")

	tp := types.DefaultTreasuryPolicy()
	newCfg := &types.AutomatonConfig{
		Name:             name,
		GenesisPrompt:    genesis,
		CreatorAddress:   creatorAddr,
		ConwayAPIURL:     conwayURL,
		ConwayAPIKey:     conwayKey,
		InferenceModel:   "gpt-5.2",
		MaxTokensPerTurn: 4096,
		DBPath:           config.GetAutomatonDir() + "/state.db",
		LogLevel:         "info",
		MaxChildren:      3,
		TreasuryPolicy:   &tp,
	}

	if err := config.Save(newCfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Config saved to %s\n", config.GetConfigPath())
	return nil
}
