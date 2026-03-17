package simulation

import "time"

// ChaosLevel controls chaos injection intensity.
type ChaosLevel string

const (
	ChaosNone   ChaosLevel = "none"
	ChaosLow    ChaosLevel = "low"
	ChaosMedium ChaosLevel = "medium"
	ChaosHigh   ChaosLevel = "high"
)

// SimulationConfig holds simulation/backtest settings.
type SimulationConfig struct {
	Days              int        `json:"days"`
	SpeedMultiplier   int        `json:"speedMultiplier"`   // e.g. 100 = 100x realtime
	ChaosLevel        ChaosLevel `json:"chaosLevel"`
	Seed              int64      `json:"seed"`
	MarketDataSource  string     `json:"marketDataSource"`  // e.g. "csv:./data/binance-2025.csv"
	ReportFormat      string     `json:"reportFormat"`       // "json" | "html"
	ReportOutputDir   string     `json:"reportOutputDir"`
	StartDate         time.Time  `json:"-"` // derived or from config
	Strategies        []string   `json:"strategies,omitempty"` // optional strategy filters
}

// DefaultSimulationConfig returns sensible defaults.
func DefaultSimulationConfig() SimulationConfig {
	return SimulationConfig{
		Days:             7,
		SpeedMultiplier:  100,
		ChaosLevel:       ChaosNone,
		Seed:             42,
		MarketDataSource: "",
		ReportFormat:     "json",
		ReportOutputDir:   "sim-results",
		StartDate:        time.Now().UTC().Truncate(24 * time.Hour),
	}
}
