package dto

import (
	"encoding/json"

	"github.com/morpheumlabs/mormoneyos-go/internal/marketplace/entity"
)

// FormatOffer returns an offer as JSON string.
func FormatOffer(offer *entity.Offer) string {
	if offer == nil {
		return `{"offer":null}`
	}
	b, _ := json.Marshal(map[string]any{"offer": offer})
	return string(b)
}
