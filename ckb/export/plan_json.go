package export

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/planning"
)

// ExportPlanJSON serializes an ArtifactPlan to JSON in a deterministic order.
func ExportPlanJSON(plan *planning.ArtifactPlan, w io.Writer) error {
	// Shallow copy of plan
	copied := *plan

	// 1. Sort Conflicts alphabetically by ID
	if len(copied.Conflicts) > 0 {
		sortedConflicts := make([]claims.ClaimConflict, len(copied.Conflicts))
		copy(sortedConflicts, copied.Conflicts)
		sort.Slice(sortedConflicts, func(i, j int) bool {
			return sortedConflicts[i].ID < sortedConflicts[j].ID
		})
		copied.Conflicts = sortedConflicts
	}

	// 2. Sort Gaps alphabetically by Code, then Description
	if len(copied.Gaps) > 0 {
		sortedGaps := make([]planning.Gap, len(copied.Gaps))
		copy(sortedGaps, copied.Gaps)
		sort.Slice(sortedGaps, func(i, j int) bool {
			if sortedGaps[i].Code != sortedGaps[j].Code {
				return sortedGaps[i].Code < sortedGaps[j].Code
			}
			return sortedGaps[i].Description < sortedGaps[j].Description
		})
		copied.Gaps = sortedGaps
	}

	// 3. Sort Provenance alphabetically by ClaimID
	if len(copied.Provenance) > 0 {
		sortedProv := make([]planning.ProvenanceRecord, len(copied.Provenance))
		copy(sortedProv, copied.Provenance)
		sort.Slice(sortedProv, func(i, j int) bool {
			return sortedProv[i].ClaimID < sortedProv[j].ClaimID
		})
		copied.Provenance = sortedProv
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(copied)
}
