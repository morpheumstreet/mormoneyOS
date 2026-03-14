package web

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

// HeartbeatScheduleAPI provides DB access for heartbeat schedule API.
type HeartbeatScheduleAPI interface {
	GetHeartbeatSchedule() ([]state.HeartbeatScheduleRow, error)
	SetHeartbeatEnabled(name string, enabled bool) error
	SetHeartbeatSchedule(name string, schedule string) error
}

func (s *Server) handleAPIHeartbeatList(w http.ResponseWriter, r *http.Request) {
	hb, ok := s.DB.(HeartbeatScheduleAPI)
	if !ok {
		http.Error(w, "Heartbeat API not available", http.StatusNotFound)
		return
	}
	rows, err := hb.GetHeartbeatSchedule()
	if err != nil {
		s.Log.Error("heartbeat list failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"name":         r.Name,
			"schedule":     r.Schedule,
			"task":         r.Task,
			"enabled":      r.Enabled == 1,
			"tierMinimum":  r.TierMinimum,
			"lastRun":      r.LastRun,
			"nextRun":      r.NextRun,
			"leaseUntil":   r.LeaseUntil,
			"leaseOwner":   r.LeaseOwner,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"schedules": out})
}

func (s *Server) handleAPIHeartbeatPatch(w http.ResponseWriter, r *http.Request) {
	hb, ok := s.DB.(HeartbeatScheduleAPI)
	if !ok {
		http.Error(w, "Heartbeat API not available", http.StatusNotFound)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
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
	if err := hb.SetHeartbeatEnabled(name, *body.Enabled); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "heartbeat schedule not found: "+name, http.StatusNotFound)
			return
		}
		s.Log.Error("heartbeat patch failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"name": name, "enabled": *body.Enabled})
}

func (s *Server) handleAPIHeartbeatSchedulePatch(w http.ResponseWriter, r *http.Request) {
	hb, ok := s.DB.(HeartbeatScheduleAPI)
	if !ok {
		http.Error(w, "Heartbeat API not available", http.StatusNotFound)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	var body struct {
		Schedule string `json:"schedule"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.Schedule == "" {
		http.Error(w, "schedule field required", http.StatusBadRequest)
		return
	}
	if err := hb.SetHeartbeatSchedule(name, body.Schedule); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "heartbeat schedule not found: "+name, http.StatusNotFound)
			return
		}
		s.Log.Error("heartbeat schedule patch failed", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"name": name, "schedule": body.Schedule})
}
