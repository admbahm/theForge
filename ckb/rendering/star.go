package rendering

import (
	"fmt"
	"sort"
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/model"
)

// RenderSTARStory processes the plan to build STAR narrative blocks.
func RenderSTARStory(req RenderRequest, policy claims.Policy) ([]ArtifactSection, []model.Diagnostic, []string) {
	var sections []ArtifactSection
	var diags []model.Diagnostic
	var warnings []string

	selected := req.Plan.SelectedClaims
	behavior := req.Options.IncompleteSTARBehavior
	if behavior == "" {
		behavior = "WarnAndRender"
	}

	type planningClaim struct {
		id   claims.ClaimID
		text string
		srcs []string
		evs  []string
	}

	// 1. Group selected claims by parent source ID (e.g. project:failover, exp:stark-devops)
	starGroups := make(map[string]map[claims.ClaimKind][]planningClaim)

	for _, pc := range selected {
		k := pc.Claim.Kind
		if k == claims.KindSTARSituation || k == claims.KindSTARTask || k == claims.KindSTARAction || k == claims.KindSTARResult {
			srcID := "unknown"
			if len(pc.Claim.SourceObjectIDs) > 0 {
				srcID = pc.Claim.SourceObjectIDs[0]
			}
			if _, exists := starGroups[srcID]; !exists {
				starGroups[srcID] = make(map[claims.ClaimKind][]planningClaim)
			}
			starGroups[srcID][k] = append(starGroups[srcID][k], planningClaim{
				id:   pc.Claim.ID,
				text: pc.Claim.Statement,
				srcs: pc.Claim.SourceObjectIDs,
				evs:  pc.Claim.EvidenceObjectIDs,
			})
		}
	}

	// 2. Render each group
	groupCount := 0
	var sourceIDs []string
	for srcID := range starGroups {
		sourceIDs = append(sourceIDs, srcID)
	}
	sort.Strings(sourceIDs)

	for _, srcID := range sourceIDs {
		components := starGroups[srcID]
		sit := components[claims.KindSTARSituation]
		tsk := components[claims.KindSTARTask]
		act := components[claims.KindSTARAction]
		res := components[claims.KindSTARResult]

		var missing []string
		if len(sit) == 0 {
			missing = append(missing, "Situation")
		}
		if len(tsk) == 0 {
			missing = append(missing, "Task")
		}
		if len(act) == 0 {
			missing = append(missing, "Action")
		}
		if len(res) == 0 {
			missing = append(missing, "Result")
		}

		if len(missing) > 0 {
			if behavior == "Fail" {
				diags = append(diags, model.Diagnostic{
					Code:     "CKB-RENDER-INCOMPLETE-STAR",
					Severity: model.SeverityFatal,
					Message:  fmt.Sprintf("STAR Story Render Failure: Experience %q is missing components: %s", srcID, strings.Join(missing, ", ")),
					Source:   model.SourceLocation{FilePath: "renderer"},
				})
				return nil, diags, nil
			} else if behavior == "Skip" {
				diags = append(diags, model.Diagnostic{
					Code:     "CKB-RENDER-INCOMPLETE-STAR",
					Severity: model.SeverityWarning,
					Message:  fmt.Sprintf("STAR Story Skipped: Experience %q is missing components: %s", srcID, strings.Join(missing, ", ")),
					Source:   model.SourceLocation{FilePath: "renderer"},
				})
				continue
			} else {
				// WarnAndRender
				diags = append(diags, model.Diagnostic{
					Code:     "CKB-RENDER-INCOMPLETE-STAR",
					Severity: model.SeverityWarning,
					Message:  fmt.Sprintf("Incomplete STAR Story: Experience %q is missing components: %s", srcID, strings.Join(missing, ", ")),
					Source:   model.SourceLocation{FilePath: "renderer"},
				})
			}
		}

		// Subtype options (labeled, paragraph, bullet)
		style := req.Options.Subtype
		if style == "" {
			style = "labeled"
		}

		var entries []ArtifactEntry
		var allClaimIDs []claims.ClaimID
		var allEvIDs []string
		var allSrcIDs []string

		addComponents := func(comp []planningClaim, prefix string) {
			for _, pc := range comp {
				allClaimIDs = append(allClaimIDs, pc.id)
				allEvIDs = append(allEvIDs, pc.evs...)
				allSrcIDs = append(allSrcIDs, pc.srcs...)

				if style == "labeled" {
					entries = append(entries, ArtifactEntry{
						ID:              fmt.Sprintf("entry:star-%d-%s", groupCount, strings.ToLower(prefix)),
						Kind:            "bullet",
						Text:            fmt.Sprintf("%s: %s", prefix, pc.text),
						ClaimIDs:        []claims.ClaimID{pc.id},
						EvidenceIDs:     pc.evs,
						SourceObjectIDs: pc.srcs,
					})
				} else if style == "bullet" {
					entries = append(entries, ArtifactEntry{
						ID:              fmt.Sprintf("entry:star-%d-%s", groupCount, strings.ToLower(prefix)),
						Kind:            "bullet",
						Text:            pc.text,
						ClaimIDs:        []claims.ClaimID{pc.id},
						EvidenceIDs:     pc.evs,
						SourceObjectIDs: pc.srcs,
					})
				}
			}
		}

		if style == "paragraph" {
			var sentences []string
			for _, c := range [][]planningClaim{sit, tsk, act, res} {
				for _, pc := range c {
					sentences = append(sentences, pc.text)
					allClaimIDs = append(allClaimIDs, pc.id)
					allEvIDs = append(allEvIDs, pc.evs...)
					allSrcIDs = append(allSrcIDs, pc.srcs...)
				}
			}
			entries = append(entries, ArtifactEntry{
				ID:              fmt.Sprintf("entry:star-%d-paragraph", groupCount),
				Kind:            "paragraph",
				Text:            strings.Join(sentences, " "),
				ClaimIDs:        allClaimIDs,
				EvidenceIDs:     allEvIDs,
				SourceObjectIDs: allSrcIDs,
			})
		} else {
			// Labeled or Bullet list
			addComponents(sit, "Situation")
			addComponents(tsk, "Task")
			addComponents(act, "Action")
			addComponents(res, "Result")
		}

		sections = append(sections, ArtifactSection{
			ID:      fmt.Sprintf("section:star-story-%d", groupCount),
			Kind:    "STARStory",
			Heading: fmt.Sprintf("STAR Story - %s", srcID),
			Entries: entries,
		})
		groupCount++
	}

	return sections, diags, warnings
}
