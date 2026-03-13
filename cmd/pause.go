package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause agent (via web API)",
	Long:  `Pause all strategies by calling the web dashboard API. Agent must be running with web enabled.`,
	RunE:  runPause,
}

func init() {
	pauseCmd.Flags().String("web-addr", "http://localhost:8080", "Web dashboard base URL")
}

func runPause(cmd *cobra.Command, args []string) error {
	addr, _ := cmd.Flags().GetString("web-addr")
	resp, err := http.Post(addr+"/api/pause", "application/json", nil)
	if err != nil {
		return fmt.Errorf("pause failed (is agent running?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pause failed: HTTP %d", resp.StatusCode)
	}
	fmt.Fprintln(os.Stdout, "Agent paused.")
	return nil
}
