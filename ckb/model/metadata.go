package model

import "time"

// Metadata represents the parsed and validated canonical metadata of a CKB object.
type Metadata struct {
	SchemaVersion string            `json:"schema_version"`
	ID            string            `json:"id"`
	Type          ObjectType        `json:"type"`
	Status        Status            `json:"status"`
	Verification  VerificationLevel `json:"verification_level"`
	Confidence    float64           `json:"confidence"`
	Visibility    Visibility        `json:"visibility"`
	Source        string            `json:"source"`
	LastUpdated   time.Time         `json:"last_updated"`
	Lifecycle     LifecycleState    `json:"lifecycle_state"`
	RelatedDocs   []string          `json:"related_documents,omitempty"`
	RelatedExps   []string          `json:"related_experiences,omitempty"`
	RelatedProjs  []string          `json:"related_projects,omitempty"`
	RelatedEvs    []string          `json:"related_evidence,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
}
