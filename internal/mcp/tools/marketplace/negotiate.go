package marketplace

import (
	"context"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/dto"
	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/usecase"
	"github.com/morpheumlabs/mormoneyos-go/internal/tools"
)

// NegotiateTool implements mormaegis.negotiate.
type NegotiateTool struct {
	UseCase *usecase.NegotiateOffer
}

var _ tools.Tool = (*NegotiateTool)(nil)

func (t *NegotiateTool) Name() string { return "mormaegis.negotiate" }
func (t *NegotiateTool) Description() string {
	return "Post or counter-offer on a skill"
}
func (t *NegotiateTool) Parameters() string {
	return `{"type":"object","properties":{"offer_id":{"type":"string"},"morm_amount":{"type":"number"}},"required":["morm_amount"]}`
}

func (t *NegotiateTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	offerID, _ := args["offer_id"].(string)
	mormAmount := 0.0
	if m, ok := args["morm_amount"].(float64); ok {
		mormAmount = m
	}
	offer, err := t.UseCase.Execute(ctx, offerID, mormAmount)
	if err != nil {
		return "", err
	}
	return dto.FormatOffer(offer), nil
}
