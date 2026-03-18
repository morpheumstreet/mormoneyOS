// Package adapter provides stub implementations of marketplace ports for Phase 1.
package adapter

import "github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"

// StubScanner implements ScannerPort with no-op behavior.
// Phase 2 will replace with real mwvm/static scanner.
type StubScanner struct{}

var _ port.ScannerPort = (*StubScanner)(nil)

// GetBadges returns empty badges for Phase 1 stub.
func (s *StubScanner) GetBadges(securityHash string) []string {
	return nil
}

// GetReport returns a placeholder message for Phase 1 stub.
func (s *StubScanner) GetReport(hash string) (string, error) {
	return "Security report: Phase 2 will wire mwvm + static scanner.", nil
}
