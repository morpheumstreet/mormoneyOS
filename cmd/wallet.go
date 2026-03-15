package cmd

import (
	"fmt"
	"os"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/spf13/cobra"
)

var (
	rotateToIndex uint32
	rotatePreview bool
	rotateConfirm bool
)

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Wallet management",
	Long:  `Manage mnemonic wallet: rotate HD account index, show addresses.`,
}

var walletRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate HD account index",
	Long: `Update the HD account index (derivation path) for the wallet.
Shows new addresses per chain. Use --preview to see without writing.
Use --confirm to actually update wallet.json.
Does NOT sweep funds — operator must migrate balances manually.`,
	RunE: runWalletRotate,
}

func init() {
	walletRotateCmd.Flags().Uint32Var(&rotateToIndex, "to-index", 0, "Target HD account index")
	walletRotateCmd.Flags().BoolVar(&rotatePreview, "preview", false, "Show new addresses without writing")
	walletRotateCmd.Flags().BoolVar(&rotateConfirm, "confirm", false, "Confirm: write new index to wallet.json")

	walletCmd.AddCommand(walletRotateCmd)
	rootCmd.AddCommand(walletCmd)
}

func runWalletRotate(cmd *cobra.Command, args []string) error {
	if !identity.WalletExists() {
		return fmt.Errorf("no wallet: run 'moneyclaw init' first")
	}
	currentIdx := identity.CurrentIndex()
	if rotateToIndex == currentIdx {
		return fmt.Errorf("index already %d; choose a different --to-index", currentIdx)
	}

	// Collect chains to show: defaultChain + chainProviders
	cfg, _ := config.Load()
	seen := make(map[string]bool)
	var unique []string
	if cfg != nil && cfg.DefaultChain != "" {
		unique = append(unique, cfg.DefaultChain)
		seen[cfg.DefaultChain] = true
	}
	if len(unique) == 0 {
		unique = append(unique, identity.DefaultChainBase)
		seen[identity.DefaultChainBase] = true
	}
	if cfg != nil {
		for c := range cfg.ChainProviders {
			if c != "" && !seen[c] {
				unique = append(unique, c)
				seen[c] = true
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Current index: %d\n", currentIdx)
	fmt.Fprintf(os.Stderr, "Target index:  %d\n\n", rotateToIndex)

	// Show current addresses
	fmt.Fprintln(os.Stderr, "Current addresses:")
	for _, chain := range unique {
		addr, err := identity.DeriveAddressAtExplicitIndex(chain, currentIdx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: (error: %v)\n", chain, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "  %s: %s\n", chain, addr)
	}

	// Show new addresses
	fmt.Fprintln(os.Stderr, "\nNew addresses (after rotate):")
	for _, chain := range unique {
		addr, err := identity.DeriveAddressAtExplicitIndex(chain, rotateToIndex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: (error: %v)\n", chain, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "  %s: %s\n", chain, addr)
	}

	if rotatePreview {
		fmt.Fprintln(os.Stderr, "\n[Preview only — no changes written. Use --confirm to apply.]")
		return nil
	}
	if !rotateConfirm {
		fmt.Fprintln(os.Stderr, "\n[Use --confirm to write new index to wallet.json]")
		return nil
	}

	if err := identity.RotateIndex(rotateToIndex, false); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "\nRotated to index %d. ClearDerivedKeys called.\n", rotateToIndex)
	fmt.Fprintln(os.Stderr, "You must re-fund / migrate balances to the new addresses manually.")
	return nil
}
