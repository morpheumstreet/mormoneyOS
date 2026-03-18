// Package adapter provides real implementations of marketplace ports for Phase 2.
package adapter

import (
	"context"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
	"github.com/morpheumlabs/mormoneyos-go/internal/skills"
	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// SkillsStore provides installed skills for ListMySkills.
type SkillsStore interface {
	GetAllSkills() ([]state.SkillRow, error)
	GetSkillByName(name string) (*state.SkillRow, error)
}

// RegistryAdapter implements RegistryPort using ClawHub registry + local DB.
type RegistryAdapter struct {
	Client *skills.RegistryClient
	Store  SkillsStore
	Config *types.SkillsConfig
}

var _ port.RegistryPort = (*RegistryAdapter)(nil)

// NewRegistryAdapter creates a registry adapter with real ClawHub + DB.
func NewRegistryAdapter(client *skills.RegistryClient, store SkillsStore, cfg *types.SkillsConfig) *RegistryAdapter {
	return &RegistryAdapter{Client: client, Store: store, Config: cfg}
}

// Search returns skills from ClawHub matching query; filter applied client-side.
func (a *RegistryAdapter) Search(query, filter string) ([]entity.Skill, error) {
	if a.Client == nil {
		return nil, nil
	}
	ctx := context.Background()
	limit := 20
	results, err := a.Client.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	out := make([]entity.Skill, 0, len(results))
	for _, r := range results {
		s := searchResultToSkill(r)
		if matchesFilter(s, filter) {
			out = append(out, s)
		}
	}
	return out, nil
}

// GetByID returns a skill by slug (ClawHub) or name (local). Tries registry first, then local.
func (a *RegistryAdapter) GetByID(id string) (*entity.Skill, error) {
	if id == "" {
		return nil, nil
	}
	// Try ClawHub registry first
	if a.Client != nil {
		ctx := context.Background()
		meta, _, err := a.Client.Resolve(ctx, id)
		if err == nil && meta != nil {
			s := registryMetaToSkill(meta)
			return &s, nil
		}
	}
	// Fallback: local installed skill by name
	if a.Store != nil {
		row, err := a.Store.GetSkillByName(id)
		if err == nil && row != nil {
			s := skillRowToSkill(row)
			return &s, nil
		}
	}
	return nil, nil
}

// GetOffers returns empty; offers are on-chain (Phase 3).
func (a *RegistryAdapter) GetOffers(skillID string) ([]entity.Offer, error) {
	return []entity.Offer{}, nil
}

// ListMySkills returns installed skills from local DB.
func (a *RegistryAdapter) ListMySkills() ([]entity.Skill, error) {
	if a.Store == nil {
		return []entity.Skill{}, nil
	}
	rows, err := a.Store.GetAllSkills()
	if err != nil {
		return nil, err
	}
	out := make([]entity.Skill, 0, len(rows))
	for _, r := range rows {
		s := skillRowToSkill(&r)
		out = append(out, s)
	}
	return out, nil
}

func searchResultToSkill(r skills.SearchResult) entity.Skill {
	hash := ""
	if r.Version != nil {
		hash = *r.Version
	}
	return entity.Skill{
		ID:           r.Slug,
		Name:         r.DisplayName,
		Description:  r.Summary,
		PriceMORM:    0,
		Badges:       []string{"ClawHub"},
		Permissions:  nil,
		SecurityHash: hash,
		PerpReady:    false,
	}
}

func registryMetaToSkill(m *skills.RegistryMeta) entity.Skill {
	hash := ""
	if m.LatestVersion != nil {
		hash = m.LatestVersion.Version
	}
	return entity.Skill{
		ID:           m.Slug,
		Name:         m.DisplayName,
		Description:  m.Summary,
		PriceMORM:    0,
		Badges:       []string{"ClawHub"},
		Permissions:  nil,
		SecurityHash: hash,
		PerpReady:    false,
	}
}

func skillRowToSkill(r *state.SkillRow) entity.Skill {
	badges := []string{"Installed"}
	if r.Source == "registry" {
		badges = append(badges, "ClawHub")
	}
	return entity.Skill{
		ID:           r.Name,
		Name:         r.Name,
		Description:  r.Description,
		PriceMORM:    0,
		Badges:       badges,
		Permissions:  nil,
		SecurityHash: "",
		PerpReady:    false,
	}
}

func matchesFilter(s entity.Skill, filter string) bool {
	if filter == "" {
		return true
	}
	f := strings.ToLower(filter)
	switch f {
	case "perp_ready", "perp-ready":
		return s.PerpReady
	case "clawhub", "registry":
		for _, b := range s.Badges {
			if strings.EqualFold(b, "ClawHub") {
				return true
			}
		}
		return false
	default:
		return true
	}
}
