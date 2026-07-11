package planning

import (
	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/model"
)

// PlannedClaim defines an eligible claim incorporated into the generation plan.
type PlannedClaim struct {
	Claim           claims.Claim   `json:"claim"`
	SelectionReason string         `json:"selection_reason"`
	RelevanceScore  int            `json:"relevance_score"`
	ScoreComponents map[string]int `json:"score_components"`
	Section         string         `json:"section"`
	Rank            int            `json:"rank"`
	Warnings        []string       `json:"warnings,omitempty"`
}

// ExcludedClaim records why a claim was filtered out during planning.
type ExcludedClaim struct {
	ClaimID              claims.ClaimID `json:"claim_id"`
	ExclusionReason      string         `json:"exclusion_reason"`
	ExclusionReasonCode  string         `json:"exclusion_reason_code"`
	SectionConsidered    string         `json:"section_considered,omitempty"`
	HumanOverrideAllowed bool           `json:"human_override_allowed"`
}

// ProvenanceRecord defines audit details linking output back to source materials.
type ProvenanceRecord struct {
	ClaimID      claims.ClaimID          `json:"claim_id"`
	SourceFile   string                  `json:"source_file"`
	SourceLine   int                     `json:"source_line"`
	EvidenceIDs  []string                `json:"evidence_ids,omitempty"`
	Confidence   float64                 `json:"confidence"`
	Verification model.VerificationLevel `json:"verification_level"`
}

// Gap represents a missing requirement or evidence deficiency.
type Gap struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"` // "Warning", "Error"
	Description string `json:"description"`
	TargetField string `json:"target_field,omitempty"`
	Context     string `json:"context,omitempty"`
}

// ArtifactPlan represents the deterministic outcome of the planning process.
type ArtifactPlan struct {
	ID                     string                 `json:"plan_id"`
	ArtifactType           ArtifactType           `json:"artifact_type"`
	Policy                 claims.Policy          `json:"policy"`
	Target                 *TargetProfile         `json:"target,omitempty"`
	SelectedClaims         []PlannedClaim         `json:"selected_claims"`
	ExcludedClaims         []ExcludedClaim        `json:"excluded_claims"`
	Conflicts              []claims.ClaimConflict `json:"conflicts,omitempty"`
	Gaps                   []Gap                  `json:"gaps,omitempty"`
	Provenance             []ProvenanceRecord     `json:"provenance"`
	Diagnostics            []model.Diagnostic     `json:"diagnostics,omitempty"`
	SchemaVersion          string                 `json:"schema_version"`
	Overrides              []Override             `json:"overrides,omitempty"`
	MaxRoles               int                    `json:"max_roles,omitempty"`
	MaxAchievementsPerRole int                    `json:"max_achievements_per_role,omitempty"`
	MaxSkills              int                    `json:"max_skills,omitempty"`
	MaxEducation           int                    `json:"max_education,omitempty"`
}
