package marketplace

import (
	"context"
	"fmt"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// SecurityReportTool implements mormaegis.security_report.
type SecurityReportTool struct {
	UseCase *usecase.SecurityReport
}

var _ tools.Tool = (*SecurityReportTool)(nil)

func (t *SecurityReportTool) Name() string { return "mormaegis.security_report" }
func (t *SecurityReportTool) Description() string {
	return "View full static + mwvm scanner result"
}
func (t *SecurityReportTool) Parameters() string {
	return `{"type":"object","properties":{"hash":{"type":"string"}},"required":["hash"]}`
}

func (t *SecurityReportTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	hash, _ := args["hash"].(string)
	if hash == "" {
		return "", fmt.Errorf("hash is required")
	}
	return t.UseCase.Execute(ctx, hash)
}
