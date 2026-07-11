package rendering

import (
	"sort"

	"github.com/admbahm/theForge/ckb/claims"
)

func claimIDsForClaims(claimList []claims.Claim) []claims.ClaimID {
	seen := make(map[claims.ClaimID]bool)
	for _, c := range claimList {
		seen[c.ID] = true
	}
	ids := make([]claims.ClaimID, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func sourceIDsForClaims(claimList []claims.Claim) []string {
	seen := make(map[string]bool)
	for _, c := range claimList {
		for _, id := range c.SourceObjectIDs {
			seen[id] = true
		}
	}
	return sortedStringKeys(seen)
}

func evidenceIDsForClaims(claimList []claims.Claim) []string {
	seen := make(map[string]bool)
	for _, c := range claimList {
		for _, id := range c.EvidenceObjectIDs {
			seen[id] = true
		}
	}
	return sortedStringKeys(seen)
}

func sortedStringKeys(seen map[string]bool) []string {
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
