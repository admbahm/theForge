package planning

import "github.com/admbahm/theForge/ckb/claims"

// ArtifactType represents the downstream document target.
type ArtifactType string

const (
	TypeResume        ArtifactType = "OnePageResume"
	TypeCV            ArtifactType = "FullCV"
	TypeBiography     ArtifactType = "ProfessionalBiography"
	TypeSTARStory     ArtifactType = "STARStory"
	TypeSkillsSummary ArtifactType = "SkillsSummary"
	TypeCoverLetter   ArtifactType = "CoverLetter"
)

// TargetProfile defines matching criteria representing the target job description.
type TargetProfile struct {
	RoleTitle              string   `json:"target_role_title,omitempty"`
	Company                string   `json:"company,omitempty"`
	RoleFamily             string   `json:"role_family,omitempty"`
	DesiredSkills          []string `json:"desired_skills,omitempty"`
	DesiredTechnologies    []string `json:"desired_technologies,omitempty"`
	DesiredTraits          []string `json:"desired_traits,omitempty"`
	DesiredKnowledge       []string `json:"desired_domain_knowledge,omitempty"`
	DesiredOutcomes        []string `json:"desired_outcomes,omitempty"`
	RecencyPreferenceYears int      `json:"recency_preference_years,omitempty"`
	KeywordPriorities      []string `json:"keyword_priorities,omitempty"`
}

// Override specifies human-in-the-loop validation updates.
type Override struct {
	ClaimID  claims.ClaimID `json:"claim_id"`
	Action   string         `json:"action"` // "Include", "Exclude", "PinSection"
	Section  string         `json:"section,omitempty"`
	Priority int            `json:"priority,omitempty"`
}

// PlanRequest configures the planning parameters.
type PlanRequest struct {
	ArtifactType           ArtifactType   `json:"artifact_type"`
	PolicyID               string         `json:"policy_id"`
	Target                 *TargetProfile `json:"target,omitempty"`
	Overrides              []Override     `json:"overrides,omitempty"`
	MaxRoles               int            `json:"max_roles,omitempty"`
	MaxAchievementsPerRole int            `json:"max_achievements_per_role,omitempty"`
	MaxSkills              int            `json:"max_skills,omitempty"`
	MaxEducation           int            `json:"max_education,omitempty"`
}
