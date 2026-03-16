package memory

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// MemoryWriter writes consolidated items to the 5-tier tables.
type MemoryWriter interface {
	InsertEpisodicMemory(sessionID, eventType, summary, detail, outcome string, importance float64) error
	InsertSemanticMemory(category, key, value string, confidence float64, source string) error
	InsertProceduralMemory(name, description, steps string, successCount, failureCount int) error
	InsertRelationshipMemory(entityAddress, entityName, relType string, trustScore float64, interactionCount int) error
}

// IngestCandidateReader reads unprocessed candidates and marks them processed.
type IngestCandidateReader interface {
	GetUnprocessedIngestCandidates(limit int) ([]struct {
		ID             int64
		SessionID      string
		TurnID         string
		ExtractionJSON string
		Importance     float64
	}, error)
	MarkIngestCandidatesProcessed(ids []int64) error
}

// Consolidator runs background consolidation of ingest candidates into 5-tier memory.
type Consolidator struct {
	writer MemoryWriter
	reader IngestCandidateReader
	interval time.Duration
	maxBatch int
	stopCh  chan struct{}
	doneCh  chan struct{}
	log     *slog.Logger
	mu      sync.Mutex
}

// ConsolidatorConfig holds consolidation parameters.
type ConsolidatorConfig struct {
	IntervalMinutes   int
	MaxCandidatesBatch int
}

// DefaultConsolidatorConfig returns sensible defaults.
func DefaultConsolidatorConfig() ConsolidatorConfig {
	return ConsolidatorConfig{
		IntervalMinutes:    12,
		MaxCandidatesBatch: 40,
	}
}

// NewConsolidator creates a consolidator with the given store and config.
func NewConsolidator(writer MemoryWriter, reader IngestCandidateReader, cfg ConsolidatorConfig, log *slog.Logger) *Consolidator {
	if log == nil {
		log = slog.Default()
	}
	interval := time.Duration(cfg.IntervalMinutes) * time.Minute
	if interval < time.Minute {
		interval = 12 * time.Minute
	}
	maxBatch := cfg.MaxCandidatesBatch
	if maxBatch <= 0 {
		maxBatch = 40
	}
	return &Consolidator{
		writer:   writer,
		reader:   reader,
		interval: interval,
		maxBatch: maxBatch,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		log:      log,
	}
}

// Start begins the background consolidation loop.
func (c *Consolidator) Start(ctx context.Context) {
	go c.run(ctx)
}

// Stop signals the consolidator to stop and waits for it.
func (c *Consolidator) Stop() {
	close(c.stopCh)
	<-c.doneCh
}

func (c *Consolidator) run(ctx context.Context) {
	defer close(c.doneCh)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.consolidate(ctx)
		}
	}
}

func (c *Consolidator) consolidate(ctx context.Context) {
	candidates, err := c.reader.GetUnprocessedIngestCandidates(c.maxBatch)
	if err != nil {
		c.log.Warn("consolidator get candidates failed", "err", err)
		return
	}
	if len(candidates) == 0 {
		return
	}
	var processed []int64
	for _, cand := range candidates {
		var ext Extraction
		if err := json.Unmarshal([]byte(cand.ExtractionJSON), &ext); err != nil {
			c.log.Debug("consolidator parse failed", "id", cand.ID, "err", err)
			processed = append(processed, cand.ID)
			continue
		}
		c.applyExtraction(cand.SessionID, &ext)
		processed = append(processed, cand.ID)
	}
	if err := c.reader.MarkIngestCandidatesProcessed(processed); err != nil {
		c.log.Warn("consolidator mark processed failed", "err", err)
		return
	}
	c.log.Info("consolidator batch complete", "count", len(processed))
}

func (c *Consolidator) applyExtraction(sessionID string, ext *Extraction) {
	for _, f := range ext.Facts {
		if f.Category == "" || f.Key == "" || f.Value == "" {
			continue
		}
		conf := f.Confidence
		if conf <= 0 || conf > 1 {
			conf = 0.8
		}
		_ = c.writer.InsertSemanticMemory(f.Category, f.Key, f.Value, conf, "ingestion")
	}
	for _, e := range ext.Episodes {
		if e.Summary == "" {
			continue
		}
		imp := e.Importance
		if imp < 0 || imp > 1 {
			imp = 0.5
		}
		eventType := e.EventType
		if eventType == "" {
			eventType = "event"
		}
		_ = c.writer.InsertEpisodicMemory(sessionID, eventType, e.Summary, e.Detail, e.Outcome, imp)
	}
	for _, p := range ext.Procedures {
		if p.Name == "" {
			continue
		}
		steps := strings.Join(p.Steps, "\n")
		if steps == "" {
			steps = p.Name
		}
		success, fail := 0, 0
		if p.SuccessRate >= 0.5 {
			success = 1
		} else {
			fail = 1
		}
		_ = c.writer.InsertProceduralMemory(p.Name, "", steps, success, fail)
	}
	for _, r := range ext.Relationships {
		if r.EntityAddress == "" {
			continue
		}
		trust := 0.5 + r.TrustDelta
		if trust < 0 {
			trust = 0
		}
		if trust > 1 {
			trust = 1
		}
		_ = c.writer.InsertRelationshipMemory(r.EntityAddress, r.EntityName, r.Type, trust, 1)
	}
}

// Ensure interfaces are implemented.
var _ MemoryWriter = (*state.Database)(nil)
var _ IngestCandidateReader = (*state.Database)(nil)
