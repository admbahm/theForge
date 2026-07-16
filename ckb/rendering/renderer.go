package rendering

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/planning"
)

// RenderResult represents the outcome of the rendering process.
type RenderResult struct {
	Artifact    *Artifact          `json:"artifact,omitempty"`
	Diagnostics []model.Diagnostic `json:"diagnostics,omitempty"`
}

// Render compiles an authorized planning into a structured intermediate artifact document.
func Render(ctx context.Context, req RenderRequest) RenderResult {
	// 1. Run Plan Validations
	diags := ValidatePlan(req)
	for _, d := range diags {
		if d.Severity == model.SeverityFatal {
			return RenderResult{Diagnostics: diags}
		}
	}

	plan := req.Plan
	var sections []ArtifactSection
	var warnings []string

	// 2. Select corresponding renderer
	switch plan.ArtifactType {
	case planning.TypeResume:
		sections, warnings = RenderResume(req, plan.Policy)
	case planning.TypeCV:
		sections, warnings = RenderCV(req, plan.Policy)
	case planning.TypeBiography:
		sections, warnings = RenderBiography(req, plan.Policy)
	case planning.TypeSTARStory:
		var starDiags []model.Diagnostic
		sections, starDiags, warnings = RenderSTARStory(req, plan.Policy)
		diags = append(diags, starDiags...)
		for _, sd := range starDiags {
			if sd.Severity == model.SeverityFatal {
				return RenderResult{Diagnostics: diags}
			}
		}
	case planning.TypeSkillsSummary:
		sections, warnings = RenderSkillsSummary(req, plan.Policy)
	case planning.TypeCoverLetter:
		sections, warnings = RenderCoverLetter(req, plan.Policy)
	default:
		// CV default fallback
		sections, warnings = RenderCV(req, plan.Policy)
	}

	// 3. Assemble Audit Manifest
	var renderedEntryIDs []string
	claimToEntryMap := make(map[claims.ClaimID][]string)
	seenSrcs := make(map[string]bool)
	seenEvs := make(map[string]bool)

	for _, sec := range sections {
		for _, ent := range sec.Entries {
			renderedEntryIDs = append(renderedEntryIDs, ent.ID)
			for _, cid := range ent.ClaimIDs {
				claimToEntryMap[cid] = append(claimToEntryMap[cid], ent.ID)
			}
			for _, sid := range ent.SourceObjectIDs {
				seenSrcs[sid] = true
			}
			for _, eid := range ent.EvidenceIDs {
				seenEvs[eid] = true
			}
		}
	}

	var selectedClaimIDs []claims.ClaimID
	for _, pc := range plan.SelectedClaims {
		selectedClaimIDs = append(selectedClaimIDs, pc.Claim.ID)
	}

	var sourceRefs []string
	for k := range seenSrcs {
		sourceRefs = append(sourceRefs, k)
	}
	var evidenceRefs []string
	for k := range seenEvs {
		evidenceRefs = append(evidenceRefs, k)
	}

	// Sort references for determinism
	sortStrings(sourceRefs)
	sortStrings(evidenceRefs)

	contentDigest := computeContentDigest(sections)
	artifactID := fmt.Sprintf("artifact:plan-%s", plan.ID)

	manifest := ArtifactManifest{
		ArtifactID:         artifactID,
		ArtifactType:       plan.ArtifactType,
		PlanID:             plan.ID,
		PlannerVersion:     "1.0",
		RendererVersion:    "1.0",
		RenderingPolicy:    plan.Policy.ID,
		SelectedClaimIDs:   selectedClaimIDs,
		ExcludedClaimCount: len(plan.ExcludedClaims),
		RenderedEntryIDs:   renderedEntryIDs,
		ClaimToEntryMap:    claimToEntryMap,
		SourceReferences:   sourceRefs,
		EvidenceReferences: evidenceRefs,
		Warnings:           warnings,
		Overrides:          plan.Overrides,
		ContentDigest:      contentDigest,
	}

	// Supply default titles if options contact name is present
	title := "Career Portfolio"
	if req.Options.Contact.Name != "" {
		title = req.Options.Contact.Name
	}
	subtitle := string(plan.ArtifactType)

	// Audit all rendered entries for Strength Upgrades
	var strengthDiags []model.Diagnostic
	for _, sec := range sections {
		for _, ent := range sec.Entries {
			for _, cid := range ent.ClaimIDs {
				var matchingClaim *claims.Claim
				for _, pc := range plan.SelectedClaims {
					if pc.Claim.ID == cid {
						matchingClaim = &pc.Claim
						break
					}
				}
				if matchingClaim != nil {
					if err := CheckStrengthPreservation(*matchingClaim, ent.Text); err != nil {
						severity := model.SeverityWarning
						if plan.ArtifactType == planning.TypeResume || plan.ArtifactType == planning.TypeCV || plan.ArtifactType == planning.TypeBiography {
							severity = model.SeverityError
						}
						strengthDiags = append(strengthDiags, model.Diagnostic{
							Code:     "CKB-RENDER-STRENGTH-UPGRADE",
							Severity: severity,
							Message:  fmt.Sprintf("Strength upgrade detected on claim %s: %v", cid, err),
							Source:   model.SourceLocation{FilePath: "renderer"},
						})
					}
				}
			}
		}
	}
	diags = append(diags, strengthDiags...)

	// If there are any Error or Fatal diagnostics, return nil Artifact
	hasBlockingDiag := false
	for _, d := range diags {
		if d.Severity == model.SeverityError || d.Severity == model.SeverityFatal {
			hasBlockingDiag = true
			break
		}
	}

	var art *Artifact
	if !hasBlockingDiag {
		art = &Artifact{
			ID:            ArtifactID(artifactID),
			Type:          plan.ArtifactType,
			SchemaVersion: "1.0",
			Title:         title,
			Subtitle:      subtitle,
			Sections:      sections,
			Manifest:      manifest,
			Diagnostics:   diags,
		}
	}

	return RenderResult{
		Artifact:    art,
		Diagnostics: diags,
	}
}

func computeContentDigest(sections []ArtifactSection) string {
	var sb strings.Builder
	for _, sec := range sections {
		sb.WriteString(sec.Heading)
		for _, ent := range sec.Entries {
			sb.WriteString(ent.Text)
		}
	}
	hash := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(hash[:])
}

func sortStrings(arr []string) {
	for i := 0; i < len(arr); i++ {
		for j := i + 1; j < len(arr); j++ {
			if arr[i] > arr[j] {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
}
