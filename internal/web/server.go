package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/morpheumlabs/mormoneyos-go/internal/identity"
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

// ServerConfig holds config for status API (TS-aligned).
type ServerConfig struct {
	Name          string
	WalletAddress string
	DefaultChain  string // CAIP-2, e.g. eip155:8453
	Version       string
	CreditsGetter CreditsGetter
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
	Addr   string
	State  *RuntimeState
	DB     WebDB
	Cfg    *ServerConfig
	Log    *slog.Logger
	mux    *http.ServeMux
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
	msg := strings.ToLower(strings.TrimSpace(payload.Message))

	if strings.Contains(msg, "status") && s.DB != nil {
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
	if strings.Contains(msg, "help") || strings.Contains(msg, "帮助") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"response": "I can help with:\n1. status — show agent state\n2. strategies — list skills/children\n3. cost — LLM cost summary\n\nTry: 'status' or 'strategies'",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"response": fmt.Sprintf("I don't understand %q. Type 'help' for commands.", payload.Message),
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
	srv := &http.Server{
		Addr:         s.Addr,
		Handler:      s,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.Log.Info("web dashboard listening", "addr", s.Addr)
	return srv.ListenAndServe()
}
