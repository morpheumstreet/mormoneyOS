// Package port defines interfaces for Dependency Inversion.
package port

import "context"

// InstallerPort installs skills from the registry into local store.
type InstallerPort interface {
	Install(ctx context.Context, skillID string, name, desc string) (skillRoot string, err error)
}
