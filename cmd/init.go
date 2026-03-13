package cmd

import (
	"fmt"
	"os"

	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize wallet and automaton directory",
	Long:  `Create ~/.automaton, generate wallet if missing, and print address.`,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	acc, isNew, err := identity.GetWallet()
	if err != nil {
		return fmt.Errorf("wallet: %w", err)
	}
	if isNew {
		fmt.Fprintf(os.Stderr, "Created new wallet at %s\n", identity.GetWalletPath())
	}
	fmt.Println(acc.Address())
	return nil
}
