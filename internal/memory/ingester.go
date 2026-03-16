package memory

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// IngestCandidateStore persists raw extractions for consolidation.
type IngestCandidateStore interface {
	InsertIngestCandidate(sessionID, turnID, extractionJSON string, importance float64) error
}

const extractionPrompt = `Extract structured knowledge from this agent turn. Return ONLY valid JSON, no markdown or explanation.

Format:
{"facts":[{"category":"","key":"","value":"","confidence":0.9}],"episodes":[{"event_type":"event","summary":"","detail":"","outcome":"","importance":0.5}],"procedures":[{"name":"","steps":[],"success_rate":0.9}],"relationships":[{"entity_address":"","entity_name":"","type":"","trust_delta":0}],"importance":0.5}

Rules: max 3 facts, 1 procedure, 1 episode, relationship changes only if entities interacted. importance 0-1. Empty arrays ok.`

// TurnData holds the turn content for ingestion (passed from agent loop).
type TurnData struct {
	TurnID      string
	Timestamp   string
	SessionID   string
	Input       string
	InputSource string
	Thinking    string
	ToolCalls   string // JSON array of {name, result, error}
}

// IngestMetricsRecorder records ingestion metrics (optional, avoids import cycles).
type IngestMetricsRecorder interface {
	RecordIngestTurn()
	RecordLatencyMs(ms int64)
}

// Ingester extracts structured knowledge from turns and stores as candidates.
type Ingester struct {
	client  inference.Client
	model   string
	store   IngestCandidateStore
	metrics IngestMetricsRecorder
	log     *slog.Logger
}

// NewIngester creates an ingester with the given inference client and store.
func NewIngester(client inference.Client, model string, store IngestCandidateStore, log *slog.Logger) *Ingester {
	if log == nil {
		log = slog.Default()
	}
	if model == "" {
		model = "gpt-4o-mini" // fallback cheap model
	}
	return &Ingester{client: client, model: model, store: store, log: log}
}

// SetMetrics sets the optional metrics recorder.
func (i *Ingester) SetMetrics(m IngestMetricsRecorder) {
	i.metrics = m
}

// Ingest extracts from the turn and stores as a candidate. Non-blocking; errors are logged.
func (i *Ingester) Ingest(ctx context.Context, turn *TurnData) error {
	if turn == nil || (turn.Thinking == "" && turn.ToolCalls == "") {
		return nil
	}
	start := time.Now()
	content := buildTurnSummary(turn)
	if content == "" {
		return nil
	}
	msgs := []inference.ChatMessage{
		{Role: "system", Content: extractionPrompt},
		{Role: "user", Content: content},
	}
	opts := &inference.InferenceOptions{
		Model:       i.model,
		MaxTokens:   512,
		Temperature: 0.1,
	}
	resp, err := i.client.Chat(ctx, msgs, opts)
	if err != nil {
		i.log.Warn("ingestion inference failed", "err", err)
		return err
	}
	extraction, importance := parseExtraction(resp.Content)
	if extraction == nil {
		return nil
	}
	extJSON, _ := json.Marshal(extraction)
	sessionID := turn.SessionID
	if sessionID == "" {
		sessionID = "default"
	}
	if err := i.store.InsertIngestCandidate(sessionID, turn.TurnID, string(extJSON), importance); err != nil {
		i.log.Warn("insert ingest candidate failed", "err", err)
		return err
	}
	latencyMs := time.Since(start).Milliseconds()
	if i.metrics != nil {
		i.metrics.RecordIngestTurn()
		i.metrics.RecordLatencyMs(latencyMs)
	}
	i.log.Debug("ingestion complete", "turn_id", turn.TurnID, "latency_ms", latencyMs)
	return nil
}

func buildTurnSummary(t *TurnData) string {
	var b strings.Builder
	if t.Input != "" {
		b.WriteString("Input: ")
		b.WriteString(truncate(t.Input, 300))
		b.WriteString("\n")
	}
	if t.Thinking != "" {
		b.WriteString("Thought: ")
		b.WriteString(truncate(t.Thinking, 500))
		b.WriteString("\n")
	}
	if t.ToolCalls != "" {
		b.WriteString("Tool results: ")
		b.WriteString(truncate(t.ToolCalls, 400))
	}
	return strings.TrimSpace(b.String())
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func parseExtraction(content string) (*Extraction, float64) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	var raw struct {
		Facts         []Fact               `json:"facts"`
		Episodes      []Episode            `json:"episodes"`
		Procedures    []Procedure          `json:"procedures"`
		Relationships []RelationshipUpdate `json:"relationships"`
		Importance    float64              `json:"importance"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, 0
	}
	imp := raw.Importance
	if imp < 0 || imp > 1 {
		imp = 0.5
	}
	return &Extraction{
		Facts:         raw.Facts,
		Episodes:      raw.Episodes,
		Procedures:    raw.Procedures,
		Relationships: raw.Relationships,
		Importance:    imp,
	}, imp
}

// Ensure IngestCandidateStore is implemented by *state.Database.
var _ IngestCandidateStore = (*state.Database)(nil)
