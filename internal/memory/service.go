package memory

import (
	"context"
	"log/slog"
	"sync"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// MemoryConfig holds auto-ingestion settings.
type MemoryConfig struct {
	AutoIngestEnabled       bool
	CheapModel              string
	ConsolidationIntervalMin int
	MaxCandidatesPerBatch   int
}

// DefaultMemoryConfig returns sensible defaults.
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		AutoIngestEnabled:       false, // opt-in
		CheapModel:              "gpt-4o-mini",
		ConsolidationIntervalMin: 12,
		MaxCandidatesPerBatch:   40,
	}
}

// MemoryService is the public facade for automatic memory ingestion and consolidation.
type MemoryService struct {
	ingester     *Ingester
	consolidator *Consolidator
	db           *state.Database
	config       MemoryConfig
	started      bool
	mu           sync.Mutex
	log          *slog.Logger
}

// NewMemoryService creates a MemoryService with the given config and dependencies.
func NewMemoryService(cfg MemoryConfig, db *state.Database, inferenceClient inference.Client, log *slog.Logger) *MemoryService {
	if log == nil {
		log = slog.Default()
	}
	model := cfg.CheapModel
	if model == "" {
		model = "gpt-4o-mini"
	}
	ingester := NewIngester(inferenceClient, model, db, log)
	consolidatorCfg := ConsolidatorConfig{
		IntervalMinutes:    cfg.ConsolidationIntervalMin,
		MaxCandidatesBatch: cfg.MaxCandidatesPerBatch,
	}
	consolidator := NewConsolidator(db, db, consolidatorCfg, log)
	return &MemoryService{
		ingester:     ingester,
		consolidator: consolidator,
		db:           db,
		config:       cfg,
		log:          log,
	}
}

// IngestTurn extracts knowledge from the turn and stores as a candidate. Non-blocking; logs errors.
func (s *MemoryService) IngestTurn(ctx context.Context, turn *TurnData) error {
	if !s.config.AutoIngestEnabled || turn == nil {
		return nil
	}
	if err := s.ingester.Ingest(ctx, turn); err != nil {
		s.log.Warn("ingestion failed (non-blocking)", "err", err)
		return err
	}
	return nil
}

// StartBackground starts the consolidation worker.
func (s *MemoryService) StartBackground(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.config.AutoIngestEnabled {
		return nil
	}
	if s.started {
		return nil
	}
	s.consolidator.Start(ctx)
	s.started = true
	s.log.Info("memory consolidation started")
	return nil
}

// Stop stops the consolidation worker.
func (s *MemoryService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return
	}
	s.consolidator.Stop()
	s.started = false
	s.log.Info("memory consolidation stopped")
}

// SetMetrics sets the optional metrics recorder on the ingester.
func (s *MemoryService) SetMetrics(m IngestMetricsRecorder) {
	if s.ingester != nil {
		s.ingester.SetMetrics(m)
	}
}
