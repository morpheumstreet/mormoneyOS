// Package web provides REST API handlers for the Mormaegis marketplace.
// Reuses the same usecases as MCP tools (DRY).
package web

import (
	"encoding/json"
	"net/http"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace"
)

// handleAPIMarketplaceSearch handles GET /api/marketplace/search?q=...&filter=...
func (s *Server) handleAPIMarketplaceSearch(w http.ResponseWriter, r *http.Request) {
	svc := s.getMarketplaceService()
	if svc == nil {
		writeJSONError(w, "marketplace not configured", http.StatusServiceUnavailable)
		return
	}
	query := r.URL.Query().Get("q")
	filter := r.URL.Query().Get("filter")
	skills, err := svc.SearchSkills.Execute(r.Context(), query, filter)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"skills": skills})
}

// handleAPIMarketplaceSkill handles GET /api/marketplace/skills/{id}
func (s *Server) handleAPIMarketplaceSkill(w http.ResponseWriter, r *http.Request) {
	svc := s.getMarketplaceService()
	if svc == nil {
		writeJSONError(w, "marketplace not configured", http.StatusServiceUnavailable)
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, "skill id required", http.StatusBadRequest)
		return
	}
	skill, err := svc.GetSkill.Execute(r.Context(), id)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if skill == nil {
		writeJSONError(w, "skill not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"skill": skill})
}

// handleAPIMarketplaceInstall handles POST /api/marketplace/install
func (s *Server) handleAPIMarketplaceInstall(w http.ResponseWriter, r *http.Request) {
	svc := s.getMarketplaceService()
	if svc == nil {
		writeJSONError(w, "marketplace not configured", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		SkillID      string `json:"skill_id"`
		AgentCardSig string `json:"agent_card_sig"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	result, err := svc.InstallSkill.Execute(r.Context(), body.SkillID, body.AgentCardSig)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"result": result})
}

// handleAPIMarketplaceMySkills handles GET /api/marketplace/my-skills
func (s *Server) handleAPIMarketplaceMySkills(w http.ResponseWriter, r *http.Request) {
	svc := s.getMarketplaceService()
	if svc == nil {
		writeJSONError(w, "marketplace not configured", http.StatusServiceUnavailable)
		return
	}
	skills, err := svc.MySkills.Execute(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"skills": skills})
}

// handleAPIMarketplaceSecurityReport handles GET /api/marketplace/security-report?hash=...
func (s *Server) handleAPIMarketplaceSecurityReport(w http.ResponseWriter, r *http.Request) {
	svc := s.getMarketplaceService()
	if svc == nil {
		writeJSONError(w, "marketplace not configured", http.StatusServiceUnavailable)
		return
	}
	hash := r.URL.Query().Get("hash")
	if hash == "" {
		writeJSONError(w, "hash required", http.StatusBadRequest)
		return
	}
	report, err := svc.SecurityReport.Execute(r.Context(), hash)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"report": report})
}

// getMarketplaceService returns the marketplace service when configured.
func (s *Server) getMarketplaceService() *marketplace.Service {
	if s.Cfg == nil {
		return nil
	}
	return s.Cfg.MarketplaceService
}
