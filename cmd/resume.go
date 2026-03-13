package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume agent",
	Long:  `Resume operations by calling the web dashboard API. Agent must be running with web enabled.`,
	RunE:  runResume,
}

func init() {
	resumeCmd.Flags().String("web-addr", "http://localhost:8080", "Web dashboard base URL")
}

func runResume(cmd *cobra.Command, args []string) error {
	addr, _ := cmd.Flags().GetString("web-addr")
	resp, err := http.Post(addr+"/api/resume", "application/json", nil)
	if err != nil {
		return fmt.Errorf("resume failed (is agent running?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resume failed: HTTP %d", resp.StatusCode)
	}
	fmt.Fprintln(os.Stdout, "Agent resumed.")
	return nil
}
