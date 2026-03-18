// Package entity holds pure data models for the marketplace domain.
package entity

// Skill represents a marketplace skill (Perp Trading, Mirofish Decision Pack, Prediction Resolution).
type Skill struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	PriceMORM       float64        `json:"price_morm"`
	Badges          []string       `json:"badges"` // Verified, Audited, Perp-Ready
	Permissions     map[string]any `json:"permissions"`
	SecurityHash    string         `json:"security_hash"`
	MirofishPreview string         `json:"mirofish_preview,omitempty"`
	PerpReady       bool           `json:"perp_ready"`
}
