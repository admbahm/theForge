package claims

import (
	"time"

	"github.com/admbahm/theForge/ckb/model"
)

// ClaimID represents a deterministic unique identifier for a claim.
type ClaimID string

// ClaimKind classifies the type of career assertion being made.
type ClaimKind string

// ClaimSubject defines the entity the claim is about (typically the candidate or an organization).
type ClaimSubject string

// ClaimPredicate describes the action or assertion relation.
type ClaimPredicate string

// ClaimValue holds the payload or parameter value of the claim.
type ClaimValue string

const (
	KindEmployment          ClaimKind = "employment"
	KindRole                ClaimKind = "role"
	KindResponsibility      ClaimKind = "responsibility"
	KindAccomplishment      ClaimKind = "accomplishment"
	KindProjectContribution ClaimKind = "project_contribution"
	KindTechnicalSkill      ClaimKind = "technical_skill"
	KindLeadershipSkill     ClaimKind = "leadership_skill"
	KindEducation           ClaimKind = "education"
	KindCredential          ClaimKind = "credential"
	KindAward               ClaimKind = "award"
	KindPublication         ClaimKind = "publication"
	KindSpeaking            ClaimKind = "speaking"
	KindMentoring           ClaimKind = "mentoring"
	KindBusinessImpact      ClaimKind = "business_impact"
	KindTechnicalImpact     ClaimKind = "technical_impact"
	KindMeasurableResult    ClaimKind = "measurable_result"
	KindCareerObjective     ClaimKind = "career_objective"
	KindProfessionalPref    ClaimKind = "professional_preference"
	KindChronologyEvent     ClaimKind = "chronology_event"
	KindSTARSituation       ClaimKind = "star_situation"
	KindSTARTask            ClaimKind = "star_task"
	KindSTARAction          ClaimKind = "star_action"
	KindSTARResult          ClaimKind = "star_result"
)

// TimeRange represents a duration spanning two dates.
type TimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"` // Zero time represents ongoing/present
}

// Metric represents a quantitative outcome.
type Metric struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Status      string  `json:"status,omitempty"` // "Approved", "Ambiguous", "Target", "Estimate", "Contextual"
	IsAmbiguous bool    `json:"is_ambiguous,omitempty"`
	Reason      string  `json:"reason,omitempty"`
}

// Claim represents a strongly typed normalized career assertion with full provenance.
type Claim struct {
	ID                ClaimID                 `json:"id"`
	Kind              ClaimKind               `json:"kind"`
	Statement         string                  `json:"statement"`
	Subject           ClaimSubject            `json:"subject"`
	Predicate         ClaimPredicate          `json:"predicate"`
	Value             ClaimValue              `json:"value"`
	SourceObjectIDs   []string                `json:"source_object_ids"`
	EvidenceObjectIDs []string                `json:"evidence_object_ids"`
	Verification      model.VerificationLevel `json:"verification_level"`
	Confidence        float64                 `json:"confidence"`
	Visibility        model.Visibility        `json:"visibility"`
	TimeRange         *TimeRange              `json:"time_range,omitempty"`
	Skills            []string                `json:"skills,omitempty"`
	Organizations     []string                `json:"organizations,omitempty"`
	Projects          []string                `json:"projects,omitempty"`
	Metrics           []Metric                `json:"metrics,omitempty"`
	SourceLocations   []model.SourceLocation  `json:"source_locations,omitempty"`
	Status            string                  `json:"status"` // Active, Superseded, etc.
	Lifecycle         string                  `json:"lifecycle_state"`
}
