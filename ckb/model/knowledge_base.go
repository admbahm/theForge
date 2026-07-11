package model

// KnowledgeBase represents the parsed career graph container.
type KnowledgeBase struct {
	Objects       map[string]*Object `json:"objects"`
	Relationships []Relationship     `json:"relationships,omitempty"`
}

// NewKnowledgeBase initializes an empty KnowledgeBase instance.
func NewKnowledgeBase() *KnowledgeBase {
	return &KnowledgeBase{
		Objects:       make(map[string]*Object),
		Relationships: make([]Relationship, 0),
	}
}
