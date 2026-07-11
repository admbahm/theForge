package claims

import (
	"sort"
	"strings"
)

// NormalizeClaim normalizes whitespace, metrics ordering, and list string values deterministically.
// Normalization must never perform unsafe inferences or strengthen claims.
func NormalizeClaim(c Claim) Claim {
	// 1. Whitespace normalization
	c.Statement = strings.Join(strings.Fields(c.Statement), " ")
	c.Subject = ClaimSubject(strings.Join(strings.Fields(string(c.Subject)), " "))
	c.Predicate = ClaimPredicate(strings.Join(strings.Fields(string(c.Predicate)), " "))
	c.Value = ClaimValue(strings.Join(strings.Fields(string(c.Value)), " "))

	// 2. Sorted source and evidence ID lists for stable IDs/comparisons
	if len(c.SourceObjectIDs) > 0 {
		sort.Strings(c.SourceObjectIDs)
	}
	if len(c.EvidenceObjectIDs) > 0 {
		sort.Strings(c.EvidenceObjectIDs)
	}
	if len(c.Skills) > 0 {
		sort.Strings(c.Skills)
	}
	if len(c.Organizations) > 0 {
		sort.Strings(c.Organizations)
	}
	if len(c.Projects) > 0 {
		sort.Strings(c.Projects)
	}

	// 3. Deterministic metrics sorting
	if len(c.Metrics) > 0 {
		sort.Slice(c.Metrics, func(i, j int) bool {
			if c.Metrics[i].Name != c.Metrics[j].Name {
				return c.Metrics[i].Name < c.Metrics[j].Name
			}
			if c.Metrics[i].Unit != c.Metrics[j].Unit {
				return c.Metrics[i].Unit < c.Metrics[j].Unit
			}
			return c.Metrics[i].Value < c.Metrics[j].Value
		})
	}

	// Recompute ID based on the normalized state
	c.ID = GenerateClaimID(c.SourceObjectIDs, c.Kind, c.Statement)

	return c
}
