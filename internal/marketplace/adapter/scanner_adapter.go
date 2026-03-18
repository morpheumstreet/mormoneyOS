// Package adapter provides real implementations of marketplace ports for Phase 2.
package adapter

import (
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// ScannerAdapter implements ScannerPort. Phase 2: returns ClawHub badge for registry skills.
// Phase 3 will wire mwvm/static scanner for hash-based verification.
type ScannerAdapter struct{}

var _ port.ScannerPort = (*ScannerAdapter)(nil)

// NewScannerAdapter creates a scanner adapter.
func NewScannerAdapter() *ScannerAdapter {
	return &ScannerAdapter{}
}

// GetBadges returns badges for a security hash. Phase 2: ClawHub skills get "ClawHub" badge.
// When hash is non-empty, also returns "Verified" as placeholder for Phase 3 mwvm integration.
func (s *ScannerAdapter) GetBadges(securityHash string) []string {
	if securityHash == "" {
		return []string{"ClawHub"}
	}
	return []string{"ClawHub", "Verified"}
}

// GetReport returns a security report. Phase 2: placeholder; Phase 3 wires mwvm + static scanner.
func (s *ScannerAdapter) GetReport(hash string) (string, error) {
	if hash == "" {
		return "No hash provided. Phase 3 will wire mwvm + static scanner for full reports.", nil
	}
	preview := hash
	if len(hash) > 12 {
		preview = hash[:12] + "..."
	}
	return "Security report: Phase 3 will wire mwvm + static scanner. Hash: " + preview, nil
}
