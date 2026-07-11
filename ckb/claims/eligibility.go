package claims

import (
	"fmt"

	"github.com/admbahm/theForge/ckb/model"
)

// EligibilityStatus represents whether a claim can be used.
type EligibilityStatus string

const (
	StatusEligible            EligibilityStatus = "Eligible"
	StatusEligibleWithCaution EligibilityStatus = "EligibleWithCaution"
	StatusIneligible          EligibilityStatus = "Ineligible"
)

// Stable diagnostic and decision reason codes
const (
	CodeClaimEligible             = "CKB-CLAIM-ELIGIBLE"
	CodeClaimPrivate              = "CKB-CLAIM-PRIVATE"
	CodeClaimUnverified           = "CKB-CLAIM-UNVERIFIED"
	CodeClaimLowConfidence        = "CKB-CLAIM-LOW-CONFIDENCE"
	CodeClaimDisputed             = "CKB-CLAIM-DISPUTED"
	CodeClaimSuperseded           = "CKB-CLAIM-SUPERSEDED"
	CodeClaimConflicting          = "CKB-CLAIM-CONFLICTING"
	CodeClaimMissingEvidence      = "CKB-CLAIM-MISSING-EVIDENCE"
	CodeClaimArtifactIncompatible = "CKB-CLAIM-ARTIFACT-INCOMPATIBLE"
	CodeClaimDuplicate            = "CKB-CLAIM-DUPLICATE"
	CodeClaimOutsideScope         = "CKB-CLAIM-OUTSIDE-TARGET-SCOPE"
	CodeClaimUnsupportedInference = "CKB-CLAIM-UNSUPPORTED-INFERENCE"
	CodeClaimInactiveStatus       = "CKB-CLAIM-INACTIVE-STATUS"
	CodeClaimInactiveLifecycle    = "CKB-CLAIM-INACTIVE-LIFECYCLE"
)

// Policy defines eligibility evaluation parameters.
type Policy struct {
	ID                  string                  `json:"policy_id"`
	MinVerification     model.VerificationLevel `json:"min_verification"`
	MinConfidence       float64                 `json:"min_confidence"`
	AllowedVisibilities []model.Visibility      `json:"allowed_visibilities"`
	ExcludeDisputed     bool                    `json:"exclude_disputed"`
	ExcludeSuperseded   bool                    `json:"exclude_superseded"`
	RequireEvidence     bool                    `json:"require_evidence"`
	AllowedStatuses     []model.Status          `json:"allowed_statuses,omitempty"`
	AllowedLifecycles   []model.LifecycleState  `json:"allowed_lifecycles,omitempty"`
	AllowInactive       bool                    `json:"allow_inactive,omitempty"`
}

// EvaluateEligibility evaluates a claim against a validation policy.
func EvaluateEligibility(c Claim, p Policy) (EligibilityStatus, string, string) {
	if !p.AllowInactive {
		// 1. Check allowed statuses (defaults to Active only)
		cStatus := c.Status
		if cStatus == "" {
			cStatus = string(model.StatusActive)
		}
		allowedStatuses := p.AllowedStatuses
		if len(allowedStatuses) == 0 {
			allowedStatuses = []model.Status{model.StatusActive}
		}
		statusAllowed := false
		for _, s := range allowedStatuses {
			if cStatus == string(s) {
				statusAllowed = true
				break
			}
		}
		if !statusAllowed {
			return StatusIneligible, CodeClaimInactiveStatus, fmt.Sprintf("Claim status %q is not allowed by policy", c.Status)
		}

		// 2. Check allowed lifecycles (defaults to Active and Completed only)
		cLifecycle := c.Lifecycle
		if cLifecycle == "" {
			cLifecycle = string(model.LifecycleActive)
		}
		allowedLifecycles := p.AllowedLifecycles
		if len(allowedLifecycles) == 0 {
			allowedLifecycles = []model.LifecycleState{model.LifecycleActive, model.LifecycleCompleted}
		}
		lifecycleAllowed := false
		for _, l := range allowedLifecycles {
			if cLifecycle == string(l) {
				lifecycleAllowed = true
				break
			}
		}
		if !lifecycleAllowed {
			return StatusIneligible, CodeClaimInactiveLifecycle, fmt.Sprintf("Claim lifecycle %q is not allowed by policy", c.Lifecycle)
		}
	}

	// 3. Exclude disputed
	if p.ExcludeDisputed && (c.Verification == model.VerificationDisputed) {
		return StatusIneligible, CodeClaimDisputed, "Claim has been marked as Disputed"
	}

	// 4. Exclude superseded
	if p.ExcludeSuperseded && (c.Verification == model.VerificationSuperseded || c.Status == "Superseded") {
		return StatusIneligible, CodeClaimSuperseded, "Claim has been marked as Superseded"
	}

	// 4. Visibility enforcement
	allowedVis := false
	for _, vis := range p.AllowedVisibilities {
		if c.Visibility == vis {
			allowedVis = true
			break
		}
	}
	if !allowedVis {
		return StatusIneligible, CodeClaimPrivate, fmt.Sprintf("Claim visibility %q is not allowed by policy", c.Visibility)
	}

	// 5. Verification level threshold check
	if VerificationRank(c.Verification) < VerificationRank(p.MinVerification) {
		return StatusIneligible, CodeClaimUnverified, fmt.Sprintf("Claim verification %q is below policy minimum %q", c.Verification, p.MinVerification)
	}

	// 6. Confidence threshold check
	if c.Confidence < p.MinConfidence {
		return StatusIneligible, CodeClaimLowConfidence, fmt.Sprintf("Claim confidence %.2f is below policy minimum %.2f", c.Confidence, p.MinConfidence)
	}

	// 7. Require evidence check
	if p.RequireEvidence && len(c.EvidenceObjectIDs) == 0 {
		return StatusIneligible, CodeClaimMissingEvidence, "Claim has zero authorized direct supporting evidence IDs"
	}

	return StatusEligible, CodeClaimEligible, "Claim is eligible"
}

func VerificationRank(v model.VerificationLevel) int {
	switch v {
	case model.VerificationDisputed:
		return -1
	case model.VerificationSuperseded:
		return 0
	case model.VerificationUnverified:
		return 1
	case model.VerificationSelfAttested:
		return 2
	case model.VerificationArtifactSupported:
		return 3
	case model.VerificationIndependentlyVerified:
		return 4
	default:
		return 1
	}
}
