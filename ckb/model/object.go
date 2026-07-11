package model

// Object represents a parsed and validated CKB object (node).
type Object struct {
	ID            string         `json:"id"`
	Type          ObjectType     `json:"type"`
	SourceFile    string         `json:"source_file"`
	Metadata      Metadata       `json:"metadata"`
	Sections      []Section      `json:"sections,omitempty"`
	Relationships []Relationship `json:"relationships,omitempty"`
}
