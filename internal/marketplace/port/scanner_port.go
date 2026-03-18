// Package port defines interfaces for Dependency Inversion.
package port

// ScannerPort provides security scanning and badge resolution.
type ScannerPort interface {
	GetBadges(securityHash string) []string
	GetReport(hash string) (string, error)
}
