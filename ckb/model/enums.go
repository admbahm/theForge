package model

// ObjectType represents the type of a CKB node.
type ObjectType string

const (
	TypeProfile        ObjectType = "Profile"
	TypeTimeline       ObjectType = "Timeline"
	TypeExperience     ObjectType = "Experience"
	TypeProject        ObjectType = "Project"
	TypeSkill          ObjectType = "Skill"
	TypeAccomplishment ObjectType = "Accomplishment"
	TypeCredential     ObjectType = "Credential"
	TypeContribution   ObjectType = "Contribution"
	TypeReference      ObjectType = "Reference"
	TypeEvidence       ObjectType = "Evidence"
	TypeEducation      ObjectType = "Education"
)

// Status represents the curation status of a CKB document.
type Status string

const (
	StatusDraft      Status = "Draft"
	StatusActive     Status = "Active"
	StatusDeprecated Status = "Deprecated"
)

// VerificationLevel represents the evidence level backing a record.
type VerificationLevel string

const (
	VerificationUnverified            VerificationLevel = "Unverified"
	VerificationSelfAttested          VerificationLevel = "Self-Attested"
	VerificationArtifactSupported     VerificationLevel = "Artifact-Supported"
	VerificationIndependentlyVerified VerificationLevel = "Independently-Verified"
	VerificationDisputed              VerificationLevel = "Disputed"
	VerificationSuperseded            VerificationLevel = "Superseded"
)

// Visibility represents document access scope.
type Visibility string

const (
	VisibilityPublic       Visibility = "Public"
	VisibilityConfidential Visibility = "Confidential"
	VisibilityInternal     Visibility = "Internal"
)

// LifecycleState represents the real-world lifecycle state of a career item.
type LifecycleState string

const (
	LifecyclePlanned   LifecycleState = "Planned"
	LifecycleActive    LifecycleState = "Active"
	LifecycleCompleted LifecycleState = "Completed"
	LifecycleArchived  LifecycleState = "Archived"
)

// RelationshipType represents the field type declaring a graph relation.
type RelationshipType string

const (
	RelDocuments  RelationshipType = "Related Documents"
	RelExperience RelationshipType = "Related Experience"
	RelProjects   RelationshipType = "Related Projects"
	RelEvidence   RelationshipType = "Related Evidence"
)
