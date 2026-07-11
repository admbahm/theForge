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

// ObjectTypePrefixes defines the canonical mapping between object type and ID prefix.
var ObjectTypePrefixes = map[ObjectType]string{
	TypeProfile:        "profile",
	TypeTimeline:       "timeline",
	TypeExperience:     "exp",
	TypeProject:        "proj",
	TypeSkill:          "skill",
	TypeAccomplishment: "acc",
	TypeCredential:     "cred",
	TypeContribution:   "contrib",
	TypeReference:      "ref",
	TypeEvidence:       "ev",
	TypeEducation:      "edu",
}

// RequiredPrefixForObjectType returns the required ID prefix for an object type.
func RequiredPrefixForObjectType(t ObjectType) (string, bool) {
	prefix, ok := ObjectTypePrefixes[t]
	return prefix, ok
}

// IsValidObjectType reports whether t is one of the canonical CKB object types.
func IsValidObjectType(t ObjectType) bool {
	_, ok := ObjectTypePrefixes[t]
	return ok
}

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
