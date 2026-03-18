package entity

// Deal represents a completed purchase/transaction for a skill.
type Deal struct {
	ID        string  `json:"id"`
	SkillID   string  `json:"skill_id"`
	OfferID   string  `json:"offer_id"`
	MORMAmount float64 `json:"morm_amount"`
	Status    string  `json:"status"` // completed, disputed
}
