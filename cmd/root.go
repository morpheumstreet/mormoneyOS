package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagRun        = "run"
	flagSetup      = "setup"
	flagConfigure  = "configure"
	flagStatus     = "status"
	flagInit       = "init"
	flagVersion    = "version"
	envConwayURL   = "CONWAY_API_URL"
	envConwayKey   = "CONWAY_API_KEY"
)

var (
	version   = "0.1.0"
	buildTime = ""
	commit    = ""
)

var rootCmd = &cobra.Command{
	Use:   "moneyclaw",
	Short: "7x24 AI Agent that saves and makes money autonomously",
	Long: `MoneyClaw — 7x24 AI Agent for financial optimization.

Inspired by OpenClaw. Focuses on scanning for opportunities, executing strategies,
and minimizing operating costs. Aligned with moneyclaw-py design.

Subcommands:
  run        Start runtime (agent loop + heartbeat + web dashboard)
  setup      Interactive setup wizard
  status     Show runtime status
  strategies List discovered strategies
  cost       Show LLM cost summary
  pause      Pause agent (via web API)
  resume     Resume agent
  init       Initialize config directory
  test-api      Verify inference API connectivity (ChatJimmy, etc.)
  test-telegram Verify Telegram bot connectivity and message flow
`,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().Bool(flagRun, false, "Start runtime")
	rootCmd.PersistentFlags().Bool(flagSetup, false, "Run setup wizard")
	rootCmd.PersistentFlags().Bool(flagConfigure, false, "Edit config")
	rootCmd.PersistentFlags().Bool(flagStatus, false, "Show status")
	rootCmd.PersistentFlags().Bool(flagInit, false, "Initialize directory")

	_ = viper.BindPFlag(flagRun, rootCmd.PersistentFlags().Lookup(flagRun))
	_ = viper.BindPFlag(flagSetup, rootCmd.PersistentFlags().Lookup(flagSetup))
	_ = viper.BindPFlag(flagConfigure, rootCmd.PersistentFlags().Lookup(flagConfigure))
	_ = viper.BindPFlag(flagStatus, rootCmd.PersistentFlags().Lookup(flagStatus))
	_ = viper.BindPFlag(flagInit, rootCmd.PersistentFlags().Lookup(flagInit))

	_ = viper.BindEnv("conwayApiUrl", envConwayURL)
	_ = viper.BindEnv("conwayApiKey", envConwayKey)

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(simCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(strategiesCmd)
	rootCmd.AddCommand(costCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(provisionCmd)

	rootCmd.Version = version
	if buildTime != "" || commit != "" {
		tmpl := "moneyclaw {{.Version}}"
		if buildTime != "" {
			tmpl += " built " + buildTime
		}
		if commit != "" {
			tmpl += " " + commit
		}
		rootCmd.SetVersionTemplate(tmpl + "\n")
	}
}

func initConfig() {
	viper.SetEnvPrefix("AUTOMATON")
	viper.AutomaticEnv()
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
