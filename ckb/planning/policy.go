package planning

import (
	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/model"
)

// Predefined Policy Constants
const (
	PolicyStrictPublic    = "StrictPublic"
	PolicyComprehensiveCV = "ComprehensiveCV"
	PolicyInterviewPrep   = "InterviewPrep"
	PolicyInternalRecord  = "InternalRecord"
)

var policies = map[string]claims.Policy{
	PolicyStrictPublic: {
		ID:              PolicyStrictPublic,
		MinVerification: model.VerificationSelfAttested,
		MinConfidence:   0.80,
		AllowedVisibilities: []model.Visibility{
			model.VisibilityPublic,
		},
		ExcludeDisputed:   true,
		ExcludeSuperseded: true,
		RequireEvidence:   false,
		AllowedStatuses: []model.Status{
			model.StatusActive,
		},
		AllowedLifecycles: []model.LifecycleState{
			model.LifecycleActive,
			model.LifecycleCompleted,
		},
	},
	PolicyComprehensiveCV: {
		ID:              PolicyComprehensiveCV,
		MinVerification: model.VerificationUnverified,
		MinConfidence:   0.50,
		AllowedVisibilities: []model.Visibility{
			model.VisibilityPublic,
			model.VisibilityInternal,
		},
		ExcludeDisputed:   true,
		ExcludeSuperseded: true,
		RequireEvidence:   false,
		AllowedStatuses: []model.Status{
			model.StatusActive,
		},
		AllowedLifecycles: []model.LifecycleState{
			model.LifecycleActive,
			model.LifecycleCompleted,
		},
	},
	PolicyInterviewPrep: {
		ID:              PolicyInterviewPrep,
		MinVerification: model.VerificationUnverified,
		MinConfidence:   0.00,
		AllowedVisibilities: []model.Visibility{
			model.VisibilityPublic,
			model.VisibilityInternal,
			model.VisibilityConfidential,
		},
		ExcludeDisputed:   true,
		ExcludeSuperseded: false,
		RequireEvidence:   false,
		AllowedStatuses: []model.Status{
			model.StatusActive,
		},
		AllowedLifecycles: []model.LifecycleState{
			model.LifecycleActive,
			model.LifecycleCompleted,
		},
	},
	PolicyInternalRecord: {
		ID:              PolicyInternalRecord,
		MinVerification: model.VerificationUnverified,
		MinConfidence:   0.00,
		AllowedVisibilities: []model.Visibility{
			model.VisibilityPublic,
			model.VisibilityInternal,
			model.VisibilityConfidential,
		},
		ExcludeDisputed:   false,
		ExcludeSuperseded: false,
		RequireEvidence:   false,
		AllowInactive:     true,
		AllowedStatuses: []model.Status{
			model.StatusActive,
			model.StatusDraft,
			model.StatusDeprecated,
		},
		AllowedLifecycles: []model.LifecycleState{
			model.LifecycleActive,
			model.LifecycleCompleted,
			model.LifecyclePlanned,
			model.LifecycleArchived,
		},
	},
}

// GetPolicyByID retrieves a predefined policy by its identifier.
func GetPolicyByID(id string) (claims.Policy, bool) {
	p, ok := policies[id]
	return p, ok
}
