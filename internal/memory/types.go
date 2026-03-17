package memory

// MemoryTier identifies the 5-tier memory system (memory-system-5-tier.md).
type MemoryTier string

const (
	TierWorking     MemoryTier = "working"
	TierEpisodic    MemoryTier = "episodic"
	TierSemantic    MemoryTier = "semantic"
	TierProcedural  MemoryTier = "procedural"
	TierRelationship MemoryTier = "relationship"
)

// Extraction holds structured knowledge extracted from a ReAct turn.
// Used by the ingester and consolidator pipeline.
type Extraction struct {
	Facts         []Fact
	Episodes      []Episode
	Procedures    []Procedure
	Relationships []RelationshipUpdate
	Importance    float64 // 0.0–1.0 for pruning
}

// Fact is a semantic memory entry (category, key, value).
type Fact struct {
	Category   string  `json:"category"`
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

// Episode is an episodic memory entry (event with outcome).
type Episode struct {
	EventType string  `json:"event_type"`
	Summary   string  `json:"summary"`
	Detail    string  `json:"detail,omitempty"`
	Outcome   string  `json:"outcome,omitempty"`
	Importance float64 `json:"importance"`
}

// Procedure is a procedural memory entry (name + steps).
type Procedure struct {
	Name        string   `json:"name"`
	Steps       []string `json:"steps"`
	SuccessRate float64  `json:"success_rate"`
}

// RelationshipUpdate describes a relationship memory change.
type RelationshipUpdate struct {
	EntityAddress string  `json:"entity_address"`
	EntityName    string  `json:"entity_name,omitempty"`
	Type          string  `json:"type,omitempty"`
	TrustDelta    float64 `json:"trust_delta"`
}
