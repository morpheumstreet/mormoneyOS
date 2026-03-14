package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/morpheumlabs/mormoneyos-go/internal/config"
	"github.com/morpheumlabs/mormoneyos-go/internal/skills"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// SkillsAPIStore provides skills CRUD for the API.
type SkillsAPIStore interface {
	GetAllSkills() ([]state.SkillRow, error)
	GetSkillByName(name string) (*state.SkillRow, error)
	InsertSkill(name, description, instructions, source, path string, enabled bool) error
	DeleteSkill(name string) error
	UpdateSkillEnabled(name string, enabled bool) error
	UpdateSkill(name, description, instructions string) error
}

// skillsAPIRecommendedSlugs is a curated list of recommended skills from ClawHub.
var skillsAPIRecommendedSlugs = []string{
	"gmail-secretary", "calendar", "market-global-snapshot", "codex-hook",
	"openclaw-quick-start", "bailian-studio", "ai-cost-calculator",
}

// --- Helpers (DRY, Single Responsibility) ---

func (s *Server) getSkillsConfig() *types.SkillsConfig {
	if s.Cfg != nil && s.Cfg.SkillsConfigGetter != nil {
		if cfg := s.Cfg.SkillsConfigGetter(); cfg != nil {
			return cfg
		}
	}
	if cfg, err := config.Load(); err == nil && cfg.Skills != nil {
		return cfg.Skills
	}
	return nil
}

func (s *Server) getRegistryClient() *skills.RegistryClient {
	cfg := s.getSkillsConfig()
	regURL, timeoutSec := skills.RegistryConfigFrom(cfg)
	return skills.NewRegistryClient(regURL, timeoutSec)
}

func isTrustedSource(source string) bool {
	return source == "registry" || source == "builtin"
}

func skillRowToMap(row *state.SkillRow) map[string]any {
	if row == nil {
		return nil
	}
	return map[string]any{
		"name":          row.Name,
		"description":   row.Description,
		"instructions":  row.Instructions,
		"source":        row.Source,
		"path":          row.Path,
		"enabled":       row.Enabled,
		"trusted":       isTrustedSource(row.Source),
		"auto_activate": row.AutoActivate,
	}
}

func writeJSONError(w http.ResponseWriter, err string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err})
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// requireSkillsStore returns (store, true) if ok; else writes error and returns (nil, false).
func (s *Server) requireSkillsStore(w http.ResponseWriter) (SkillsAPIStore, bool) {
	store, ok := s.DB.(SkillsAPIStore)
	if !ok {
		writeJSONError(w, "skills API not available", http.StatusServiceUnavailable)
		return nil, false
	}
	return store, true
}

// --- Handlers ---

func (s *Server) handleAPISkillsList(w http.ResponseWriter, r *http.Request) {
	store, ok := s.requireSkillsStore(w)
	if !ok {
		return
	}
	all, err := store.GetAllSkills()
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filter := r.URL.Query().Get("filter")
	trusted := r.URL.Query().Get("trusted")
	var out []map[string]any
	for _, row := range all {
		if filter == "enabled" && !row.Enabled {
			continue
		}
		if filter == "disabled" && row.Enabled {
			continue
		}
		trustedVal := isTrustedSource(row.Source)
		if trusted == "trusted" && !trustedVal {
			continue
		}
		if trusted == "untrusted" && trustedVal {
			continue
		}
		m := skillRowToMap(&row)
		out = append(out, m)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"skills": out})
}

func (s *Server) handleAPISkillsGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, "name required", http.StatusBadRequest)
		return
	}
	store, ok := s.requireSkillsStore(w)
	if !ok {
		return
	}
	row, err := store.GetSkillByName(name)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row == nil {
		writeJSONError(w, "skill not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skillRowToMap(row))
}

func (s *Server) handleAPISkillsPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	store, ok := s.requireSkillsStore(w)
	if !ok {
		return
	}
	var body struct {
		Source      string `json:"source"`
		ID          string `json:"id"`
		Version     string `json:"version"`
		Name        string `json:"name"`
		Path        string `json:"path"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	// Install from ClawHub (single path via skills.InstallFromRegistry)
	if body.Source == "clawhub" || (body.Source == "" && body.ID != "" && body.Path == "") {
		client := s.getRegistryClient()
		cfg := s.getSkillsConfig()
		skillRoot, skillName, err := skills.InstallFromRegistry(r.Context(), client, store, cfg, body.ID, body.Version, body.Name, body.Description)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"name":    skillName,
			"source":  "registry",
			"path":    skillRoot,
			"enabled": true,
		})
		return
	}
	// Install from path
	if body.Path == "" || body.Name == "" {
		writeJSONError(w, "name and path required for path install", http.StatusBadRequest)
		return
	}
	if err := store.InsertSkill(body.Name, body.Description, "", "installed", body.Path, true); err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"name":    body.Name,
		"source":  "installed",
		"path":    body.Path,
		"enabled": true,
	})
}

func (s *Server) handleAPISkillsPatch(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, "name required", http.StatusBadRequest)
		return
	}
	store, ok := s.requireSkillsStore(w)
	if !ok {
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if enabled, ok := body["enabled"].(bool); ok {
		if err := store.UpdateSkillEnabled(name, enabled); err != nil {
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if desc, ok := body["description"].(string); ok {
		instructions, _ := body["instructions"].(string)
		if err := store.UpdateSkill(name, desc, instructions); err != nil {
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	row, _ := store.GetSkillByName(name)
	if row == nil {
		writeJSONError(w, "skill not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name":        row.Name,
		"description": row.Description,
		"enabled":     row.Enabled,
	})
}

func (s *Server) handleAPISkillsDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, "name required", http.StatusBadRequest)
		return
	}
	store, ok := s.requireSkillsStore(w)
	if !ok {
		return
	}
	if err := store.DeleteSkill(name); err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPISkillsActivate(w http.ResponseWriter, r *http.Request) {
	s.handleAPISkillsSetEnabled(w, r, true)
}

func (s *Server) handleAPISkillsDeactivate(w http.ResponseWriter, r *http.Request) {
	s.handleAPISkillsSetEnabled(w, r, false)
}

func (s *Server) handleAPISkillsSetEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, "name required", http.StatusBadRequest)
		return
	}
	store, ok := s.requireSkillsStore(w)
	if !ok {
		return
	}
	if err := store.UpdateSkillEnabled(name, enabled); err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"name": name, "enabled": enabled})
}

func (s *Server) handleAPISkillsDiscovery(w http.ResponseWriter, r *http.Request) {
	client := s.getRegistryClient()
	query := r.URL.Query().Get("q")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := parseInt(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if query != "" {
		results, err := client.Search(r.Context(), query, limit)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"results": results})
		return
	}
	cursor := r.URL.Query().Get("cursor")
	resp, err := client.List(r.Context(), cursor, limit)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items":      resp.Items,
		"nextCursor": resp.NextCursor,
	})
}

func (s *Server) handleAPISkillsRecommended(w http.ResponseWriter, r *http.Request) {
	client := s.getRegistryClient()
	installed := s.installedSkillNames()
	var out []map[string]any
	for _, slug := range skillsAPIRecommendedSlugs {
		meta, version, err := client.Resolve(r.Context(), slug)
		if err != nil {
			continue
		}
		v := ""
		if meta.LatestVersion != nil {
			v = meta.LatestVersion.Version
		}
		if version != "" {
			v = version
		}
		out = append(out, map[string]any{
			"slug":        meta.Slug,
			"displayName": meta.DisplayName,
			"summary":     meta.Summary,
			"version":     v,
			"installed":   installed[meta.Slug] || installed[meta.DisplayName],
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"recommended": out})
}

func (s *Server) installedSkillNames() map[string]bool {
	out := make(map[string]bool)
	if store, ok := s.DB.(SkillsAPIStore); ok {
		all, _ := store.GetAllSkills()
		for _, row := range all {
			out[row.Name] = true
		}
	}
	return out
}
