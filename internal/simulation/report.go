package simulation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Reporter generates simulation reports (JSON, HTML).
type Reporter struct {
	outputDir string
	format    string
}

// NewReporter creates a reporter for the given output dir and format.
func NewReporter(outputDir, format string) *Reporter {
	if outputDir == "" {
		outputDir = "sim-results"
	}
	if format == "" {
		format = "json"
	}
	return &Reporter{outputDir: outputDir, format: format}
}

// Generate writes the report to disk.
func (r *Reporter) Generate(result *RunResult) error {
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return err
	}

	if r.format == "json" || r.format == "html" {
		path := filepath.Join(r.outputDir, "sim-report.json")
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return err
		}
	}

	if r.format == "html" {
		path := filepath.Join(r.outputDir, "sim-report.html")
		html := r.buildHTML(result)
		if err := os.WriteFile(path, []byte(html), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reporter) buildHTML(result *RunResult) string {
	return `<!DOCTYPE html>
<html>
<head><title>Simulation Report</title>
<style>
body{font-family:system-ui;max-width:800px;margin:2rem auto;padding:0 1rem}
h1{color:#333}
pre{background:#f5f5f5;padding:1rem;overflow:auto}
.meta{color:#666;font-size:0.9rem}
</style>
</head>
<body>
<h1>mormoneyOS Simulation Report</h1>
<p class="meta">Generated ` + time.Now().UTC().Format(time.RFC3339) + `</p>
<p><strong>Total turns:</strong> ` + strconv.Itoa(result.TotalTurns) + `</p>
<p><strong>Start:</strong> ` + result.StartTime.Format(time.RFC3339) + `</p>
<p><strong>End:</strong> ` + result.EndTime.Format(time.RFC3339) + `</p>
<pre>` + mustJSON(result) + `</pre>
</body>
</html>`
}

func mustJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
