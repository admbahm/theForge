package claims

import (
	"sort"
	"strings"

	"github.com/admbahm/theForge/ckb/model"
)

// DeduplicateClaims merges exact duplicate claims (same Kind and Statement)
// and aggregates their SourceObjectIDs, EvidenceObjectIDs, and SourceLocations.
func DeduplicateClaims(claims []Claim) []Claim {
	grouped := make(map[string][]Claim)
	for _, c := range claims {
		// Key by kind, active eligibility, and normalized statement text
		isActive := "active"
		cStatus := c.Status
		if cStatus == "" {
			cStatus = "Active"
		}
		cLifecycle := c.Lifecycle
		if cLifecycle == "" {
			cLifecycle = "Active"
		}
		if cStatus != "Active" || (cLifecycle != "Active" && cLifecycle != "Completed") {
			isActive = "inactive"
		}
		key := string(c.Kind) + "|" + isActive + "|" + strings.ToLower(strings.Join(strings.Fields(c.Statement), " "))
		grouped[key] = append(grouped[key], c)
	}

	var result []Claim
	var keys []string
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		group := grouped[key]
		if len(group) == 1 {
			result = append(result, group[0])
			continue
		}

		// Merge details
		merged := group[0]
		seenSources := make(map[string]bool)
		seenEvs := make(map[string]bool)

		for _, s := range merged.SourceObjectIDs {
			seenSources[s] = true
		}
		for _, e := range merged.EvidenceObjectIDs {
			seenEvs[e] = true
		}

		for i := 1; i < len(group); i++ {
			c := group[i]
			for _, s := range c.SourceObjectIDs {
				if !seenSources[s] {
					seenSources[s] = true
					merged.SourceObjectIDs = append(merged.SourceObjectIDs, s)
				}
			}
			for _, e := range c.EvidenceObjectIDs {
				if !seenEvs[e] {
					seenEvs[e] = true
					merged.EvidenceObjectIDs = append(merged.EvidenceObjectIDs, e)
				}
			}
			merged.SourceLocations = append(merged.SourceLocations, c.SourceLocations...)

			// Merge visibility (most restrictive)
			merged.Visibility = getMoreRestrictiveVisibility(merged.Visibility, c.Visibility)

			// Merge verification rank
			merged.Verification = getMergedVerification(merged.Verification, c.Verification)
		}

		sort.Strings(merged.SourceObjectIDs)
		sort.Strings(merged.EvidenceObjectIDs)

		merged.ID = GenerateClaimID(merged.SourceObjectIDs, merged.Kind, merged.Statement)
		result = append(result, merged)
	}

	return result
}

func getMoreRestrictiveVisibility(v1, v2 model.Visibility) model.Visibility {
	if v1 == model.VisibilityConfidential || v2 == model.VisibilityConfidential {
		return model.VisibilityConfidential
	}
	if v1 == model.VisibilityInternal || v2 == model.VisibilityInternal {
		return model.VisibilityInternal
	}
	return model.VisibilityPublic
}

func getMergedVerification(v1, v2 model.VerificationLevel) model.VerificationLevel {
	if v1 == model.VerificationDisputed || v2 == model.VerificationDisputed {
		return model.VerificationDisputed
	}
	if v1 == model.VerificationSuperseded || v2 == model.VerificationSuperseded {
		return model.VerificationSuperseded
	}
	r1 := VerificationRank(v1)
	r2 := VerificationRank(v2)
	if r1 >= r2 {
		return v1
	}
	return v2
}
