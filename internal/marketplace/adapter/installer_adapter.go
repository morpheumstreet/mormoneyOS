// Package adapter provides real implementations of marketplace ports for Phase 2.
package adapter

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
	"github.com/morpheumlabs/mormoneyos-go/internal/skills"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
)

// InstallerAdapter implements InstallerPort using skills.InstallFromRegistry.
type InstallerAdapter struct {
	Client *skills.RegistryClient
	Store  skills.SkillInserter
	Config *types.SkillsConfig
}

var _ port.InstallerPort = (*InstallerAdapter)(nil)

// NewInstallerAdapter creates an installer adapter.
func NewInstallerAdapter(client *skills.RegistryClient, store skills.SkillInserter, cfg *types.SkillsConfig) *InstallerAdapter {
	return &InstallerAdapter{Client: client, Store: store, Config: cfg}
}

// Install fetches and installs a skill from the registry.
func (a *InstallerAdapter) Install(ctx context.Context, skillID string, name, desc string) (skillRoot string, err error) {
	if a.Client == nil || a.Store == nil {
		return "", nil
	}
	cfg := a.Config
	if cfg == nil {
		cfg = &types.SkillsConfig{}
	}
	root, _, err := skills.InstallFromRegistry(ctx, a.Client, a.Store, cfg, skillID, "", name, desc)
	return root, err
}
