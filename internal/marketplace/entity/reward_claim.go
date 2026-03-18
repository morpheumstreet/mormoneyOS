package entity

// RewardClaim represents a MORM reward claim after safe install/run.
type RewardClaim struct {
	ID        string  `json:"id"`
	SkillID   string  `json:"skill_id"`
	RunProof  string  `json:"run_proof"`
	MORMAmount float64 `json:"morm_amount"`
	Status    string  `json:"status"` // pending, claimed, rejected
}
