package entity

// Offer represents a negotiation offer on a skill.
type Offer struct {
	ID         string  `json:"id"`
	SkillID    string  `json:"skill_id"`
	MORMAmount float64 `json:"morm_amount"`
	Status     string  `json:"status"` // pending, accepted, rejected
}
