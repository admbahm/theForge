package rendering

import (
	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/planning"
)

// ArtifactID represents a deterministic identifier for an artifact.
type ArtifactID string

// ArtifactSchemaVersion represents the schema layout version of the intermediate artifact model.
type ArtifactSchemaVersion string

// Artifact represents a strongly-typed intermediate career document generated from an authorized plan.
type Artifact struct {
	ID            ArtifactID            `json:"artifact_id"`
	Type          planning.ArtifactType `json:"artifact_type"`
	SchemaVersion ArtifactSchemaVersion `json:"schema_version"`
	Title         string                `json:"title"`
	Subtitle      string                `json:"subtitle,omitempty"`
	Sections      []ArtifactSection     `json:"sections"`
	Manifest      ArtifactManifest      `json:"manifest"`
	Diagnostics   []model.Diagnostic    `json:"diagnostics,omitempty"`
}

// ArtifactSection represents a logical group of content entries (e.g. Experience, Skills).
type ArtifactSection struct {
	ID         string                      `json:"section_id"`
	Kind       string                      `json:"section_kind"`
	Heading    string                      `json:"heading"`
	Entries    []ArtifactEntry             `json:"entries"`
	Provenance []planning.ProvenanceRecord `json:"provenance,omitempty"`
}

// ArtifactEntry represents a structured block of prose, list items, or keywords mapping back to claims.
type ArtifactEntry struct {
	ID              string           `json:"entry_id"`
	Kind            string           `json:"entry_kind"` // "bullet", "paragraph", "header", "list"
	Text            string           `json:"text"`
	ClaimIDs        []claims.ClaimID `json:"claim_ids,omitempty"`
	EvidenceIDs     []string         `json:"evidence_ids,omitempty"`
	SourceObjectIDs []string         `json:"source_object_ids,omitempty"`
	Warnings        []string         `json:"warnings,omitempty"`
}

// ArtifactManifest defines an audit trail mapping the rendered entries back to plan selection decisions.
type ArtifactManifest struct {
	ArtifactID         string                      `json:"artifact_id"`
	ArtifactType       planning.ArtifactType       `json:"artifact_type"`
	PlanID             string                      `json:"plan_id"`
	PlannerVersion     string                      `json:"planner_version"`
	RendererVersion    string                      `json:"renderer_version"`
	RenderingPolicy    string                      `json:"rendering_policy"`
	TargetIdentifier   string                      `json:"target_identifier,omitempty"`
	SelectedClaimIDs   []claims.ClaimID            `json:"selected_claim_ids"`
	ExcludedClaimCount int                         `json:"excluded_claim_count"`
	RenderedEntryIDs   []string                    `json:"rendered_entry_ids"`
	ClaimToEntryMap    map[claims.ClaimID][]string `json:"claim_to_entry_map"`
	SourceReferences   []string                    `json:"source_references"`
	EvidenceReferences []string                    `json:"evidence_references,omitempty"`
	Warnings           []string                    `json:"warnings,omitempty"`
	Overrides          []planning.Override         `json:"overrides,omitempty"`
	ContentDigest      string                      `json:"content_digest"`
}
