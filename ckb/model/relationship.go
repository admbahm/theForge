package model

// Relationship represents a typed, directional graph relationship edge.
type Relationship struct {
	SourceID string           `json:"source_id"`
	TargetID string           `json:"target_id"`
	Type     RelationshipType `json:"type"`
	Source   SourceLocation   `json:"source"`
}
