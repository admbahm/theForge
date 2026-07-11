package export

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/admbahm/theForge/ckb/rendering"
)

// ProvenanceSidecarEntry defines audit trail metadata for an entry in the rendered document.
type ProvenanceSidecarEntry struct {
	ArtifactEntryID string   `json:"artifact_entry_id"`
	ClaimIDs        []string `json:"claim_ids"`
	SourceObjectIDs []string `json:"source_object_ids"`
	EvidenceIDs     []string `json:"evidence_ids"`
	Transformations []string `json:"transformations"`
}

// ExportProvenanceSidecar outputs a deterministic machine-readable JSON mapping of entry-to-claim audit linkages.
func ExportProvenanceSidecar(artifact *rendering.Artifact, w io.Writer) error {
	var entries []ProvenanceSidecarEntry

	for _, sec := range artifact.Sections {
		for _, ent := range sec.Entries {
			var claimIDs []string
			for _, cid := range ent.ClaimIDs {
				claimIDs = append(claimIDs, string(cid))
			}

			// Deduce transformations based on entry content properties
			var trans []string
			trans = append(trans, "whitespace-normalization")

			if len(claimIDs) > 1 {
				trans = append(trans, "compatible-claim-combination")
			}

			// Fuzzy check if tense was adjusted (e.g. if text starts with verb-conjugated forms)
			hasConjugatedVerb := false
			words := strings.Fields(strings.ToLower(ent.Text))
			if len(words) > 0 {
				firstWord := strings.TrimRight(words[0], ".,*")
				if firstWord == "led" || firstWord == "served" || firstWord == "utilized" || firstWord == "reduced" || firstWord == "improved" {
					hasConjugatedVerb = true
				}
			}
			if hasConjugatedVerb {
				trans = append(trans, "tense-adaptation")
			}

			// Check if abbreviations were expanded
			if strings.Contains(ent.Text, "Google Cloud Platform") || strings.Contains(ent.Text, "Amazon Web Services") {
				trans = append(trans, "abbreviation-expansion")
			}

			// Check if metrics were converted to compact style
			if strings.Contains(ent.Text, "~") {
				trans = append(trans, "metric-formatting")
			}

			entries = append(entries, ProvenanceSidecarEntry{
				ArtifactEntryID: ent.ID,
				ClaimIDs:        claimIDs,
				SourceObjectIDs: ent.SourceObjectIDs,
				EvidenceIDs:     ent.EvidenceIDs,
				Transformations: trans,
			})
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}
