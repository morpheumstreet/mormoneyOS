package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/memory"
	"github.com/morpheumlabs/mormoneyos-go/internal/prompts"
	"github.com/morpheumlabs/mormoneyos-go/internal/replication"
	"github.com/morpheumlabs/mormoneyos-go/internal/skills"
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

// InboxStore provides inbox claim/process for pendingInput (TS step 2: claim inbox messages).
// Optional: when store implements this, agent uses claimed messages as pendingInput.
type InboxStore interface {
	ClaimInboxMessages(limit int) ([]state.InboxMessage, error)
	MarkInboxProcessed(ids []string) error
}

// TierStateStore provides SetAgentState for survival tier transitions (TS step 4).
// Optional: when store implements this, agent updates agent_state from tier.
type TierStateStore interface {
	SetAgentState(state string) error
}

// ToolExecutor runs tools by name. When nil, tool calls are stubbed.
type ToolExecutor interface {
	Execute(ctx context.Context, name string, args map[string]any) (result string, err error)
}

// FallbackSender sends a fallback message when LLM fails to process claimed inbox messages.
// Called with ctx and claimed message IDs; implementation looks up route and sends via channels.
type FallbackSender func(ctx context.Context, claimedIds []string)

// MemoryIngester ingests turn data into memory (optional, for automatic memory pipeline).
type MemoryIngester interface {
	IngestTurn(ctx context.Context, turn *memory.TurnData) error
}

// Loop implements the ReAct cycle per mormoneyOS design.
type Loop struct {
	policy              *PolicyEngine
	persister           TurnPersister
	store               AgentStore
	inference           inference.Client
	tools               ToolExecutor
	config              *LoopConfig
	creditsFn           func(ctx context.Context) int64
	lineageStore        replication.LineageStore
	memoryRetriever     memory.MemoryRetriever
	memoryIngester      MemoryIngester
	modelRouter         *inference.ModelRouter
	reflectionEngine    *ReflectionEngine
	disabledToolsGetter func() []string
	fallbackSender      FallbackSender
	log                 *slog.Logger
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
	Policy              *PolicyEngine
	Store               AgentStore
	Inference           inference.Client
	Tools               ToolExecutor
	Config              *LoopConfig
	CreditsFn           func(ctx context.Context) int64
	LineageStore        replication.LineageStore   // optional; for GetLineageSummary in system prompt
	MemoryRetriever     memory.MemoryRetriever     // optional; TS step 6 pre-turn memory injection
	MemoryIngester      MemoryIngester             // optional; automatic memory ingestion from turns
	ModelRouter         *inference.ModelRouter    // optional; when set, routes to fast/normal/strong model per turn
	ReflectionEngine    *ReflectionEngine          // optional; when set, runs self-critique on impactful turns
	DisabledToolsGetter func() []string            // optional; when set, filters tool schemas (tools disabled via dashboard)
	FallbackSender      FallbackSender            // optional; when LLM fails with claimed inbox, send "Sorry, having trouble" to user
	Log                 *slog.Logger
}

// NewLoopWithOptions creates a loop with full ReAct support.
func NewLoopWithOptions(opts LoopOptions) *Loop {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	return &Loop{
		policy:              opts.Policy,
		persister:           opts.Store,
		store:               opts.Store,
		inference:           opts.Inference,
		tools:               opts.Tools,
		config:              opts.Config,
		creditsFn:           opts.CreditsFn,
		lineageStore:        opts.LineageStore,
		memoryRetriever:     opts.MemoryRetriever,
		memoryIngester:      opts.MemoryIngester,
		modelRouter:        opts.ModelRouter,
		reflectionEngine:   opts.ReflectionEngine,
		disabledToolsGetter: opts.DisabledToolsGetter,
		fallbackSender:      opts.FallbackSender,
		log:                 opts.Log,
	}
}

// RunOneTurn executes one ReAct iteration.
// Design: think -> act -> observe -> persist (TS-aligned).
// Returns TurnResult with State (Running/Sleeping), WasIdle, and Err (step 13 aligned).
func (l *Loop) RunOneTurn(ctx context.Context, agentState types.AgentState) TurnResult {
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
	return TurnResult{State: types.AgentStateRunning, WasIdle: false}
}

func (l *Loop) runOneTurnReAct(ctx context.Context, stateStr string) TurnResult {
	turnCount, _ := l.store.GetTurnCount()
	agentState, _, _ := l.store.GetAgentState()
	if agentState == "" {
		agentState = "running"
	}
	creditsCents := int64(0)
	if l.creditsFn != nil {
		creditsCents = l.creditsFn(ctx)
	}
	// API unreachable: creditsCents < 0 means API failed; treat as 0 for tier (TS-aligned)
	if creditsCents < 0 {
		creditsCents = 0
	}

	// Step 4: survival tier — set agent_state, low-compute mode, model selection
	tier := conway.TierFromCreditsCents(creditsCents)
	agentState = tierToAgentState(tier)
	useLowCompute := tier == types.SurvivalTierCritical || tier == types.SurvivalTierLowCompute
	if l.config != nil && l.config.ResourceConstraintMode != "" {
		switch l.config.ResourceConstraintMode {
		case "forced_on":
			useLowCompute = true
			agentState = string(types.AgentStateLowCompute)
		case "forced_off":
			useLowCompute = false
		}
	}
	if tierStore, ok := l.store.(TierStateStore); ok {
		_ = tierStore.SetAgentState(agentState)
	}
	l.inference.SetLowComputeMode(useLowCompute)

	// Build wakeup/input (TS step 2: claim inbox messages when no pendingInput)
	recentTurnsForWakeup, _ := l.store.GetRecentTurns(5)
	lastSummaries := make([]string, 0, 3)
	for i := len(recentTurnsForWakeup) - 1; i >= 0 && len(lastSummaries) < 3; i-- {
		t := recentTurnsForWakeup[i]
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

	var pendingInput string
	var inputSource string
	var claimedIds []string

	if inboxStore, ok := l.store.(InboxStore); ok {
		claimed, err := inboxStore.ClaimInboxMessages(10)
		if err != nil {
			l.log.Warn("claim inbox failed", "err", err)
		}
		if len(claimed) > 0 {
			parts := make([]string, 0, len(claimed))
			for _, m := range claimed {
				parts = append(parts, "[Message from "+m.FromAddress+"]: "+m.Content)
				claimedIds = append(claimedIds, m.ID)
			}
			pendingInput = strings.Join(parts, "\n\n")
			inputSource = "agent"
		}
	}
	if pendingInput == "" {
		pendingInput = BuildWakeupPrompt(l.config, turnCount, creditsCents, lastSummaries)
		if turnCount > 0 {
			pendingInput = "You are awake. What do you want to do next?"
		}
		inputSource = "wakeup"
	}

	// Build context
	lineageSummary := ""
	if l.lineageStore != nil {
		lineageSummary = replication.GetLineageSummary(l.lineageStore)
	}
	skillList := []*skills.Skill{}
	if l.config != nil && l.config.SkillsConfig != nil {
		if store, ok := l.store.(skills.SkillRowStore); ok {
			skillList = skills.LoadAllFromStore(store, l.config.SkillsConfig)
		}
	}

	// Fetch more turns for truncation (start generous; BuildMessagesSafe will cap)
	historyLimit := 50
	if l.config != nil && l.config.TokenLimits != nil && l.config.TokenLimits.MaxHistoryTurns > 0 {
		if l.config.TokenLimits.MaxHistoryTurns > historyLimit {
			historyLimit = l.config.TokenLimits.MaxHistoryTurns
		}
	}
	recentTurnsForContext, _ := l.store.GetRecentTurns(historyLimit)

	// Inference options with tool definitions (from registry when available)
	toolDefs := getToolSchemas(l.tools)
	if l.disabledToolsGetter != nil {
		disabled := make(map[string]bool)
		for _, n := range l.disabledToolsGetter() {
			disabled[n] = true
		}
		filtered := make([]inference.ToolDefinition, 0, len(toolDefs))
		for _, t := range toolDefs {
			if !disabled[t.Function.Name] {
				filtered = append(filtered, t)
			}
		}
		toolDefs = filtered
	}

	// Build messages with token cap and truncation (avoids prefill limit errors)
	// Step 6: MessageTrimmer orchestrates memory retrieval (with budget when supported) + BuildMessagesSafe
	limits := DefaultTokenLimits()
	if l.config != nil && l.config.TokenLimits != nil {
		limits = *l.config.TokenLimits
	}
	model := l.inference.GetDefaultModel()
	if useLowCompute && l.config != nil && l.config.LowComputeModel != "" {
		model = l.config.LowComputeModel
	}
	effectiveCap := 0
	if l.config != nil && l.config.ContextLimitForModel != nil {
		effectiveCap = l.config.ContextLimitForModel(model)
	}
	var messages []inference.ChatMessage
	if l.config != nil && l.config.PromptVersion != "" {
		// Versioned prompts: template-based system + CoT forcing
		credits := float64(creditsCents) / 100
		if creditsCents < 0 {
			credits = 0
		}
		systemData := prompts.SystemPromptData{
			State:          agentState,
			Credits:        fmt.Sprintf("%.2f", credits),
			Tier:           string(tier),
			TurnCount:      turnCount,
			Model:          model,
			LineageSummary: lineageSummary,
			SkillsBlock:    skills.FormatForPrompt(skillList, tokenBudgetFromConfig(l.config)),
			GenesisPrompt:  truncateGenesis(l.config.GenesisPrompt, 2000),
		}
		messages, _ = BuildMessagesFromPrompts(ctx, prompts.Version(l.config.PromptVersion), systemData, recentTurnsForContext, pendingInput, l.memoryRetriever, toolDefs, limits, effectiveCap, DefaultTokenizer, l.log)
	} else {
		// Legacy: ad-hoc prompt building
		systemPrompt := BuildSystemPrompt(l.config, agentState, turnCount, creditsCents, string(tier), lineageSummary, skillList)
		if l.memoryRetriever != nil {
			trimmer := NewMessageTrimmer(DefaultTokenizer)
			messages, _ = trimmer.Trim(ctx, systemPrompt, recentTurnsForContext, pendingInput, l.memoryRetriever, toolDefs, limits, effectiveCap, l.log)
		} else {
			messages = BuildMessagesSafe(systemPrompt, recentTurnsForContext, pendingInput, "", toolDefs, limits, effectiveCap, DefaultTokenizer, l.log)
		}
	}
	// Route model when router is configured
	client := l.inference
	if l.modelRouter != nil {
		tokensUsed := CountMessagesTokens(messages, toolDefs, DefaultTokenizer)
		hasMoneyImpact := strings.Contains(strings.ToLower(pendingInput), "transfer") ||
			strings.Contains(strings.ToLower(pendingInput), "fund") ||
			strings.Contains(strings.ToLower(pendingInput), "send")
		dc := inference.DecisionContext{
			TokensUsed:     tokensUsed,
			RiskLevel:      inference.RiskLow,
			HasMoneyImpact: hasMoneyImpact,
			TurnPhase:      "action",
		}
		if routed, err := l.modelRouter.Select(ctx, dc); err == nil && routed != nil {
			client = routed
			model = client.GetDefaultModel()
		}
	}

	opts := &inference.InferenceOptions{
		Model:      model,
		MaxTokens:  4096,
		Tools:      toolDefs,
		ToolChoice: "auto",
	}

	// Call inference
	l.log.Debug("inference call", "model", model, "messages", len(messages))
	resp, err := client.Chat(ctx, messages, opts)
	if err != nil {
		l.log.Warn("inference failed", "err", err)
		// Persist error turn
		id := uuid.New().String()
		ts := time.Now().UTC().Format(time.RFC3339)
		_ = l.store.InsertTurn(id, ts, stateStr, pendingInput, inputSource, "[inference error: "+err.Error()+"]", "[]", "{}", 0)
		// Send fallback to user when LLM fails with claimed inbox (e.g. Telegram never gets reply)
		if len(claimedIds) > 0 && l.fallbackSender != nil {
			l.fallbackSender(ctx, claimedIds)
		}
		return TurnResult{State: types.AgentStateRunning, Err: err}
	}

	// finishReason stop + no tool calls: natural pause, sleep (TS step 13)
	// Do NOT sleep when we have claimed inbox messages that weren't processed — stay running to handle them.
	if len(resp.ToolCalls) == 0 && resp.FinishReason == "stop" {
		turnID := uuid.New().String()
		ts := time.Now().UTC().Format(time.RFC3339)
		tokenUsage := jsonObject("prompt_tokens", resp.InputTokens, "completion_tokens", resp.OutputTokens)
		_ = l.store.InsertTurn(turnID, ts, stateStr, pendingInput, inputSource, resp.Content, "[]", tokenUsage, resp.CostCents)
		if l.memoryIngester != nil {
			_ = l.memoryIngester.IngestTurn(ctx, &memory.TurnData{
				TurnID: turnID, Timestamp: ts, SessionID: "", Input: pendingInput, InputSource: inputSource,
				Thinking: resp.Content, ToolCalls: "[]",
			})
		}
		if len(claimedIds) > 0 {
			// Inbox messages claimed but model stopped without acting; stay running to process them next turn
			return TurnResult{State: types.AgentStateRunning, WasIdle: false}
		}
		return TurnResult{State: types.AgentStateSleeping, WasIdle: false}
	}

	turnID := uuid.New().String()
	ts := time.Now().UTC().Format(time.RFC3339)
	thinking := resp.Content
	toolCallsJSON := "[]"
	tokenUsage := jsonObject("prompt_tokens", resp.InputTokens, "completion_tokens", resp.OutputTokens)

	var sleepToolSucceeded bool
	var anyMutatingToolExecuted bool

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
			// Step 13: track sleep tool success and mutating tools
			if tc.Function.Name == "sleep" && errStr == "" {
				sleepToolSucceeded = true
			}
			if tools.IsMutatingTool(tc.Function.Name) && allow {
				anyMutatingToolExecuted = true
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

	if err := l.store.InsertTurn(turnID, ts, stateStr, pendingInput, inputSource, thinking, toolCallsJSON, tokenUsage, resp.CostCents); err != nil {
		l.log.Warn("insert turn failed", "err", err)
	}
	if l.memoryIngester != nil {
		_ = l.memoryIngester.IngestTurn(ctx, &memory.TurnData{
			TurnID: turnID, Timestamp: ts, SessionID: "", Input: pendingInput, InputSource: inputSource,
			Thinking: thinking, ToolCalls: toolCallsJSON,
		})
	}

	// Self-critique on impactful turns (Reflexion-style improvement)
	if l.reflectionEngine != nil && anyMutatingToolExecuted {
		ref, err := l.reflectionEngine.CritiqueTurn(ctx, &CritiqueTurnData{
			TurnID: turnID, Input: pendingInput, Thinking: thinking, ToolCalls: toolCallsJSON,
		})
		if err == nil && ref != nil {
			rd := &memory.ReflectionData{
				TurnID:                turnID,
				SuccessScore:          ref.SuccessScore,
				Lessons:               ref.Lessons,
				MemoryRecommendations: ref.MemoryRecommendations,
			}
			if ri, ok := l.memoryIngester.(ReflectionIngester); ok {
				_ = ri.IngestReflection(ctx, rd)
			}
		}
	}

	// TS step 2: mark claimed inbox messages as processed (atomic with turn persistence)
	if len(claimedIds) > 0 {
		if inboxStore, ok := l.store.(InboxStore); ok {
			_ = inboxStore.MarkInboxProcessed(claimedIds)
		}
	}

	// Step 13: sleep tool immediate transition
	if sleepToolSucceeded {
		return TurnResult{State: types.AgentStateSleeping, WasIdle: false}
	}
	wasIdle := turnCount > 0 && !anyMutatingToolExecuted
	return TurnResult{State: types.AgentStateRunning, WasIdle: wasIdle}
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

func tokenBudgetFromConfig(cfg *LoopConfig) int {
	if cfg == nil || cfg.SkillsConfig == nil {
		return 2000
	}
	return cfg.SkillsConfig.TokenBudgetMax
}

func truncateGenesis(s string, maxLen int) string {
	if s == "" || len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// tierToAgentState maps SurvivalTier to agent state string for prompt/observability.
func tierToAgentState(tier types.SurvivalTier) string {
	switch tier {
	case types.SurvivalTierCritical:
		return string(types.AgentStateCritical)
	case types.SurvivalTierLowCompute:
		return string(types.AgentStateLowCompute)
	case types.SurvivalTierDead:
		return string(types.AgentStateDead)
	case types.SurvivalTierNormal, types.SurvivalTierHigh:
		return string(types.AgentStateRunning)
	default:
		return string(types.AgentStateRunning)
	}
}
