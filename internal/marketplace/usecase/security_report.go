package usecase

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/port"
)

// SecurityReport retrieves full static + mwvm scanner result for a hash.
type SecurityReport struct {
	Scanner port.ScannerPort
}

// Execute returns the security report. Phase 1 uses stub scanner.
func (u *SecurityReport) Execute(ctx context.Context, hash string) (string, error) {
	if u.Scanner == nil {
		return "Scanner not configured.", nil
	}
	return u.Scanner.GetReport(hash)
}
