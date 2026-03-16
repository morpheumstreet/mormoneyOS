# mormoneyOS — Test Report

**mormoneyOS-go — 17 March 2026**

## Summary

mormoneyOS unit and integration tests. **All tests passed, 0 failed.** Tests cover config, types, Conway credits, policy engine, state/database, heartbeat, agent loop (HistoryTrimmer, MessageTrimmer, history compression), tools, inference, identity, memory (TieredMemorySelector, TieredMemoryRetriever), skills, soul, tunnel, and CLI commands. Includes race-detector verification and acceptance criteria.

---

## 1. Test Results

### 1.1 Config (`internal/config`)

**File:** `internal/config/config_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| C1 | `TestGetAutomatonDir_Default` | config | PASS |
| C2 | `TestGetAutomatonDir_Override` | config | PASS |
| C3 | `TestGetConfigPath` | config | PASS |
| C4 | `TestResolvePath_WithTilde` | config | PASS |
| C5 | `TestResolvePath_WithoutTilde` | config | PASS |
| C6 | `TestLoad_NoFile` | config | PASS |
| C7 | `TestLoad_InvalidJSON` | config | PASS |
| C8 | `TestLoad_MergesWithDefaults` | config | PASS |
| C9 | `TestLoad_TreasuryMerge` | config | PASS |
| C10 | `TestSave_CreatesDir` | config | PASS |
| C11 | `TestSave_RoundTrip` | config | PASS |
| C12 | `TestLoadToolsFromFile_JSON` | config | PASS |
| C13 | `TestLoadToolsFromFile_YAML` | config | PASS |
| C14 | `TestLoad_WithTools` | config | PASS |

**Total: 14 passed, 0 failed**

---

### 1.2 Types (`internal/types`)

**File:** `internal/types/types_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| T1 | `TestDefaultTreasuryPolicy` | types | PASS |
| T2 | `TestAgentStateConstants` | types | PASS |
| T3 | `TestSurvivalTierConstants` | types | PASS |
| T4 | `TestRiskLevelConstants` | types | PASS |

**Total: 4 passed, 0 failed**

---

### 1.3 Conway Credits (`internal/conway`)

**File:** `internal/conway/credits_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| CR1 | `TestTierFromCreditsCents_High` | conway-credits | PASS |
| CR2 | `TestTierFromCreditsCents_Normal` | conway-credits | PASS |
| CR3 | `TestTierFromCreditsCents_LowCompute` | conway-credits | PASS |
| CR4 | `TestTierFromCreditsCents_Critical` | conway-credits | PASS |
| CR5 | `TestTierFromCreditsCents_Dead` | conway-credits | PASS |
| CR6 | `TestTierFromCreditsCents_BoundaryHighNormal` | conway-credits | PASS |
| CR7 | `TestTierFromCreditsCents_BoundaryNormalLow` | conway-credits | PASS |
| CR8 | `TestTierFromCreditsCents_BoundaryLowCritical` | conway-credits | PASS |

**Total: 8 passed, 0 failed**

---

### 1.4 Policy Engine (`internal/agent`)

**Files:** `internal/agent/policy_test.go`, `internal/agent/policy_rules_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| P1 | `TestToolArgsHash_Deterministic` | policy-engine | PASS |
| P2 | `TestToolArgsHash_DifferentArgs` | policy-engine | PASS |
| P3 | `TestValidationRule_EmptyName` | policy-engine | PASS |
| P4 | `TestValidationRule_WhitespaceName` | policy-engine | PASS |
| P5 | `TestValidationRule_ValidName` | policy-engine | PASS |
| P6 | `TestPathProtectionRule_ProtectedWrite` | policy-engine | PASS |
| P7 | `TestPathProtectionRule_ProtectedRead` | policy-engine | PASS |
| P8 | `TestPathProtectionRule_SafePath` | policy-engine | PASS |
| P9 | `TestPathProtectionRule_NoPathArg` | policy-engine | PASS |
| P10 | `TestPathProtectionRule_FilePathArg` | policy-engine | PASS |
| P11 | `TestAuthorityRule_Creator` | policy-engine | PASS |
| P12 | `TestAuthorityRule_Self` | policy-engine | PASS |
| P13 | `TestAuthorityRule_ExternalDangerous` | policy-engine | PASS |
| P14 | `TestAuthorityRule_ExternalSafe` | policy-engine | PASS |
| P15 | `TestPolicyEngine_Evaluate_FirstDenyWins` | policy-engine | PASS |
| P16 | `TestPolicyEngine_Evaluate_AllAllow` | policy-engine | PASS |
| P17 | `TestCreateDefaultRules` | policy-engine | PASS |
| P18 | `TestFinancialRule_OverLimit` | policy-engine | PASS |
| P19 | `TestFinancialRule_UnderLimit` | policy-engine | PASS |
| P20 | `TestCommandSafetyRule_Dangerous` | policy-engine | PASS |
| P21 | `TestCommandSafetyRule_Safe` | policy-engine | PASS |

**Total: 21 passed, 0 failed**

---

### 1.5 State / Database (`internal/state`)

**Files:** `internal/state/database_test.go`, `internal/state/heartbeat_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| S1 | `TestOpen_CreatesDB` | state | PASS |
| S2 | `TestOpen_WALMode` | state | PASS |
| S3 | `TestInsertWakeEvent` | state | PASS |
| S4 | `TestHasUnconsumedWakeEvents_Empty` | state | PASS |
| S5 | `TestHasUnconsumedWakeEvents_WithEvents` | state | PASS |
| S6 | `TestConsumeWakeEvents` | state | PASS |
| S7 | `TestConsumeWakeEvents_AfterConsume` | state | PASS |
| S8 | `TestSetKV_GetKV` | state | PASS |
| S9 | `TestGetKV_Missing` | state | PASS |
| S10 | `TestClose` | state | PASS |
| S11 | `TestSchemaTablesExist` | state | PASS |
| S12 | `TestListKeysWithPrefix` | state | PASS |
| S13 | `TestInstallTool_GetInstalledTools_RemoveTool` | state | PASS |
| S14 | `TestClaimInboxMessages_MarkInboxProcessed` | state | PASS |
| S15 | `TestGetHeartbeatSchedule_Empty` | state | PASS |
| S16 | `TestUpsertHeartbeatSchedule_GetHeartbeatSchedule` | state | PASS |
| S17 | `TestAcquireTaskLease_ReleaseTaskLease` | state | PASS |

**Total: 17 passed, 0 failed**

---

### 1.6 Heartbeat (`internal/heartbeat`)

**File:** `internal/heartbeat/daemon_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| H1 | `TestDefaultTasks` | heartbeat | PASS |
| H2 | `TestDaemon_StartStop` | heartbeat | PASS |
| H3 | `TestDaemon_ContextCancelStops` | heartbeat | PASS |

**Total: 3 passed, 0 failed**

---

### 1.7 Agent Loop & Context (`internal/agent`)

**Files:** `internal/agent/loop_test.go`, `internal/agent/context_test.go`, `internal/agent/prompt_test.go`, `internal/agent/token_test.go`, `internal/agent/trim_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| A1 | `TestRunOneTurn` | agent-loop | PASS |
| A2 | `TestShouldSleep_IdleTurns2` | agent-loop | PASS |
| A3 | `TestShouldSleep_IdleTurns3` | agent-loop | PASS |
| A4 | `TestBuildContextMessages_IncludesToolResults` | agent-context | PASS |
| A5 | `TestAppendToolResults_Empty` | agent-context | PASS |
| A6 | `TestAppendToolResults_WithResults` | agent-context | PASS |
| A7 | `TestBuildMessagesSafe_UnderCap` | token-caps-truncation | PASS |
| A8 | `TestBuildMessagesSafe_TruncatesWhenOverCap` | token-caps-truncation | PASS |
| A9 | `TestBuildMessagesSafe_WithMemory` | token-caps-truncation | PASS |
| A10 | `TestBuildMessagesSafe_SummaryWhenRemainingBudget` | token-caps-truncation | PASS |
| A11 | `TestBuildMessagesSafe_EffectiveCap` | token-caps-truncation | PASS |
| A12 | `TestEstimateToolTokens` | token-caps-truncation | PASS |
| A13 | `TestNaiveTokenizer_Empty` | token-caps-truncation | PASS |
| A14 | `TestNaiveTokenizer_Short` | token-caps-truncation | PASS |
| A15 | `TestNaiveTokenizer_Approximate` | token-caps-truncation | PASS |
| A16 | `TestNaiveTokenizer_LongText` | token-caps-truncation | PASS |
| A17 | `TestDefaultTokenLimits` | token-caps-truncation | PASS |
| A18 | `TestTokenLimits_WithOverrides` | token-caps-truncation | PASS |
| A19 | `TestTokenLimits_WithOverrides_ZeroPreservesDefault` | token-caps-truncation | PASS |
| A20 | `TestTiktokenTokenizer` | token-caps-truncation | PASS |
| A21 | `TestHistoryTrimmer_Compress_ShortHistory` | token-caps-truncation | PASS |
| A22 | `TestHistoryTrimmer_Compress_LongHistory` | token-caps-truncation | PASS |
| A23 | `TestHistoryTrimmer_SummarizeTurn_ToolCalls` | token-caps-truncation | PASS |
| A24 | `TestHistoryTrimmer_SummarizeTurn_ThinkingOnly` | token-caps-truncation | PASS |
| A25 | `TestBuildContextMessagesFromCompressed` | token-caps-truncation | PASS |
| A26 | `TestBuildMessagesSafe_WithHistoryCompression` | token-caps-truncation | PASS |
| A27 | `TestMessageTrimmer_Trim` | token-caps-truncation | PASS |
| A28 | `TestMessageTrimmer_Trim_NoMemoryRetriever` | token-caps-truncation | PASS |
| A29 | `TestMessageTrimmer_Trim_WithTieredRetriever` | token-caps-truncation | PASS |

**Total: 29 passed, 0 failed**

---

### 1.8 Tools (`internal/tools`)

**Files:** `internal/tools/file_read_test.go`, `file_write_test.go`, `shell_test.go`, `mutating_test.go`, `check_credits_test.go`, `list_sandboxes_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| TO1 | `TestFileReadTool_Execute` | tools | PASS |
| TO2 | `TestFileReadTool_Execute_FileNotFound` | tools | PASS |
| TO3 | `TestFileWriteTool_Execute` | tools | PASS |
| TO4 | `TestShellTool_Execute` | tools | PASS |
| TO5 | `TestShellTool_Execute_EmptyCommand` | tools | PASS |
| TO6 | `TestRegistry_Execute` | tools | PASS |
| TO7 | `TestRegistry_Execute_ExecAlias` | tools | PASS |
| TO8 | `TestIsMutatingTool` | tools | PASS |
| TO9 | `TestCheckCreditsTool_Execute` | tools | PASS |
| TO10 | `TestListSandboxesTool_Execute` | tools | PASS |

**Total: 10 passed, 0 failed**

---

### 1.9 Inference (`internal/inference`)

**Files:** `internal/inference/factory_test.go`, `internal/inference/chatjimmy_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| INF1 | `TestNewClientFromConfig_OpenAI` | inference | PASS |
| INF2 | `TestNewClientFromConfig_Conway` | inference | PASS |
| INF3 | `TestNewClientFromConfig_ExplicitProvider` | inference | PASS |
| INF4 | `TestNewClientFromConfig_ChatJimmyWhenNoKeys` | inference | PASS |
| INF5 | `TestNewClientFromConfig_BackwardCompatPriority` | inference | PASS |
| INF6 | `TestNewClientFromConfig_ChatJimmyExplicit` | inference | PASS |
| INF7 | `TestNewClientFromConfig_ChatJimmyAlias` | inference | PASS |
| INF8 | `TestNewClientFromConfig_ChatJimmyEnvBaseURL` | inference | PASS |
| INF9 | `TestLookupProvider` | inference | PASS |
| INF10 | `TestParseChatJimmyResponse` | inference | PASS |
| INF11 | `TestNewChatJimmyClient_Defaults` | inference | PASS |
| INF12 | `TestChatJimmyClient_Chat` | inference | PASS |
| INF13 | `TestChatJimmyClient_Health` | inference | PASS |
| INF14 | `TestChatJimmyClient_Health_Unhealthy` | inference | PASS |
| INF15 | `TestChatJimmyClient_Models` | inference | PASS |
| INF16 | `TestChatJimmyClient_ChatWithStats` | inference | PASS |

**Total: 16 passed, 0 failed**

---

### 1.10 Identity (`internal/identity`)

**Files:** `internal/identity/chain_test.go`, `internal/identity/bootstrap_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| ID1 | `TestCAIP2ToChainType` | identity | PASS |
| ID2 | `TestValidateAddressForChain` | identity | PASS |
| ID3 | `TestAddressKeyForChain` | identity | PASS |
| ID4 | `TestDeriveAddress_NonEVM` | identity | PASS |
| ID5 | `TestValidateChainNonce` | identity | PASS |
| ID6 | `TestNamespace` | identity | PASS |
| ID7 | `TestChainIDFromCAIP2` | identity | PASS |
| ID8 | `TestChainIDToCAIP2` | identity | PASS |
| ID9 | `TestIsEVM` | identity | PASS |
| ID10 | `TestChainIDBig` | identity | PASS |
| ID11 | `TestBootstrapIdentity` | identity | PASS |
| ID12 | `TestBootstrapIdentity_NilInputs` | identity | PASS |
| ID13 | `TestGetAddressForChain` | identity | PASS |
| ID14 | `TestGetPrimaryAddress` | identity | PASS |

**Total: 14 passed, 0 failed**

---

### 1.11 Memory (`internal/memory`)

**Files:** `internal/memory/retriever_test.go`, `internal/memory/select_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| M1 | `TestFormatMemoryBlock_Empty` | memory-retrieval | PASS |
| M2 | `TestFormatMemoryBlock_Facts` | memory-retrieval | PASS |
| M3 | `TestFormatMemoryBlock_GoalsAndProcedures` | memory-retrieval | PASS |
| M4 | `TestFormatMemoryBlock_FiveTier` | memory-retrieval | PASS |
| M5 | `TestTieredMemorySelector_Select_Empty` | memory-retrieval | PASS |
| M6 | `TestTieredMemorySelector_Select_WithData` | memory-retrieval | PASS |
| M7 | `TestTieredMemorySelector_Select_RespectsBudget` | memory-retrieval | PASS |
| M8 | `TestDefaultTierConfig` | memory-retrieval | PASS |
| M9 | `TestTieredMemoryRetriever_Retrieve` | memory-retrieval | PASS |
| M10 | `TestTieredMemoryRetriever_RetrieveWithBudget` | memory-retrieval | PASS |

**Total: 10 passed, 0 failed**

---

### 1.12 Skills (`internal/skills`)

**File:** `internal/skills/loader_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| SK1 | `TestSubconscious_MergeFileAndDB` | skills-design | PASS |

**Total: 1 passed, 0 failed**

---

### 1.13 Soul (`internal/soul`)

**File:** `internal/soul/reflection_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| SO1 | `TestReflectOnSoul_NoSoul` | soul | PASS |
| SO2 | `TestReflectOnSoul_WithSoulAndGenesis` | soul | PASS |
| SO3 | `TestComputeGenesisAlignment` | soul | PASS |
| SO4 | `TestExtractCorePurpose` | soul | PASS |

**Total: 4 passed, 0 failed**

---

### 1.14 Tunnel (`internal/tunnel`)

**File:** `internal/tunnel/bootstrap_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| TN1 | `TestNewFromConfig_Nil` | tunnel | PASS |

**Total: 1 passed, 0 failed**

---

### 1.15 CLI Commands (`cmd`)

**Files:** `cmd/init_test.go`, `cmd/status_test.go`, `cmd/run_test.go`, `cmd/test_api_test.go`

| ID | Test | Spec / Traceability | Status |
|----|------|---------------------|--------|
| CLI1 | `TestRunInit` | cli | PASS |
| CLI2 | `TestRunInit_Idempotent` | cli | PASS |
| CLI3 | `TestRunRun_NoConfig` | cli | PASS |
| CLI4 | `TestRunStatus_NoConfig` | cli | PASS |
| CLI5 | `TestRunStatus_WithConfig` | cli | PASS |
| CLI6 | `TestRunTestAPI_NoConfig` | cli | PASS |
| CLI7 | `TestRunTestAPI_ChatJimmyOK` | cli | PASS |

**Total: 7 passed, 0 failed**

---

### 1.16 Aggregate

| Suite | Passed | Failed | Total |
|-------|--------|--------|-------|
| config | 14 | 0 | 14 |
| types | 4 | 0 | 4 |
| conway | 8 | 0 | 8 |
| agent (policy) | 21 | 0 | 21 |
| state | 17 | 0 | 17 |
| heartbeat | 3 | 0 | 3 |
| agent (loop, context, prompt, token, trim) | 29 | 0 | 29 |
| tools | 10 | 0 | 10 |
| inference | 16 | 0 | 16 |
| identity | 14 | 0 | 14 |
| memory | 10 | 0 | 10 |
| skills | 1 | 0 | 1 |
| soul | 4 | 0 | 4 |
| tunnel | 1 | 0 | 1 |
| cmd | 7 | 0 | 7 |
| **Total** | **159** | **0** | **159** |

---

## 2. How to Run & Verification

### 2.1 Commands

| Command | Scope | Result |
|---------|-------|--------|
| `make test` | All tests | PASS |
| `go test ./...` | All packages | PASS |
| `go test ./internal/... -v` | Internal packages only | PASS |
| `go test -race ./...` | Race detector | PASS |
| `make test-coverage` | Coverage report | coverage.html |

### 2.2 Test Run Output (Last Verified: 17 Mar 2026)

```bash
$ make test
go test ./...
ok  	github.com/morpheumlabs/mormoneyos-go/cmd	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/agent	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/config	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/conway	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/heartbeat	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/identity	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/inference	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/memory	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/skills	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/soul	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/state	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/tools	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/tunnel	(cached)
ok  	github.com/morpheumlabs/mormoneyos-go/internal/types	(cached)
```

**Aggregate:** 159+ tests passed, 0 failed.

---

## 3. Acceptance Criteria

| Criterion | Status |
|----------|--------|
| All unit tests pass: `make test` | PASS |
| No race conditions: `go test -race ./...` | PASS |
| CLI smoke (init, status) succeeds | PASS |
| Policy engine denies protected paths and external dangerous tools | PASS |
| `test-api` succeeds with ChatJimmy config + mock server | PASS |
| Coverage ≥ 70% for `internal/` (optional) | Pending |
| Run bootstrap completes without panic (manual stop) | Manual |

---

## 4. Traceability

| Design Doc | Coverage |
|------------|----------|
| **config** | C1–C14 |
| **types** | T1–T4 |
| **conway-credits** | CR1–CR8 |
| **policy-engine** | P1–P21 |
| **state** | S1–S17 |
| **heartbeat** | H1–H3 |
| **agent-loop** | A1–A6 |
| **token-caps-truncation** | A7–A29 |
| **tools** | TO1–TO10 |
| **inference** | INF1–INF16 |
| **identity** | ID1–ID14 |
| **memory-retrieval** | M1–M10 |
| **skills-design** | SK1 |
| **soul** | SO1–SO4 |
| **tunnel** | TN1 |
| **cli** | CLI1–CLI7 |

---

## 5. Run Commands

```bash
# All tests
make test

# Verbose
go test ./... -v

# Race detector
go test -race ./...

# Coverage
make test-coverage

# CLI smoke
./bin/moneyclaw --help && ./bin/moneyclaw init && ./bin/moneyclaw status

# E2E (manual)
AUTOMATON_DIR=/tmp/moneyclaw-test ./bin/moneyclaw init && echo "agent\nprompt\n0x0\n\n" | ./bin/moneyclaw setup && ./bin/moneyclaw run

# Soak test
bash scripts/soak-test.sh [hours] [db_path]
```

---

## 6. Related Documents

- [mormoneyOS design](./design/) — Design docs
- [ARCHITECTURE.md](../ARCHITECTURE.md) — System architecture
- [API_REFERENCE.md](./API_REFERENCE.md) — API documentation
- [memory-retrieval-step6.md](./design/memory-retrieval-step6.md) — Memory retrieval
- [token-caps-truncation.md](./design/token-caps-truncation.md) — Token caps, truncation, prefill limit avoidance
- [context-trimming-stage2.md](./design/context-trimming-stage2.md) — HistoryTrimmer, TieredMemorySelector, MessageTrimmer
- [skills-design.md](./design/skills-design.md) — Skills loader
