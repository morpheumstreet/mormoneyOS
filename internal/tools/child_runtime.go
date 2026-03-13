package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// ChildSpawnStore provides DB operations for spawn_child (extends FundChildStore).
type ChildSpawnStore interface {
	FundChildStore
	GetAllChildren() ([]state.Child, bool)
	InsertChild(c state.Child) error
	UpdateChildStatus(id, status string) error
	UpdateChildSandboxID(id, sandboxID string) error
	UpdateChildAddress(id, address string) error
	SetKV(key, value string) error
	GetKV(key string) (string, bool, error)
}

// SocialClient sends messages to child agents via relay (optional).
// When nil, message_child returns "Social relay not configured."
type SocialClient interface {
	Send(toAddress string, payload string) (id string, err error)
}


func isValidWalletAddressChild(addr string) bool {
	return ethAddressRegex.MatchString(addr) && addr != "0x"+strings.Repeat("0", 40)
}

// SpawnChildTool spawns a child automaton in a new Conway sandbox.
type SpawnChildTool struct {
	Conway         conway.Client
	Store          ChildSpawnStore
	ParentAddress  string
	GenesisPrompt  string
	ParentName     string
	MaxChildren    int
}

func (SpawnChildTool) Name() string        { return "spawn_child" }
func (SpawnChildTool) Description() string { return "Spawn a child automaton in a new Conway sandbox with lifecycle tracking." }
func (SpawnChildTool) Parameters() string {
	return `{"type":"object","properties":{"name":{"type":"string","description":"Name for the child (alphanumeric + dash, max 64 chars)"},"specialization":{"type":"string","description":"What the child should specialize in"},"message":{"type":"string","description":"Message to the child"}},"required":["name"]}`
}

func (t *SpawnChildTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "spawn_child requires store"}
	}
	name, _ := args["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidArgs{Msg: "name required"}
	}
	if len(name) > 64 || !regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(name) {
		return "", ErrInvalidArgs{Msg: "name must be alphanumeric with dashes only, max 64 chars"}
	}

	children, ok := t.Store.GetAllChildren()
	if !ok {
		return "Children table not available.", nil
	}
	active := 0
	for _, c := range children {
		if c.Status != "dead" && c.Status != "cleaned_up" && c.Status != "failed" {
			active++
		}
	}
	maxChildren := t.MaxChildren
	if maxChildren <= 0 {
		maxChildren = 3
	}
	if active >= maxChildren {
		return fmt.Sprintf("Cannot spawn: already at max children (%d). Kill or wait for existing children to die.", maxChildren), nil
	}

	childID := uuid.New().String()
	genesisPrompt := t.GenesisPrompt
	if spec, _ := args["specialization"].(string); spec != "" {
		genesisPrompt += "\n\n<specialization>\nYou are a specialized child agent. Your specific focus:\n" + spec + "\n</specialization>"
	}
	genesisPrompt += "\n\n<lineage>\nYou were spawned by " + t.ParentName + " (" + t.ParentAddress + ").\nYou inherit their mission but have your own identity and wallet.\n</lineage>"
	if len(genesisPrompt) > 32000 {
		genesisPrompt = genesisPrompt[:32000]
	}
	creatorMsg, _ := args["message"].(string)

	// Insert child (sandbox_id empty initially)
	if err := t.Store.InsertChild(state.Child{
		ID:               childID,
		Name:             name,
		Address:          "",
		SandboxID:        "",
		GenesisPrompt:    genesisPrompt,
		CreatorMessage:   creatorMsg,
		FundedAmountCents: 0,
		Status:           "requested",
	}); err != nil {
		return "", fmt.Errorf("insert child: %w", err)
	}

	sandboxName := "automaton-child-" + strings.ToLower(regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "-"))
	sandbox, err := t.Conway.CreateSandbox(ctx, conway.CreateSandboxOptions{
		Name:     sandboxName,
		VCPU:     1,
		MemoryMB: 512,
		DiskGB:   5,
	})
	if err != nil {
		_ = t.Store.UpdateChildStatus(childID, "failed")
		return "", fmt.Errorf("create sandbox: %w", err)
	}
	var cleanupSandbox bool = true
	defer func() {
		if cleanupSandbox {
			_ = t.Conway.DeleteSandbox(ctx, sandbox.ID)
		}
	}()

	if err := t.Store.UpdateChildSandboxID(childID, sandbox.ID); err != nil {
		return "", fmt.Errorf("update sandbox id: %w", err)
	}

	// Install runtime in child sandbox
	_, err = t.Conway.ExecInSandbox(ctx, sandbox.ID, "apt-get update -qq && apt-get install -y -qq nodejs npm git curl", 120_000)
	if err != nil {
		_ = t.Store.UpdateChildStatus(childID, "failed")
		return "", fmt.Errorf("install deps: %w", err)
	}
	_, _ = t.Conway.ExecInSandbox(ctx, sandbox.ID, "npm install -g @conway/automaton@latest 2>/dev/null || true", 60_000)

	// Write genesis
	genesisMap := map[string]string{
		"name":            name,
		"genesisPrompt":   genesisPrompt,
		"creatorMessage":  creatorMsg,
		"creatorAddress":  t.ParentAddress,
		"parentAddress":   t.ParentAddress,
	}
	genesisBytes, _ := json.MarshalIndent(genesisMap, "", "  ")
	if err := t.Conway.WriteFileInSandbox(ctx, sandbox.ID, "/root/.automaton/genesis.json", string(genesisBytes)); err != nil {
		_ = t.Store.UpdateChildStatus(childID, "failed")
		return "", fmt.Errorf("write genesis: %w", err)
	}

	// Propagate constitution (optional - may not exist locally)
	propagateConstitution(ctx, t.Conway, t.Store, sandbox.ID)

	if err := t.Store.UpdateChildStatus(childID, "runtime_ready"); err != nil {
		return "", fmt.Errorf("update status: %w", err)
	}

	// Init child wallet
	result, err := t.Conway.ExecInSandbox(ctx, sandbox.ID, "automaton --init 2>&1", 60_000)
	if err != nil {
		_ = t.Store.UpdateChildStatus(childID, "failed")
		return "", fmt.Errorf("init wallet: %w", err)
	}
	walletMatch := ethAddressRegex.FindString(result.Stdout)
	if !isValidWalletAddressChild(walletMatch) {
		_ = t.Store.UpdateChildStatus(childID, "failed")
		return "", fmt.Errorf("child wallet address invalid: %s", walletMatch)
	}
	if err := t.Store.UpdateChildAddress(childID, walletMatch); err != nil {
		return "", fmt.Errorf("update address: %w", err)
	}
	if err := t.Store.UpdateChildStatus(childID, "wallet_verified"); err != nil {
		return "", fmt.Errorf("update status: %w", err)
	}

	cleanupSandbox = false // success; do not delete sandbox
	return fmt.Sprintf("Child spawned: %s in sandbox %s (status: wallet_verified)", name, sandbox.ID), nil
}

func propagateConstitution(ctx context.Context, c conway.Client, store ChildSpawnStore, sandboxID string) {
	// TODO: read local constitution from ~/.automaton/constitution.md when available
	// For now we skip - no local config path in tool context
	_ = ctx
	_ = c
	_ = store
	_ = sandboxID
}

// StartChildTool starts a funded child automaton.
type StartChildTool struct {
	Conway conway.Client
	Store  interface {
		GetChildByID(id string) (*state.Child, bool)
		UpdateChildStatus(id, status string) error
	}
}

func (StartChildTool) Name() string        { return "start_child" }
func (StartChildTool) Description() string { return "Start a funded child automaton. Transitions from funded to starting." }
func (StartChildTool) Parameters() string {
	return `{"type":"object","properties":{"child_id":{"type":"string","description":"Child automaton ID"}},"required":["child_id"]}`
}

func (t *StartChildTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "start_child requires store"}
	}
	childID, _ := args["child_id"].(string)
	childID = strings.TrimSpace(childID)
	if childID == "" {
		return "", ErrInvalidArgs{Msg: "child_id required"}
	}
	child, ok := t.Store.GetChildByID(childID)
	if !ok || child == nil {
		return fmt.Sprintf("Child %s not found.", childID), nil
	}
	if child.Status != "funded" && child.Status != "wallet_verified" {
		return fmt.Sprintf("Child %s status is %q; must be funded or wallet_verified to start.", child.Name, child.Status), nil
	}

	_ = t.Store.UpdateChildStatus(childID, "starting")
	_, err := t.Conway.ExecInSandbox(ctx, child.SandboxID, "automaton --init && automaton --provision && systemctl start automaton 2>/dev/null || automaton --run &", 60_000)
	if err != nil {
		_ = t.Store.UpdateChildStatus(childID, "unhealthy")
		return "", fmt.Errorf("start child: %w", err)
	}
	_ = t.Store.UpdateChildStatus(childID, "healthy")
	return fmt.Sprintf("Child %s started and healthy.", child.Name), nil
}

// MessageChildTool sends a signed message to a child via social relay.
type MessageChildTool struct {
	Social SocialClient
	Store  interface {
		GetChildByID(id string) (*state.Child, bool)
	}
}

func (MessageChildTool) Name() string        { return "message_child" }
func (MessageChildTool) Description() string { return "Send a signed message to a child automaton via social relay." }
func (MessageChildTool) Parameters() string {
	return `{"type":"object","properties":{"child_id":{"type":"string"},"content":{"type":"string"},"type":{"type":"string","description":"Message type (default: parent_message)"}},"required":["child_id","content"]}`
}

func (t *MessageChildTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Social == nil {
		return "Social relay not configured. Set socialRelayUrl in config.", nil
	}
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "message_child requires store"}
	}
	childID, _ := args["child_id"].(string)
	childID = strings.TrimSpace(childID)
	if childID == "" {
		return "", ErrInvalidArgs{Msg: "child_id required"}
	}
	content, _ := args["content"].(string)
	if content == "" {
		return "", ErrInvalidArgs{Msg: "content required"}
	}
	msgType, _ := args["type"].(string)
	if msgType == "" {
		msgType = "parent_message"
	}
	child, ok := t.Store.GetChildByID(childID)
	if !ok || child == nil {
		return fmt.Sprintf("Child %s not found.", childID), nil
	}
	payload := fmt.Sprintf(`{"type":%q,"content":%q,"sentAt":"%s"}`, msgType, content, "now")
	id, err := t.Social.Send(child.Address, payload)
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}
	return fmt.Sprintf("Message sent to child %s (id: %s)", child.Name, id), nil
}

// VerifyChildConstitutionTool verifies the constitution integrity of a child.
type VerifyChildConstitutionTool struct {
	Conway conway.Client
	Store  interface {
		GetChildByID(id string) (*state.Child, bool)
		GetKV(key string) (string, bool, error)
	}
}

func (VerifyChildConstitutionTool) Name() string        { return "verify_child_constitution" }
func (VerifyChildConstitutionTool) Description() string { return "Verify the constitution integrity of a child automaton." }
func (VerifyChildConstitutionTool) Parameters() string {
	return `{"type":"object","properties":{"child_id":{"type":"string"}},"required":["child_id"]}`
}

func (t *VerifyChildConstitutionTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.Conway == nil {
		return "", ErrConwayNotConfigured
	}
	if t.Store == nil {
		return "", ErrInvalidArgs{Msg: "verify_child_constitution requires store"}
	}
	childID, _ := args["child_id"].(string)
	childID = strings.TrimSpace(childID)
	if childID == "" {
		return "", ErrInvalidArgs{Msg: "child_id required"}
	}
	child, ok := t.Store.GetChildByID(childID)
	if !ok || child == nil {
		return fmt.Sprintf("Child %s not found.", childID), nil
	}
	storedHash, ok, _ := t.Store.GetKV("constitution_hash:" + child.SandboxID)
	if !ok || storedHash == "" {
		return `{"valid":false,"detail":"no stored constitution hash found"}`, nil
	}
	content, err := t.Conway.ReadFileInSandbox(ctx, child.SandboxID, "/root/.automaton/constitution.md")
	if err != nil {
		return fmt.Sprintf(`{"valid":false,"detail":"failed to read child constitution: %s"}`, err.Error()), nil
	}
	h := sha256.Sum256([]byte(content))
	childHash := hex.EncodeToString(h[:])
	if childHash == storedHash {
		return `{"valid":true,"detail":"constitution hash matches"}`, nil
	}
	return fmt.Sprintf(`{"valid":false,"detail":"hash mismatch: expected %s..., got %s..."}`, storedHash[:16], childHash[:16]), nil
}
