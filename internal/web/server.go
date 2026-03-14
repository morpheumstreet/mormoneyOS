package web

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/conway"
	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
	"github.com/morpheumlabs/mormoneyos-go/internal/inference"
	"github.com/morpheumlabs/mormoneyos-go/internal/identity/signverify"
	"github.com/morpheumlabs/mormoneyos-go/internal/social"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/tunnel"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

//go:embed static
var staticFS embed.FS

const Version = "0.1.0"

// WebDB is the minimal DB interface for the web server (pause/resume, status).
type WebDB interface {
	InsertWakeEvent(source, reason string) error
	SetKV(key, value string) error
	GetKV(key string) (string, bool, error)
	DeleteKV(key string) error
	SetAgentState(state string) error
	GetAgentState() (string, bool, error)
	GetTurnCount() (int64, error)
	GetIdentity(key string) (string, bool, error)
}

// CreditsGetter returns credits balance (e.g. Conway client).
type CreditsGetter interface {
	GetCreditsBalance(ctx context.Context) (int64, error)
}

// ToolsLister lists available tools (name, description) for the config UI.
type ToolsLister interface {
	List() []string
	Schemas() []inference.ToolDefinition
}

// ServerConfig holds config for status API (TS-aligned).
type ServerConfig struct {
	Name            string
	WalletAddress   string
	CreatorAddress  string // Only this address may pass login wallet verification
	DefaultChain    string // CAIP-2, e.g. eip155:8453
	Version       string
	CreditsGetter CreditsGetter
	JWTSecret     string // For issuing tokens on wallet verify; if empty, a random one is used (tokens invalid on restart)
	ChatClient    inference.Client // Optional; when set, Agent Comm Link uses LLM for chat
	ToolsLister    ToolsLister       // Optional; when set, GET /api/tools and PATCH /api/tools/:name work
	TunnelManager  *tunnel.TunnelManager // Optional; when set, GET /api/tunnels returns active tunnels
	TunnelReloader      func(cfg *types.TunnelConfig) // Optional; when set, POST /api/tunnels/providers/{name}/restart reloads providers
	SkillsConfigGetter func() *types.SkillsConfig     // Optional; when set, skills API uses this for registry config (DI)
}

// RuntimeState holds shared runtime state for the web API.
type RuntimeState struct {
	mu         sync.RWMutex
	Paused     bool
	Running    bool
	TickNum    int64
	AgentState string
}

// Server is the MoneyClaw web dashboard HTTP server.
type Server struct {
	Addr       string
	State      *RuntimeState
	DB         WebDB
	Cfg        *ServerConfig
	Log        *slog.Logger
	mux        *http.ServeMux
	httpServer *http.Server
}

// NewServer creates a new web server.
func NewServer(addr string, state *RuntimeState, db WebDB, cfg *ServerConfig, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	if cfg == nil {
		cfg = &ServerConfig{Version: Version}
	}
	if cfg.Version == "" {
		cfg.Version = Version
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = os.Getenv("MONEYCLAW_JWT_SECRET")
	}
	if cfg.JWTSecret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err == nil {
			cfg.JWTSecret = hex.EncodeToString(b)
		}
	}
	s := &Server{Addr: addr, State: state, DB: db, Cfg: cfg, Log: log, mux: http.NewServeMux()}
	s.routes()
	s.loadPausedFromDB()
	return s
}

func (s *Server) routes() {
	// Static assets (embedded) — subFS so /static/ serves static/*
	staticSub, _ := fs.Sub(staticFS, "static")
	s.mux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.FS(staticSub))))

	// Dashboard
	s.mux.HandleFunc("/", s.handleIndex)

	// API (aligned with moneyclaw-py)
	s.mux.HandleFunc("GET /api/status", s.handleAPIStatus)
	s.mux.HandleFunc("GET /api/strategies", s.handleAPIStrategies)
	s.mux.HandleFunc("GET /api/history", s.handleAPIHistory)
	s.mux.HandleFunc("GET /api/cost", s.handleAPICost)
	s.mux.HandleFunc("GET /api/risk", s.handleAPIRisk)
	s.mux.HandleFunc("POST /api/pause", s.handleAPIPause)
	s.mux.HandleFunc("POST /api/resume", s.handleAPIResume)
	s.mux.HandleFunc("POST /api/chat", s.handleAPIChat)
	s.mux.HandleFunc("GET /api/config", s.handleAPIConfigGet)
	s.mux.HandleFunc("PUT /api/config", s.handleAPIConfigPut)
	s.mux.HandleFunc("POST /api/auth/verify", s.handleAPIAuthVerify)
	if os.Getenv("MONEYCLAW_DEV_BYPASS") == "1" {
		s.mux.HandleFunc("POST /api/auth/dev-bypass", s.handleAPIAuthDevBypass)
	}
	s.mux.HandleFunc("GET /api/reports", s.handleAPIReports)
	s.mux.HandleFunc("GET /api/tunnels/providers", s.handleAPITunnelsProviders)
	s.mux.HandleFunc("PUT /api/tunnels/providers/{name}", s.handleAPITunnelsProviderPut)
	s.mux.HandleFunc("POST /api/tunnels/providers/{name}/restart", s.handleAPITunnelsProviderRestart)
	s.mux.HandleFunc("GET /api/tunnels", s.handleAPITunnels)
	s.mux.HandleFunc("GET /api/tools", s.handleAPIToolsList)
	s.mux.HandleFunc("PATCH /api/tools/{name}", s.handleAPIToolsPatch)
	s.mux.HandleFunc("GET /api/social", s.handleAPISocialList)
	s.mux.HandleFunc("PATCH /api/social/{name}", s.handleAPISocialPatch)
	s.mux.HandleFunc("PUT /api/social/{name}/config", s.handleAPISocialConfigPut)
	s.mux.HandleFunc("GET /api/soul/config", s.handleAPISoulConfigGet)
	s.mux.HandleFunc("PUT /api/soul/config", s.handleAPISoulConfigPut)
	s.mux.HandleFunc("POST /api/soul/enhance", s.handleAPISoulEnhance)
	s.mux.HandleFunc("GET /api/economic", s.handleAPIEconomicGet)
	s.mux.HandleFunc("PUT /api/economic", s.handleAPIEconomicPut)
	s.mux.HandleFunc("GET /api/models", s.handleAPIModelsList)
	s.mux.HandleFunc("POST /api/models", s.handleAPIModelsPost)
	s.mux.HandleFunc("PUT /api/models/order", s.handleAPIModelsOrder)
	s.mux.HandleFunc("PATCH /api/models/{id}", s.handleAPIModelsPatch)
	s.mux.HandleFunc("DELETE /api/models/{id}", s.handleAPIModelsDelete)

	// Skills API (list, CRUD, discovery, recommended, activate/deactivate)
	s.mux.HandleFunc("GET /api/skills", s.handleAPISkillsList)
	s.mux.HandleFunc("GET /api/skills/discovery", s.handleAPISkillsDiscovery)
	s.mux.HandleFunc("GET /api/skills/recommended", s.handleAPISkillsRecommended)
	s.mux.HandleFunc("GET /api/skills/{name}", s.handleAPISkillsGet)
	s.mux.HandleFunc("POST /api/skills", s.handleAPISkillsPost)
	s.mux.HandleFunc("PATCH /api/skills/{name}", s.handleAPISkillsPatch)
	s.mux.HandleFunc("DELETE /api/skills/{name}", s.handleAPISkillsDelete)
	s.mux.HandleFunc("PATCH /api/skills/{name}/activate", s.handleAPISkillsActivate)
	s.mux.HandleFunc("PATCH /api/skills/{name}/deactivate", s.handleAPISkillsDeactivate)

	// Heartbeat schedule API
	s.mux.HandleFunc("GET /api/heartbeat", s.handleAPIHeartbeatList)
	s.mux.HandleFunc("PATCH /api/heartbeat/{name}", s.handleAPIHeartbeatPatch)
	s.mux.HandleFunc("PATCH /api/heartbeat/{name}/schedule", s.handleAPIHeartbeatSchedulePatch)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (s *Server) loadPausedFromDB() {
	if s.DB == nil {
		return
	}
	agentState, ok, _ := s.DB.GetAgentState()
	if !ok || agentState != "sleeping" {
		return
	}
	sleepUntil, ok, _ := s.DB.GetKV("sleep_until")
	if ok && sleepUntil != "" {
		s.State.mu.Lock()
		s.State.Paused = true
		s.State.mu.Unlock()
	}
}

func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	s.State.mu.RLock()
	running := s.State.Running
	paused := s.State.Paused
	agentState := s.State.AgentState
	tickNum := s.State.TickNum
	s.State.mu.RUnlock()

	// TS-aligned: is_running, state, tick_count, wallet_value, today_pnl, dry_run, address, name, version
	tickCount := int64(0)
	if s.DB != nil {
		tickCount, _ = s.DB.GetTurnCount()
	}
	if tickCount == 0 {
		tickCount = tickNum
	}
	walletValue := 0.0
	if s.Cfg != nil && s.Cfg.CreditsGetter != nil {
		if cents, err := s.Cfg.CreditsGetter.GetCreditsBalance(r.Context()); err == nil {
			walletValue = float64(cents) / 100
		}
	}
	if walletValue == 0 && s.DB != nil {
		if v, ok, _ := s.DB.GetKV("credits_cents"); ok && v != "" {
			var c int
			if _, err := fmt.Sscanf(v, "%d", &c); err == nil {
				walletValue = float64(c) / 100
			}
		}
	}
	address := "0x0"
	chainParam := r.URL.Query().Get("chain")
	if chainParam == "" && s.Cfg != nil && s.Cfg.DefaultChain != "" {
		chainParam = s.Cfg.DefaultChain
	}
	if chainParam == "" {
		chainParam = identity.DefaultChainBase
	}
	if s.DB != nil {
		if chainParam != "" {
			if a, ok, _ := s.DB.GetIdentity(identity.AddressKeyForChain(chainParam)); ok && a != "" {
				address = a
			}
		}
		if address == "0x0" {
			if a, ok, _ := s.DB.GetIdentity("address"); ok && a != "" {
				address = a
			}
		}
	}
	if address == "0x0" && s.Cfg != nil && s.Cfg.WalletAddress != "" {
		address = s.Cfg.WalletAddress
	}
	if address == "0x0" && chainParam != "" {
		if addr, err := identity.DeriveAddress(chainParam); err == nil && addr != "" {
			address = addr
		}
	}
	name := ""
	version := Version
	if s.Cfg != nil {
		if s.Cfg.Name != "" {
			name = s.Cfg.Name
		}
		if s.Cfg.Version != "" {
			version = s.Cfg.Version
		}
	}
	todayPnl := 0.0

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"is_running":    running && (agentState == "running" || agentState == "waking"),
		"state":         agentState,
		"tick_count":    tickCount,
		"wallet_value":  walletValue,
		"today_pnl":     todayPnl,
		"dry_run":       true,
		"address":       address,
		"chain":         chainParam,
		"name":          name,
		"version":       version,
		"running":       running,
		"paused":        paused,
		"agent_state":   agentState,
		"tick":          tickNum,
	})
}

func (s *Server) handleAPIStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := []map[string]any{}
	if s.DB != nil {
		// Skills from DB (TS-aligned)
		if skillsDB, ok := s.DB.(interface {
			GetSkills() ([]map[string]any, bool)
		}); ok {
			if skills, ok := skillsDB.GetSkills(); ok && len(skills) > 0 {
				strategies = append(strategies, skills...)
			}
		}
		// Children from DB when table exists (TS-aligned)
		if childrenDB, ok := s.DB.(interface {
			GetChildren() ([]map[string]any, bool)
		}); ok {
			if children, ok := childrenDB.GetChildren(); ok && len(children) > 0 {
				strategies = append(strategies, children...)
			}
		}
	}
	if len(strategies) == 0 {
		strategies = []map[string]any{
			{"name": "agent", "description": "Core ReAct agent loop", "risk_level": "low", "enabled": true},
			{"name": "crypto_dca", "description": "DCA into crypto", "risk_level": "low", "enabled": false},
			{"name": "crypto_price_alert", "description": "Price threshold alerts", "risk_level": "low", "enabled": false},
			{"name": "crypto_funding", "description": "Funding rate arbitrage", "risk_level": "medium", "enabled": false},
			{"name": "smart_rebalance", "description": "Portfolio rebalancing", "risk_level": "medium", "enabled": false},
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategies)
}

func (s *Server) handleAPIHistory(w http.ResponseWriter, r *http.Request) {
	// Placeholder: memory/history not yet implemented
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]any{})
}

func (s *Server) handleAPICost(w http.ResponseWriter, r *http.Request) {
	todayCost, todayCalls, totalCost := 0.0, 0.0, 0.0
	if s.DB != nil {
		if costDB, ok := s.DB.(interface {
			GetInferenceCostSummary() (todayCost, todayCalls, totalCost float64, ok bool)
		}); ok {
			todayCost, todayCalls, totalCost, _ = costDB.GetInferenceCostSummary()
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"today_cost":     todayCost,
		"today_calls":    int(todayCalls),
		"total_cost":     totalCost,
		"over_budget":    false,
		"by_layer":       map[string]float64{},
		"calls_by_layer": map[string]int{},
	})
}

func (s *Server) handleAPIRisk(w http.ResponseWriter, r *http.Request) {
	s.State.mu.RLock()
	defer s.State.mu.RUnlock()
	json.NewEncoder(w).Encode(map[string]any{
		"paused":     s.State.Paused,
		"daily_loss": 0.0,
		"risk_level": "LOW",
	})
}

func (s *Server) handleAPIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		payload.Message = ""
	}
	msg := strings.TrimSpace(payload.Message)
	msgLower := strings.ToLower(msg)

	// Fast shortcuts: status, help
	if strings.Contains(msgLower, "status") && s.DB != nil {
		agentState, _, _ := s.DB.GetAgentState()
		if agentState == "" {
			agentState = "waking"
		}
		tickCount, _ := s.DB.GetTurnCount()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"response": fmt.Sprintf("System is %s. Turn count: %d", strings.ToUpper(agentState), tickCount),
		})
		return
	}
	if strings.Contains(msgLower, "help") || strings.Contains(msgLower, "帮助") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"response": "I can help with:\n1. status — show agent state\n2. strategies — list skills/children\n3. cost — LLM cost summary\n\nTry: 'status' or 'strategies'",
		})
		return
	}

	// Use LLM when ChatClient is configured (default: chatjimmy)
	if s.Cfg != nil && s.Cfg.ChatClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		messages := []inference.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant for the mormoneyOS agent dashboard. Respond briefly and helpfully."},
			{Role: "user", Content: msg},
		}
		resp, err := s.Cfg.ChatClient.Chat(ctx, messages, &inference.InferenceOptions{MaxTokens: 512})
		if err != nil {
			s.Log.Warn("chat inference failed", "err", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"response": fmt.Sprintf("Chat error: %v", err),
			})
			return
		}
		content := strings.TrimSpace(resp.Content)
		if content == "" {
			content = "(No response)"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"response": content,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"response": fmt.Sprintf("I don't understand %q. Type 'help' for commands. (Agent Comm Link requires inference client.)", payload.Message),
	})
}

func (s *Server) handleAPIPause(w http.ResponseWriter, r *http.Request) {
	if s.DB != nil {
		_ = s.DB.SetAgentState("sleeping")
		// TS: sleep_until far future so main loop stays sleeping until resume
		farFuture := "2099-12-31T23:59:59.000Z"
		_ = s.DB.SetKV("sleep_until", farFuture)
	}
	s.State.mu.Lock()
	s.State.Paused = true
	s.State.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "paused"})
}

func (s *Server) handleAPIResume(w http.ResponseWriter, r *http.Request) {
	if s.DB != nil {
		_ = s.DB.DeleteKV("sleep_until")
		_ = s.DB.InsertWakeEvent("web", "resume from dashboard")
	}
	s.State.mu.Lock()
	s.State.Paused = false
	s.State.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
}

func (s *Server) handleAPIConfigGet(w http.ResponseWriter, r *http.Request) {
	path := config.GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Config file not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"content": string(data)})
}

func (s *Server) handleAPIConfigPut(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	if len(body) < 2 {
		http.Error(w, "Config body too short", http.StatusBadRequest)
		return
	}
	var cfg types.AutomatonConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := config.Save(&cfg); err != nil {
		s.Log.Error("config save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Chain        string `json:"chain"`
		Message      string `json:"message"`
		Signature    string `json:"signature"`
		Address      string `json:"address"`
		ECPubBytes   string `json:"ecPubBytes"`
		MLDSAPubBytes string `json:"mldsaPubBytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Chain == "" || req.Message == "" || req.Signature == "" {
		http.Error(w, "chain, message, and signature are required", http.StatusBadRequest)
		return
	}

	chain := signverify.ChainType(strings.ToLower(req.Chain))
	var ecPub, mldsaPub []byte
	if req.ECPubBytes != "" {
		ecPub, _ = decodeBase64(req.ECPubBytes)
	}
	if req.MLDSAPubBytes != "" {
		mldsaPub, _ = decodeBase64(req.MLDSAPubBytes)
	}

	result, err := signverify.VerifyWithAddress(chain, req.Message, req.Signature, req.Address, ecPub, mldsaPub)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"valid": false, "error": err.Error()})
		return
	}
	if !result.Valid {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"valid": false, "error": "signature verification failed"})
		return
	}

	// Require verified address to match creatorAddress from internal config.
	// Treat zero address (setup default) as "no restriction".
	const zeroAddr = "0x0000000000000000000000000000000000000000"
	creatorAddr := ""
	if s.Cfg != nil && s.Cfg.CreatorAddress != "" {
		creatorAddr = strings.TrimSpace(strings.ToLower(s.Cfg.CreatorAddress))
	}
	if creatorAddr != "" && creatorAddr != zeroAddr && !strings.EqualFold(strings.TrimSpace(result.Address), creatorAddr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"valid": false, "error": "wallet address does not match creator"})
		return
	}

	// Issue JWT for user operations (pause, resume, chat, config)
	token, err := s.issueJWT(result.Address)
	if err != nil {
		s.Log.Error("jwt issue failed", "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"valid": false, "error": "failed to issue token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"valid":   true,
		"address": result.Address,
		"token":   token,
	})
}

func (s *Server) issueJWT(address string) (string, error) {
	secret := ""
	if s.Cfg != nil {
		secret = s.Cfg.JWTSecret
	}
	if secret == "" {
		return "", fmt.Errorf("jwt secret not configured")
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"address": address,
		"iat":     now.Unix(),
		"exp":     now.Add(24 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// handleAPIAuthDevBypass returns a JWT for address "0xdev" when MONEYCLAW_DEV_BYPASS=1.
// For agent browser / automated testing. Accepts POST with optional JSON body (ignored).
func (s *Server) handleAPIAuthDevBypass(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if os.Getenv("MONEYCLAW_DEV_BYPASS") != "1" {
		http.Error(w, "dev bypass disabled", http.StatusNotFound)
		return
	}
	token, err := s.issueJWT("0xdev")
	if err != nil {
		s.Log.Error("dev bypass jwt failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"valid":   true,
		"address": "0xdev",
		"token":   token,
	})
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func (s *Server) handleAPIReports(w http.ResponseWriter, r *http.Request) {
	lastReport := ""
	if s.DB != nil {
		if v, ok, _ := s.DB.GetKV("last_metrics_report"); ok && v != "" {
			lastReport = v
		}
	}
	snapshots := []map[string]any{}
	if s.DB != nil {
		if db, ok := s.DB.(*state.Database); ok {
			rows, err := db.MetricsGetRecent(20)
			if err == nil {
				for _, row := range rows {
					var metrics, alerts any
					_ = json.Unmarshal([]byte(row.MetricsJSON), &metrics)
					_ = json.Unmarshal([]byte(row.AlertsJSON), &alerts)
					snapshots = append(snapshots, map[string]any{
						"id":          row.ID,
						"snapshot_at": row.SnapshotAt,
						"metrics":     metrics,
						"alerts":      alerts,
					})
				}
			}
		}
	}
	var lastReportObj any
	if lastReport != "" {
		_ = json.Unmarshal([]byte(lastReport), &lastReportObj)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"last_report": lastReportObj,
		"snapshots":   snapshots,
	})
}

// tunnelProviderSchemas defines config fields per provider for the Config UI.
var tunnelProviderSchemas = map[string]map[string]any{
	"bore": {
		"fields": []map[string]any{},
	},
	"localtunnel": {
		"fields": []map[string]any{},
	},
	"cloudflare": {
		"fields": []map[string]any{
			{"name": "token", "type": "password", "required": true, "label": "Tunnel Token", "help": "Paste the tunnel token from cloudflared tunnel create (or credentials JSON content)"},
		},
	},
	"ngrok": {
		"fields": []map[string]any{
			{"name": "authToken", "type": "password", "required": true, "label": "ngrok authtoken", "help": "Your authtoken from ngrok dashboard"},
			{"name": "domain", "type": "string", "required": false, "label": "Custom domain", "help": "Optional reserved domain (ngrok paid)"},
		},
	},
	"tailscale": {
		"fields": []map[string]any{
			{"name": "authKey", "type": "password", "required": true, "label": "Tailscale Auth Key", "help": "Generate from admin console > Keys > Generate auth key (tskey-auth...)"},
			{"name": "hostname", "type": "string", "required": false, "label": "Hostname", "help": "Optional hostname for serve/funnel"},
			{"name": "funnel", "type": "boolean", "required": false, "label": "Funnel (public HTTPS)", "help": "Enable tailscale funnel for public HTTPS"},
		},
	},
	"custom": {
		"fields": []map[string]any{
			{"name": "startCommand", "type": "string", "required": true, "label": "Start command", "help": "Command with {port} and {host} placeholders"},
			{"name": "urlPattern", "type": "string", "required": false, "label": "URL pattern", "help": "Substring to find public URL in stdout (default: https://)"},
		},
	},
}

func (s *Server) handleAPITunnelsProviders(w http.ResponseWriter, r *http.Request) {
	providers := []string{"bore", "localtunnel", "cloudflare", "ngrok", "tailscale", "custom"}
	cfg, err := config.Load()
	if err != nil {
		s.Log.Warn("tunnels providers: config load failed", "err", err)
		cfg = nil
	}
	configOut := map[string]any{"defaultProvider": "bore", "providers": map[string]any{}}
	if cfg != nil && cfg.Tunnel != nil {
		configOut["defaultProvider"] = cfg.Tunnel.DefaultProvider
		if cfg.Tunnel.DefaultProvider == "" {
			configOut["defaultProvider"] = "bore"
		}
		provs := make(map[string]any)
		for name, pc := range cfg.Tunnel.Providers {
			m := map[string]any{"enabled": pc.Enabled}
			if pc.StartCommand != "" {
				m["startCommand"] = pc.StartCommand
			}
			if pc.URLPattern != "" {
				m["urlPattern"] = pc.URLPattern
			}
			mask := "***"
			if pc.Token != "" {
				m["token"] = mask
			}
			if pc.AuthToken != "" {
				m["authToken"] = mask
			}
			if pc.AuthKey != "" {
				m["authKey"] = mask
			}
			if pc.Domain != "" {
				m["domain"] = pc.Domain
			}
			if pc.Hostname != "" {
				m["hostname"] = pc.Hostname
			}
			if pc.Funnel {
				m["funnel"] = true
			}
			provs[name] = m
		}
		configOut["providers"] = provs
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"providers": providers,
		"schemas":   tunnelProviderSchemas,
		"config":    configOut,
	})
}

func (s *Server) handleAPITunnels(w http.ResponseWriter, r *http.Request) {
	tunnels := []map[string]any{}
	if s.Cfg != nil && s.Cfg.TunnelManager != nil {
		for _, t := range s.Cfg.TunnelManager.Status() {
			tunnels = append(tunnels, map[string]any{
				"port":       t.Port,
				"provider":   t.Provider,
				"public_url": t.PublicURL,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"tunnels": tunnels})
}

var tunnelProvidersWithAuth = map[string][]string{
	"cloudflare": {"token"},
	"ngrok":      {"authToken"},
	"tailscale":  {"authKey"},
}

func (s *Server) handleAPITunnelsProviderPut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "provider name required", http.StatusBadRequest)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("tunnel provider put: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		cfg = &types.AutomatonConfig{}
	}
	if cfg.Tunnel == nil {
		cfg.Tunnel = &types.TunnelConfig{Providers: make(map[string]types.TunnelProviderConfig)}
	}
	if cfg.Tunnel.Providers == nil {
		cfg.Tunnel.Providers = make(map[string]types.TunnelProviderConfig)
	}
	pc := cfg.Tunnel.Providers[name]
	if v, ok := body["enabled"].(bool); ok {
		pc.Enabled = v
	}
	if v, ok := body["token"].(string); ok {
		pc.Token = v
	}
	if v, ok := body["authToken"].(string); ok {
		pc.AuthToken = v
	}
	if v, ok := body["authKey"].(string); ok {
		pc.AuthKey = v
	}
	if v, ok := body["domain"].(string); ok {
		pc.Domain = v
	}
	if v, ok := body["hostname"].(string); ok {
		pc.Hostname = v
	}
	if v, ok := body["funnel"].(bool); ok {
		pc.Funnel = v
	}
	if v, ok := body["startCommand"].(string); ok {
		pc.StartCommand = v
	}
	if v, ok := body["urlPattern"].(string); ok {
		pc.URLPattern = v
	}
	cfg.Tunnel.Providers[name] = pc
	if err := config.Save(cfg); err != nil {
		s.Log.Error("tunnel provider put: config save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "provider": name})
}

func (s *Server) handleAPITunnelsProviderRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "provider name required", http.StatusBadRequest)
		return
	}
	if s.Cfg == nil || s.Cfg.TunnelReloader == nil {
		http.Error(w, "Tunnel reload not configured", http.StatusNotFound)
		return
	}
	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("tunnel provider restart: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil || cfg.Tunnel == nil {
		http.Error(w, "Tunnel config not found", http.StatusBadRequest)
		return
	}
	pc, hasProvider := cfg.Tunnel.Providers[name]
	if !hasProvider {
		pc = types.TunnelProviderConfig{Enabled: true}
	}
	// For providers that require API keys, verify credential is present in config
	if required, ok := tunnelProvidersWithAuth[name]; ok {
		hasCred := false
		for _, f := range required {
			switch f {
			case "token":
				if pc.Token != "" {
					hasCred = true
				}
			case "authToken":
				if pc.AuthToken != "" {
					hasCred = true
				}
			case "authKey":
				if pc.AuthKey != "" {
					hasCred = true
				}
			}
		}
		if !hasCred {
			http.Error(w, "Provider requires API key in automaton.json (token/authToken/authKey)", http.StatusBadRequest)
			return
		}
	}
	if name == "custom" && pc.StartCommand == "" {
		http.Error(w, "Custom provider requires startCommand in automaton.json", http.StatusBadRequest)
		return
	}
	s.Cfg.TunnelReloader(cfg.Tunnel)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "provider": name, "restarted": true})
}

const disabledToolsKVKey = "disabled_tools"

func (s *Server) handleAPIToolsList(w http.ResponseWriter, r *http.Request) {
	if s.Cfg == nil || s.Cfg.ToolsLister == nil {
		http.Error(w, "Tools API not configured", http.StatusNotFound)
		return
	}
	disabled := s.getDisabledTools()
	schemaMap := make(map[string]string)
	for _, def := range s.Cfg.ToolsLister.Schemas() {
		if def.Function.Name != "" {
			schemaMap[def.Function.Name] = def.Function.Description
		}
	}
	names := s.Cfg.ToolsLister.List()
	// Sort for stable output
	sortStrings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		desc := schemaMap[name]
		enabled := !disabled[name]
		out = append(out, map[string]any{
			"name":        name,
			"description": desc,
			"enabled":     enabled,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"tools": out})
}

func (s *Server) getDisabledTools() map[string]bool {
	out := make(map[string]bool)
	if s.DB == nil {
		return out
	}
	raw, ok, _ := s.DB.GetKV(disabledToolsKVKey)
	if !ok || raw == "" {
		return out
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return out
	}
	for _, n := range list {
		out[n] = true
	}
	return out
}

func (s *Server) handleAPIToolsPatch(w http.ResponseWriter, r *http.Request) {
	if s.Cfg == nil || s.Cfg.ToolsLister == nil || s.DB == nil {
		http.Error(w, "Tools API not configured", http.StatusNotFound)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "tool name required", http.StatusBadRequest)
		return
	}
	validNames := make(map[string]bool)
	for _, n := range s.Cfg.ToolsLister.List() {
		validNames[n] = true
	}
	if !validNames[name] {
		http.Error(w, "unknown tool: "+name, http.StatusNotFound)
		return
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.Enabled == nil {
		http.Error(w, "enabled field required", http.StatusBadRequest)
		return
	}
	disabled := s.getDisabledTools()
	if *body.Enabled {
		delete(disabled, name)
	} else {
		disabled[name] = true
	}
	list := make([]string, 0, len(disabled))
	for n := range disabled {
		list = append(list, n)
	}
	sortStrings(list)
	raw, _ := json.Marshal(list)
	if err := s.DB.SetKV(disabledToolsKVKey, string(raw)); err != nil {
		s.Log.Error("set disabled_tools failed", "err", err)
		http.Error(w, "Failed to save", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name":    name,
		"enabled": *body.Enabled,
	})
}

func sortStrings(s []string) {
	sort.Strings(s)
}

func (s *Server) handleAPISocialList(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		s.Log.Warn("social list: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusNotFound)
		return
	}
	channels := social.ListChannelsWithStatus(cfg)
	out := make([]map[string]any, 0, len(channels))
	for _, c := range channels {
		configFields := social.GetChannelConfigSchema(c.Key)
		configValues := social.GetChannelConfigValues(c.Key, cfg)
		out = append(out, map[string]any{
			"name":         c.Key,
			"displayName":  c.DisplayName,
			"enabled":      c.Enabled,
			"ready":        c.Ready,
			"configFields": configFields,
			"config":       configValues,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"channels": out})
}

func (s *Server) handleAPISocialPatch(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "channel name required", http.StatusBadRequest)
		return
	}
	if social.LookupChannel(name) == nil {
		http.Error(w, "unknown channel: "+name, http.StatusNotFound)
		return
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.Enabled == nil {
		http.Error(w, "enabled field required", http.StatusBadRequest)
		return
	}
	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("social patch: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	// Build new socialChannels list
	curSet := make(map[string]bool)
	for _, k := range cfg.SocialChannels {
		curSet[k] = true
	}
	if *body.Enabled {
		curSet[name] = true
	} else {
		delete(curSet, name)
	}
	newList := make([]string, 0, len(curSet))
	for k := range curSet {
		newList = append(newList, k)
	}
	sort.Strings(newList)
	cfg.SocialChannels = newList
	if err := config.Save(cfg); err != nil {
		s.Log.Error("social patch: config save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name":    name,
		"enabled": *body.Enabled,
	})
}

func (s *Server) handleAPISocialConfigPut(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "channel name required", http.StatusBadRequest)
		return
	}
	if social.LookupChannel(name) == nil {
		http.Error(w, "unknown channel: "+name, http.StatusNotFound)
		return
	}
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("social config: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	social.ApplyChannelConfig(cfg, name, updates)

	// Validate before saving: create channel and run HealthCheck
	if err := social.ValidateChannel(r.Context(), cfg, name); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"ok":       false,
			"validated": false,
			"error":    err.Error(),
		})
		return
	}

	if err := config.Save(cfg); err != nil {
		s.Log.Error("social config: config save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	// Auto-enable channel on successful validation
	curSet := make(map[string]bool)
	for _, k := range cfg.SocialChannels {
		curSet[k] = true
	}
	curSet[name] = true
	newList := make([]string, 0, len(curSet))
	for k := range curSet {
		newList = append(newList, k)
	}
	sort.Strings(newList)
	cfg.SocialChannels = newList
	if err := config.Save(cfg); err != nil {
		s.Log.Error("social config: enable save failed", "err", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":        true,
		"validated": true,
		"enabled":   true,
	})
}

func (s *Server) handleAPISoulConfigGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		s.Log.Warn("soul config: load failed", "err", err)
		http.Error(w, "Config not available", http.StatusNotFound)
		return
	}
	out := map[string]any{
		"systemPrompt":         "",
		"personality":          "",
		"tone":                 "",
		"behavioralConstraints": []string{},
	}
	if cfg != nil && cfg.Soul != nil {
		out["systemPrompt"] = cfg.Soul.SystemPrompt
		out["personality"] = cfg.Soul.Personality
		out["tone"] = cfg.Soul.Tone
		if len(cfg.Soul.BehavioralConstraints) > 0 {
			out["behavioralConstraints"] = cfg.Soul.BehavioralConstraints
		}
		if len(cfg.Soul.SystemPromptVersions) > 0 {
			out["systemPromptVersions"] = cfg.Soul.SystemPromptVersions
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) handleAPISoulConfigPut(w http.ResponseWriter, r *http.Request) {
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("soul config: load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		cfg = &types.AutomatonConfig{}
	}
	if cfg.Soul == nil {
		cfg.Soul = &types.SoulConfig{}
	}
	// Merge only fields present in request body (partial PUT); sanitize to prevent injection
	if v, ok := raw["systemPrompt"].(string); ok {
		cfg.Soul.SystemPrompt = sanitizeForStorage(v, maxStoredContentLen)
	}
	if v, ok := raw["personality"].(string); ok {
		cfg.Soul.Personality = sanitizeForStorage(v, 2048)
	}
	if v, ok := raw["tone"].(string); ok {
		cfg.Soul.Tone = sanitizeForStorage(v, 2048)
	}
	if arr, ok := raw["behavioralConstraints"].([]any); ok {
		cfg.Soul.BehavioralConstraints = make([]string, 0, len(arr))
		for _, a := range arr {
			if str, ok := a.(string); ok {
				s := sanitizeForStorage(str, 1024)
				if s != "" {
					cfg.Soul.BehavioralConstraints = append(cfg.Soul.BehavioralConstraints, s)
				}
			}
		}
	}
	if err := config.Save(cfg); err != nil {
		s.Log.Error("soul config: save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const maxSystemPromptVersions = 30
const maxStoredContentLen = 32768 // 32KB max for system prompt / versions

// sanitizeForStorage removes control chars, null bytes, and limits length to prevent
// injection exploits and unsafe content in automaton.json.
func sanitizeForStorage(s string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = maxStoredContentLen
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == 0 {
			continue
		}
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			continue
		}
		if r == unicode.ReplacementChar {
			continue
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if len(out) > maxLen {
		out = out[:maxLen]
	}
	return out
}

const soulEnhancerPrompt = `You are a soul enhancer.
User gives you just a few casual words.
Turn those words into one complete, ready-to-use system prompt.

Make it natural, alive, and powerful — add clear rules, personality, and helpful details so the AI feels real and works great.
Keep the language simple and warm, never fancy or long-winded.
Output ONLY the final system prompt, nothing else.`

func (s *Server) handleAPISoulEnhance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := s.validateJWT(r); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		Words string `json:"words"`
		Apply bool   `json:"apply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	words := sanitizeForStorage(req.Words, 512)
	if words == "" {
		http.Error(w, "words is required", http.StatusBadRequest)
		return
	}
	if len(strings.Fields(words)) < 5 {
		http.Error(w, "words must be at least 5 words", http.StatusBadRequest)
		return
	}
	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("soul enhance: load config failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		http.Error(w, "Config not available", http.StatusServiceUnavailable)
		return
	}
	enhanceClient := inference.BestEnhanceClient(cfg)
	if enhanceClient == nil {
		http.Error(w, "No inference client available for enhancement", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	messages := []inference.ChatMessage{
		{Role: "system", Content: soulEnhancerPrompt},
		{Role: "user", Content: words},
	}
	resp, err := enhanceClient.Chat(ctx, messages, &inference.InferenceOptions{MaxTokens: 2048})
	if err != nil {
		s.Log.Warn("soul enhance inference failed", "err", err)
		http.Error(w, "Enhancement failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	enhanced := sanitizeForStorage(resp.Content, maxStoredContentLen)
	if enhanced == "" {
		http.Error(w, "Empty response from inference", http.StatusInternalServerError)
		return
	}
	if req.Apply {
		if cfg == nil {
			cfg = &types.AutomatonConfig{}
		}
		if cfg.Soul == nil {
			cfg.Soul = &types.SoulConfig{}
		}
		cfg.Soul.SystemPrompt = enhanced
		versions := append([]string{enhanced}, cfg.Soul.SystemPromptVersions...)
		if len(versions) > maxSystemPromptVersions {
			versions = versions[:maxSystemPromptVersions]
		}
		cfg.Soul.SystemPromptVersions = versions
		if err := config.Save(cfg); err != nil {
			s.Log.Error("soul enhance: save failed", "err", err)
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"systemPrompt": enhanced,
	})
}

func (s *Server) validateJWT(r *http.Request) (string, bool) {
	secret := ""
	if s.Cfg != nil && s.Cfg.JWTSecret != "" {
		secret = s.Cfg.JWTSecret
	}
	if secret == "" {
		return "", false
	}
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return "", false
	}
	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	token, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}
	addr, _ := claims["address"].(string)
	return addr, addr != ""
}

// knownEVMChains for USDC balance checks (identity address_<chain> keys).
var knownEVMChains = []string{"eip155:8453", "eip155:84532", "eip155:1", "eip155:137", "eip155:42161"}

func (s *Server) handleAPIEconomicGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		s.Log.Warn("economic: load config failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		cfg = &types.AutomatonConfig{}
	}

	// Collect addresses: config, identity table, children
	type addrEntry struct {
		Address string
		Chain   string
		Source  string
	}
	seen := make(map[string]bool)
	var entries []addrEntry
	add := func(addr, chain, source string) {
		if addr == "" || !strings.HasPrefix(addr, "0x") {
			return
		}
		key := strings.ToLower(addr) + "|" + chain
		if seen[key] {
			return
		}
		seen[key] = true
		entries = append(entries, addrEntry{Address: addr, Chain: chain, Source: source})
	}

	chains := knownEVMChains
	var providers map[string]conway.USDCChainProvider
	if len(cfg.ChainProviders) > 0 {
		chains = make([]string, 0, len(cfg.ChainProviders))
		providers = make(map[string]conway.USDCChainProvider)
		for ch, cp := range cfg.ChainProviders {
			if cp.RPCURL != "" && cp.USDCAddress != "" {
				chains = append(chains, ch)
				providers[ch] = conway.USDCChainProvider{RPCURL: cp.RPCURL, USDCAddress: cp.USDCAddress}
			}
		}
	}
	defaultChain := cfg.DefaultChain
	if defaultChain == "" {
		defaultChain = identity.DefaultChainBase
	}

	// Config addresses
	if cfg.WalletAddress != "" {
		add(cfg.WalletAddress, defaultChain, "wallet")
	}
	if cfg.CreatorAddress != "" && cfg.CreatorAddress != cfg.WalletAddress {
		add(cfg.CreatorAddress, defaultChain, "creator")
	}

	// Identity table
	if s.DB != nil {
		if a, ok, _ := s.DB.GetIdentity("address"); ok && a != "" {
			add(a, defaultChain, "identity")
		}
		for _, ch := range chains {
			if a, ok, _ := s.DB.GetIdentity(identity.AddressKeyForChain(ch)); ok && a != "" {
				add(a, ch, "identity_"+ch)
			}
		}
	}

	// Children
	if db, ok := s.DB.(*state.Database); ok {
		children, _ := db.GetAllChildren()
		for _, c := range children {
			if c.Address != "" {
				ch := c.Chain
				if ch == "" {
					ch = defaultChain
				}
				add(c.Address, ch, "child")
			}
		}
	}

	// Fetch USDC balances per (address, chain)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	balances := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		if !identity.IsEVM(e.Chain) {
			continue
		}
		results, err := conway.GetUSDCBalanceMulti(ctx, e.Address, []string{e.Chain}, providers)
		if err != nil {
			balances = append(balances, map[string]any{
				"address": e.Address, "chain": e.Chain, "source": e.Source,
				"balance": nil, "error": err.Error(),
			})
			continue
		}
		bal := 0.0
		if len(results) > 0 {
			bal = results[0].Balance
		}
		balances = append(balances, map[string]any{
			"address": e.Address, "chain": e.Chain, "source": e.Source,
			"balance": bal,
		})
	}

	tp := cfg.TreasuryPolicy
	if tp == nil {
		tp = &types.TreasuryPolicy{}
		*tp = types.DefaultTreasuryPolicy()
	}
	resourceMode := cfg.ResourceConstraintMode
	if resourceMode == "" {
		resourceMode = "auto"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"addresses":            entries,
		"balances":             balances,
		"treasuryPolicy":       tp,
		"resourceConstraintMode": resourceMode,
	})
}

func (s *Server) handleAPIEconomicPut(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.validateJWT(r); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("economic put: load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		cfg = &types.AutomatonConfig{}
	}
	if cfg.TreasuryPolicy == nil {
		tp := types.DefaultTreasuryPolicy()
		cfg.TreasuryPolicy = &tp
	}

	if v, ok := raw["resourceConstraintMode"].(string); ok && (v == "auto" || v == "forced_on" || v == "forced_off") {
		cfg.ResourceConstraintMode = v
	}
	if tp, ok := raw["treasuryPolicy"].(map[string]any); ok {
		cfg.TreasuryPolicy = mergeTreasuryPolicyFromMap(cfg.TreasuryPolicy, tp)
	}
	if err := config.Save(cfg); err != nil {
		s.Log.Error("economic put: save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mergeTreasuryPolicyFromMap(base *types.TreasuryPolicy, over map[string]any) *types.TreasuryPolicy {
	if base == nil {
		tp := types.DefaultTreasuryPolicy()
		base = &tp
	}
	out := *base
	if v, ok := over["maxSingleTransferCents"].(float64); ok && v >= 0 {
		out.MaxSingleTransferCents = int(v)
	}
	if v, ok := over["maxHourlyTransferCents"].(float64); ok && v >= 0 {
		out.MaxHourlyTransferCents = int(v)
	}
	if v, ok := over["maxDailyTransferCents"].(float64); ok && v >= 0 {
		out.MaxDailyTransferCents = int(v)
	}
	if v, ok := over["minReserveCents"].(float64); ok && v >= 0 {
		out.MinReserveCents = int(v)
	}
	if v, ok := over["inferenceDailyBudgetCents"].(float64); ok && v >= 0 {
		out.InferenceDailyBudgetCents = int(v)
	}
	if arr, ok := over["x402AllowedDomains"].([]any); ok {
		out.X402AllowedDomains = make([]string, 0, len(arr))
		for _, a := range arr {
			if s, ok := a.(string); ok && s != "" {
				out.X402AllowedDomains = append(out.X402AllowedDomains, s)
			}
		}
	}
	return &out
}

// maskAPIKey returns a masked version of an API key for API responses.
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "••••••••"
	}
	return key[:4] + "••••••••" + key[len(key)-4:]
}

func (s *Server) handleAPIModelsList(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		s.Log.Warn("models list: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		cfg = &types.AutomatonConfig{}
	}
	models := make([]types.LLMModelEntry, len(cfg.Models))
	copy(models, cfg.Models)
	sort.Slice(models, func(i, j int) bool { return models[i].Priority < models[j].Priority })

	out := make([]map[string]any, 0, len(models))
	for _, m := range models {
		apiKey := ""
		if m.APIKey != "" {
			apiKey = maskAPIKey(m.APIKey)
		}
		out = append(out, map[string]any{
			"id":           m.ID,
			"provider":     m.Provider,
			"modelId":      m.ModelID,
			"apiKeyMasked": apiKey,
			"contextLimit": m.ContextLimit,
			"costCapCents": m.CostCapCents,
			"priority":     m.Priority,
			"enabled":      m.Enabled,
		})
	}

	providers := inference.ListProviders()
	providerList := make([]map[string]any, 0, len(providers))
	for _, p := range providers {
		providerList = append(providerList, map[string]any{
			"key":         p.Key,
			"displayName": p.DisplayName,
			"local":       p.Local,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"models":    out,
		"providers": providerList,
	})
}

func (s *Server) handleAPIModelsPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Provider     string `json:"provider"`
		ModelID      string `json:"modelId"`
		APIKey       string `json:"apiKey,omitempty"`
		ContextLimit int    `json:"contextLimit,omitempty"`
		CostCapCents int    `json:"costCapCents,omitempty"`
		Enabled      *bool  `json:"enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.Provider == "" || body.ModelID == "" {
		http.Error(w, "provider and modelId are required", http.StatusBadRequest)
		return
	}
	if inference.LookupProvider(body.Provider) == nil {
		http.Error(w, "unknown provider: "+body.Provider, http.StatusBadRequest)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("models post: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		cfg = &types.AutomatonConfig{}
	}

	id := body.Provider + "_" + body.ModelID
	for _, m := range cfg.Models {
		if m.ID == id {
			http.Error(w, "model already exists: "+id, http.StatusConflict)
			return
		}
	}

	priority := len(cfg.Models)
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	ent := types.LLMModelEntry{
		ID:           id,
		Provider:     body.Provider,
		ModelID:      body.ModelID,
		APIKey:       body.APIKey,
		ContextLimit: body.ContextLimit,
		CostCapCents: body.CostCapCents,
		Priority:     priority,
		Enabled:      enabled,
	}
	cfg.Models = append(cfg.Models, ent)

	if err := config.Save(cfg); err != nil {
		s.Log.Error("models post: config save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	apiKey := ""
	if ent.APIKey != "" {
		apiKey = maskAPIKey(ent.APIKey)
	}
	json.NewEncoder(w).Encode(map[string]any{
		"id":           ent.ID,
		"provider":     ent.Provider,
		"modelId":      ent.ModelID,
		"apiKeyMasked": apiKey,
		"contextLimit": ent.ContextLimit,
		"costCapCents": ent.CostCapCents,
		"priority":     ent.Priority,
		"enabled":      ent.Enabled,
	})
}

func (s *Server) handleAPIModelsPatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "model id required", http.StatusBadRequest)
		return
	}
	var body struct {
		APIKey       *string `json:"apiKey,omitempty"`
		ModelID      *string `json:"modelId,omitempty"`
		ContextLimit *int    `json:"contextLimit,omitempty"`
		CostCapCents *int    `json:"costCapCents,omitempty"`
		Enabled      *bool   `json:"enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("models patch: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	var found *types.LLMModelEntry
	for i := range cfg.Models {
		if cfg.Models[i].ID == id {
			found = &cfg.Models[i]
			break
		}
	}
	if found == nil {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	if body.APIKey != nil {
		found.APIKey = *body.APIKey
	}
	if body.ModelID != nil {
		found.ModelID = *body.ModelID
	}
	if body.ContextLimit != nil {
		found.ContextLimit = *body.ContextLimit
	}
	if body.CostCapCents != nil {
		found.CostCapCents = *body.CostCapCents
	}
	if body.Enabled != nil {
		found.Enabled = *body.Enabled
	}

	if err := config.Save(cfg); err != nil {
		s.Log.Error("models patch: config save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	apiKey := ""
	if found.APIKey != "" {
		apiKey = maskAPIKey(found.APIKey)
	}
	json.NewEncoder(w).Encode(map[string]any{
		"id":           found.ID,
		"provider":     found.Provider,
		"modelId":      found.ModelID,
		"apiKeyMasked": apiKey,
		"contextLimit": found.ContextLimit,
		"costCapCents": found.CostCapCents,
		"priority":     found.Priority,
		"enabled":      found.Enabled,
	})
}

func (s *Server) handleAPIModelsDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "model id required", http.StatusBadRequest)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("models delete: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	newModels := make([]types.LLMModelEntry, 0, len(cfg.Models))
	found := false
	for _, m := range cfg.Models {
		if m.ID != id {
			newModels = append(newModels, m)
		} else {
			found = true
		}
	}
	if !found {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	for i := range newModels {
		newModels[i].Priority = i
	}
	cfg.Models = newModels

	if err := config.Save(cfg); err != nil {
		s.Log.Error("models delete: config save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIModelsOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(body.IDs) == 0 {
		http.Error(w, "ids array is required", http.StatusBadRequest)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		s.Log.Error("models order: config load failed", "err", err)
		http.Error(w, "Config not available", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		cfg = &types.AutomatonConfig{}
	}

	byID := make(map[string]types.LLMModelEntry)
	for _, m := range cfg.Models {
		byID[m.ID] = m
	}
	newModels := make([]types.LLMModelEntry, 0, len(body.IDs))
	consumed := make(map[string]bool)
	for _, id := range body.IDs {
		consumed[id] = true
		m, ok := byID[id]
		if !ok {
			http.Error(w, "unknown model id: "+id, http.StatusBadRequest)
			return
		}
		m.Priority = len(newModels)
		newModels = append(newModels, m)
	}
	for _, m := range cfg.Models {
		if !consumed[m.ID] {
			m.Priority = len(newModels)
			newModels = append(newModels, m)
		}
	}
	cfg.Models = newModels

	if err := config.Save(cfg); err != nil {
		s.Log.Error("models order: config save failed", "err", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateState updates runtime state from the agent loop.
func (rs *RuntimeState) UpdateState(running bool, agentState string, tickNum int64) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.Running = running
	rs.AgentState = agentState
	rs.TickNum = tickNum
}

// IsPaused returns whether the agent is paused.
func (rs *RuntimeState) IsPaused() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.Paused
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	s.httpServer = &http.Server{
		Addr:         s.Addr,
		Handler:      s,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.Log.Info("web dashboard listening", "addr", s.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server, closing all connections.
// Call this when the backend is shutting down to ensure the web server stops cleanly.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
