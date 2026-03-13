package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/replication"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// TurnPersister persists a turn to storage (e.g. Database.InsertTurn).
type TurnPersister interface {
	InsertTurn(id, timestamp, state, input, inputSource, thinking, toolCalls, tokenUsage string, costCents int) error
}

// AgentStore provides full state for the ReAct loop (TS-aligned).
type AgentStore interface {
	TurnPersister
	InsertToolCall(turnID, id, name, args, result string, durationMs int, errStr string) error
	GetRecentTurns(limit int) ([]state.Turn, error)
	GetTurnCount() (int64, error)
	GetAgentState() (string, bool, error)
}

// ToolExecutor runs tools by name. When nil, tool calls are stubbed.
type ToolExecutor interface {
	Execute(ctx context.Context, name string, args map[string]any) (result string, err error)
}

// Loop implements the ReAct cycle per mormoneyOS design.
type Loop struct {
	policy        *PolicyEngine
	persister     TurnPersister
	store         AgentStore
	inference     inference.Client
	tools         ToolExecutor
	config        *LoopConfig
	creditsFn     func(ctx context.Context) int64
	lineageStore  replication.LineageStore
	log           *slog.Logger
}

// NewLoop creates an agent loop.
func NewLoop(policy *PolicyEngine, log *slog.Logger) *Loop {
	return NewLoopWithPersister(policy, nil, log)
}

// NewLoopWithPersister creates an agent loop that persists turns.
func NewLoopWithPersister(policy *PolicyEngine, persister TurnPersister, log *slog.Logger) *Loop {
	if log == nil {
		log = slog.Default()
	}
	return &Loop{policy: policy, persister: persister, log: log}
}

// LoopOptions configures the full ReAct loop (TS AgentLoopOptions-aligned).
type LoopOptions struct {
	Policy       *PolicyEngine
	Store        AgentStore
	Inference    inference.Client
	Tools        ToolExecutor
	Config       *LoopConfig
	CreditsFn    func(ctx context.Context) int64
	LineageStore replication.LineageStore // optional; for GetLineageSummary in system prompt
	Log          *slog.Logger
}

// NewLoopWithOptions creates a loop with full ReAct support.
func NewLoopWithOptions(opts LoopOptions) *Loop {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	return &Loop{
		policy:       opts.Policy,
		persister:    opts.Store,
		store:        opts.Store,
		inference:    opts.Inference,
		tools:        opts.Tools,
		config:       opts.Config,
		creditsFn:    opts.CreditsFn,
		lineageStore: opts.LineageStore,
		log:          opts.Log,
	}
}

// RunOneTurn executes one ReAct iteration.
// Design: think -> act -> observe -> persist (TS-aligned).
// When inference+store are set: build prompt, call inference, parse tool calls, evaluate via policy, persist.
func (l *Loop) RunOneTurn(ctx context.Context, agentState types.AgentState) (types.AgentState, error) {
	l.log.Debug("agent loop turn", "state", agentState)

	st := string(agentState)
	if st == "" {
		st = string(types.AgentStateRunning)
	}

	// Full ReAct path when inference and store are configured
	if l.inference != nil && l.store != nil {
		return l.runOneTurnReAct(ctx, st)
	}

	// Stub path: persist turn only
	if l.persister != nil {
		id := uuid.New().String()
		ts := time.Now().UTC().Format(time.RFC3339)
		thinking := "[stub] No inference client; turn persisted for count."
		if err := l.persister.InsertTurn(id, ts, st, "", "", thinking, "[]", "{}", 0); err != nil {
			l.log.Warn("insert turn failed", "err", err)
		}
	}
	return types.AgentStateRunning, nil
}

func (l *Loop) runOneTurnReAct(ctx context.Context, stateStr string) (types.AgentState, error) {
	turnCount, _ := l.store.GetTurnCount()
	agentState, _, _ := l.store.GetAgentState()
	if agentState == "" {
		agentState = "running"
	}
	creditsCents := int64(0)
	if l.creditsFn != nil {
		creditsCents = l.creditsFn(ctx)
	}

	// Build wakeup/input
	recentTurns, _ := l.store.GetRecentTurns(5)
	lastSummaries := make([]string, 0, 3)
	for i := len(recentTurns) - 1; i >= 0 && len(lastSummaries) < 3; i-- {
		t := recentTurns[i]
		src := t.InputSource
		if src == "" {
			src = "self"
		}
		think := t.Thinking
		if len(think) > 200 {
			think = think[:200] + "..."
		}
		lastSummaries = append(lastSummaries, "["+t.Timestamp+"] "+src+": "+think)
	}
	pendingInput := BuildWakeupPrompt(l.config, turnCount, creditsCents, lastSummaries)
	if turnCount > 0 {
		pendingInput = "You are awake. What do you want to do next?"
	}

	// Build context
	lineageSummary := ""
	if l.lineageStore != nil {
		lineageSummary = replication.GetLineageSummary(l.lineageStore)
	}
	systemPrompt := BuildSystemPrompt(l.config, agentState, turnCount, creditsCents, lineageSummary)
	messages := BuildContextMessages(systemPrompt, recentTurns, pendingInput)

	// Inference options with tool definitions (from registry when available)
	toolDefs := getToolSchemas(l.tools)
	opts := &inference.InferenceOptions{
		Model:       l.inference.GetDefaultModel(),
		MaxTokens:   4096,
		Tools:       toolDefs,
		ToolChoice:  "auto",
	}

	// Call inference
	resp, err := l.inference.Chat(ctx, messages, opts)
	if err != nil {
		l.log.Warn("inference failed", "err", err)
		// Persist error turn
		id := uuid.New().String()
		ts := time.Now().UTC().Format(time.RFC3339)
		_ = l.store.InsertTurn(id, ts, stateStr, pendingInput, "wakeup", "[inference error: "+err.Error()+"]", "[]", "{}", 0)
		return types.AgentStateRunning, nil
	}

	turnID := uuid.New().String()
	ts := time.Now().UTC().Format(time.RFC3339)
	thinking := resp.Content
	toolCallsJSON := "[]"
	tokenUsage := jsonObject("prompt_tokens", resp.InputTokens, "completion_tokens", resp.OutputTokens)

	if len(resp.ToolCalls) > 0 {
		tcList := make([]map[string]any, 0, len(resp.ToolCalls))
		for _, tc := range resp.ToolCalls {
			args := tc.Function.Arguments
			var argsMap map[string]any
			_ = json.Unmarshal([]byte(args), &argsMap)
			if argsMap == nil {
				argsMap = make(map[string]any)
			}
			tool := types.ToolCall{ID: tc.ID, Name: tc.Function.Name, Args: argsMap}
			allow, reason := l.policy.Evaluate(tool, "self", types.RiskSafe)
			// Persist policy decision for audit and rate-limit tracking (TS-aligned)
			if pdStore, ok := l.store.(PolicyDecisionStore); ok {
				dec := "allow"
				if !allow {
					dec = "deny"
				}
				_ = pdStore.InsertPolicyDecision(
					"pd-"+tc.ID, turnID, tc.Function.Name, ToolArgsHash(argsMap),
					string(types.RiskSafe), dec, reason, "self",
				)
			}
			result := ""
			errStr := ""
			if !allow {
				errStr = "policy denied: " + reason
			} else if l.tools != nil {
				var execErr error
				result, execErr = l.tools.Execute(ctx, tc.Function.Name, argsMap)
				if execErr != nil {
					errStr = execErr.Error()
				}
			} else {
				result = "[stub] Tool not implemented; policy allowed."
			}
			tcList = append(tcList, map[string]any{
				"id": tc.ID, "name": tc.Function.Name, "arguments": argsMap,
				"result": result, "error": errStr,
			})
			_ = l.store.InsertToolCall(turnID, tc.ID, tc.Function.Name, args, result, 0, errStr)
		}
		tcBytes, _ := json.Marshal(tcList)
		toolCallsJSON = string(tcBytes)
	}

	if err := l.store.InsertTurn(turnID, ts, stateStr, pendingInput, "wakeup", thinking, toolCallsJSON, tokenUsage, resp.CostCents); err != nil {
		l.log.Warn("insert turn failed", "err", err)
	}

	return types.AgentStateRunning, nil
}

// toolSchemasProvider is satisfied by tools.Registry for extensible tool definitions.
type toolSchemasProvider interface {
	Schemas() []inference.ToolDefinition
}

func getToolSchemas(exec ToolExecutor) []inference.ToolDefinition {
	if sp, ok := exec.(toolSchemasProvider); ok {
		return sp.Schemas()
	}
	return tools.BuiltinToolSchemas()
}

func jsonObject(kv ...interface{}) string {
	m := make(map[string]interface{})
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok {
			m[k] = kv[i+1]
		}
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// ShouldSleep returns true if agent should sleep (idle or explicit sleep).
func (l *Loop) ShouldSleep(idleTurns int) bool {
	return idleTurns >= 3
}
