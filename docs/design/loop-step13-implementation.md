# Step 13 Implementation: Loop / Idle / Sleep (TS-Aligned)

**Date:** 2026-03-13  
**Purpose:** Align Go agent loop with TS step 13: sleep tool immediate transition, idle-only turn counting, finishReason stop handling.

---

## 1. Architecture Notes

- Single responsibility; TurnResult encapsulates turn outcome; main loop stays simple.
- Mutating tools list defined once in `internal/tools`; idle logic in one place.
- Explicit return types; no hidden state; testable.

---

## 2. Components

### 2.1 TurnResult

Replace `(AgentState, error)` with a struct that carries all turn outcome data:

```go
// TurnResult holds the outcome of one agent turn (TS step 13 aligned).
type TurnResult struct {
    State   types.AgentState  // Running or Sleeping
    WasIdle bool             // true if no mutating tools and turnCount > 0
    Err     error
}
```

- **State = Sleeping** when: sleep tool ran successfully, OR (no tool calls && finishReason == "stop")
- **State = Running** otherwise
- **WasIdle = true** when: turnCount > 0 && no mutating tools executed
- **Err** for inference/DB errors

### 2.2 Mutating Tools (Single Source of Truth)

Location: `internal/tools/mutating.go`

```go
// IsMutatingTool returns true if the tool performs side effects that count as "real work".
// Used for idle-turn detection (TS MUTATING_TOOLS-aligned).
func IsMutatingTool(name string) bool
```

Tool names that count as mutating (TS-aligned, Go tool names): shell, exec, write_file, edit_own_file, transfer_credits, fund_child, spawn_child, start_child, delete_sandbox, create_sandbox, install_npm_package, install_skill, create_skill, remove_skill, pull_upstream, git_commit, git_push, git_branch, git_clone, send_message, message_child, update_genesis_prompt, modify_heartbeat, expose_port, remove_port, distress_signal, prune_dead_children, sleep, update_soul, remember_fact, set_goal, complete_goal, save_procedure, note_about_agent, forget, enter_low_compute, switch_model, review_upstream_changes.

### 2.3 RunOneTurn Changes

```go
func (l *Loop) RunOneTurn(ctx context.Context, agentState types.AgentState) TurnResult
```

In `runOneTurnReAct`:

1. **Sleep tool immediate transition:** After tool execution loop, if any tool was "sleep" and succeeded (no errStr), return `TurnResult{State: Sleeping, WasIdle: false}`.
2. **finishReason stop:** Before tool loop, if `len(resp.ToolCalls) == 0` and `resp.FinishReason == "stop"`, return `TurnResult{State: Sleeping, WasIdle: false}`.
3. **WasIdle:** After tool loop, `WasIdle = (turnCount > 0) && !anyMutatingToolExecuted`. Track executed tool names in the loop.

### 2.4 Main Loop (cmd/run.go)

```go
res := loop.RunOneTurn(ctx, agentState)
if res.Err != nil {
    slog.Error("agent turn failed", "err", res.Err)
    continue
}
tickNum++
if res.State == types.AgentStateSleeping {
    agentState = types.AgentStateSleeping
    idleTurns = 0
    slog.Info("agent sleeping")
} else {
    agentState = res.State
    if res.WasIdle {
        idleTurns++
    } else {
        idleTurns = 0
    }
    if agentState == types.AgentStateRunning && loop.ShouldSleep(idleTurns) {
        agentState = types.AgentStateSleeping
        idleTurns = 0
        slog.Info("agent sleeping")
    }
}
```

---

## 3. Constants

| Constant | Value | Notes |
|----------|-------|-------|
| IdleSleepThreshold | 3 | Same as ShouldSleep; sleep after 3 idle turns |
| SleepToolName | "sleep" | Tool name for sleep |
| FinishReasonStop | "stop" | API finish_reason when model stops without tool calls |

---

## 4. File Changes

| File | Change |
|------|--------|
| `internal/tools/mutating.go` | New: IsMutatingTool; mutating tool names set |
| `internal/agent/turn_result.go` | New: TurnResult struct |
| `internal/agent/loop.go` | RunOneTurn returns TurnResult; runOneTurnReAct implements sleep/finishReason/WasIdle |
| `cmd/run.go` | Use TurnResult; update idle logic |
| `docs/design/ts-go-alignment.md` | Step 13 → aligned |

---

## 5. Edge Cases

- **Stub path:** RunOneTurn (no inference) returns `TurnResult{State: Running, WasIdle: false}`.
- **Inference error:** Return `TurnResult{State: Running, Err: err}`; main loop continues.
- **Sleep tool denied by policy:** errStr set; no immediate transition; turn counted as non-idle (policy denied = mutation attempted).
- **Tool alias:** "exec" resolves to "shell"; both are mutating; IsMutatingTool checks both names.
