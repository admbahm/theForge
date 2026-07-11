package rendering

import (
	"fmt"
	"sort"
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/planning"
)

// RenderResume processes the plan to build the OnePageResume structure.
func RenderResume(req RenderRequest, policy claims.Policy) ([]ArtifactSection, []string) {
	var sections []ArtifactSection
	var warnings []string

	selected := req.Plan.SelectedClaims

	// 1. Professional Summary Section
	var summaryTexts []string
	var summaryClaimIDs []claims.ClaimID
	var summarySrcIDs []string
	summaryCount := 0
	maxSummary := req.Options.MaxSummaryStatements
	if maxSummary <= 0 {
		maxSummary = 3
	}

	for _, pc := range selected {
		if pc.Section == "Summary" || pc.Claim.Kind == claims.KindCareerObjective {
			if summaryCount < maxSummary {
				stmt := pc.Claim.Statement
				// Avoid first person, vague filler words (rockstar, guru, results-driven)
				stmt = sanitizeFillerWords(stmt)
				summaryTexts = append(summaryTexts, stmt)
				summaryClaimIDs = append(summaryClaimIDs, pc.Claim.ID)
				summarySrcIDs = append(summarySrcIDs, pc.Claim.SourceObjectIDs...)
				summaryCount++
			}
		}
	}

	if len(summaryTexts) > 0 {
		sections = append(sections, ArtifactSection{
			ID:      "section:summary",
			Kind:    "Summary",
			Heading: "Professional Summary",
			Entries: []ArtifactEntry{
				{
					ID:              "entry:summary-paragraph",
					Kind:            "paragraph",
					Text:            strings.Join(summaryTexts, " "),
					ClaimIDs:        summaryClaimIDs,
					SourceObjectIDs: summarySrcIDs,
				},
			},
		})
	}

	// 2. Experience Section (with Promotion History Grouping)
	var expClaims []planning.PlannedClaim
	for _, pc := range selected {
		if pc.Section == "Experience" || pc.Claim.Kind == claims.KindRole || pc.Claim.Kind == claims.KindEmployment {
			expClaims = append(expClaims, pc)
		}
	}

	// Sort roles by date recency descending
	sort.Slice(expClaims, func(i, j int) bool {
		rI := expClaims[i].Claim.TimeRange
		rJ := expClaims[j].Claim.TimeRange
		if rI != nil && rJ != nil {
			return rI.Start.After(rJ.Start)
		}
		return expClaims[i].Claim.ID < expClaims[j].Claim.ID
	})

	// Group roles by organization to show promotion history
	type OrgGroup struct {
		Name  string
		Roles []planning.PlannedClaim
	}
	var orgGroups []OrgGroup
	orgIndex := make(map[string]int)

	for _, pc := range expClaims {
		orgName := "Other Experience"
		if len(pc.Claim.Organizations) > 0 {
			orgName = pc.Claim.Organizations[0]
		}
		idx, exists := orgIndex[orgName]
		if !exists {
			idx = len(orgGroups)
			orgGroups = append(orgGroups, OrgGroup{Name: orgName})
			orgIndex[orgName] = idx
		}
		orgGroups[idx].Roles = append(orgGroups[idx].Roles, pc)
	}

	var expEntries []ArtifactEntry
	roleIndex := 0

	for _, og := range orgGroups {
		// Output Organization Header
		expEntries = append(expEntries, ArtifactEntry{
			ID:   fmt.Sprintf("entry:experience-org-%d", roleIndex),
			Kind: "header",
			Text: og.Name,
		})

		for _, pc := range og.Roles {
			c := pc.Claim
			dates := ""
			if c.TimeRange != nil {
				startStr := c.TimeRange.Start.Format("2006-01")
				endStr := "Present"
				if !c.TimeRange.End.IsZero() {
					endStr = c.TimeRange.End.Format("2006-01")
				}
				dates = fmt.Sprintf(" (%s – %s)", startStr, endStr)
			}
			roleTitle := c.Statement
			if c.Kind == claims.KindEmployment {
				roleTitle = strings.TrimPrefix(roleTitle, "Employed at ")
			}

			// Output Role Title and Dates
			expEntries = append(expEntries, ArtifactEntry{
				ID:              fmt.Sprintf("entry:experience-role-%d", roleIndex),
				Kind:            "subheader",
				Text:            roleTitle + dates,
				ClaimIDs:        []claims.ClaimID{c.ID},
				SourceObjectIDs: c.SourceObjectIDs,
			})

			// Fetch accomplishments/achievements for this specific role
			achCount := 0
			maxAchievements := req.Plan.MaxAchievementsPerRole
			if maxAchievements <= 0 {
				maxAchievements = 4
			}

			for _, apc := range selected {
				ac := apc.Claim
				if apc.Section == "Achievements" || ac.Kind == claims.KindAccomplishment || ac.Kind == claims.KindMeasurableResult {
					// Check if this accomplishment belongs to this role
					belongs := false
					for _, srcID := range ac.SourceObjectIDs {
						for _, roleSrcID := range c.SourceObjectIDs {
							if srcID == roleSrcID {
								belongs = true
								break
							}
						}
					}
					if belongs {
						if achCount < maxAchievements {
							stmt := ac.Statement
							stmt = ExpandAbbreviations(stmt, req.Options.AbbreviationStyle)

							// Adapt metric format if compact style requested
							if req.Options.MetricStyle == "Compact" {
								stmt = formatStatementMetricsCompact(stmt, ac.Metrics)
							}

							// Strength preservation check
							if err := CheckStrengthPreservation(ac, stmt); err != nil {
								warnings = append(warnings, fmt.Sprintf("Claim %s: %v", ac.ID, err))
							}

							expEntries = append(expEntries, ArtifactEntry{
								ID:              fmt.Sprintf("entry:experience-role-%d-ach-%d", roleIndex, achCount),
								Kind:            "bullet",
								Text:            stmt,
								ClaimIDs:        []claims.ClaimID{ac.ID},
								EvidenceIDs:     ac.EvidenceObjectIDs,
								SourceObjectIDs: ac.SourceObjectIDs,
							})
							achCount++
						}
					}
				}
			}
			roleIndex++
		}
	}

	if len(expEntries) > 0 {
		sections = append(sections, ArtifactSection{
			ID:      "section:experience",
			Kind:    "Experience",
			Heading: "Professional Experience",
			Entries: expEntries,
		})
	}

	// 3. Core Skills Section
	var techSkills []claims.Claim
	var leadSkills []claims.Claim
	for _, pc := range selected {
		if pc.Section == "Skills" || pc.Claim.Kind == claims.KindTechnicalSkill || pc.Claim.Kind == claims.KindLeadershipSkill {
			if pc.Claim.Kind == claims.KindTechnicalSkill {
				techSkills = append(techSkills, pc.Claim)
			} else {
				leadSkills = append(leadSkills, pc.Claim)
			}
		}
	}

	var skillEntries []ArtifactEntry
	if len(techSkills) > 0 {
		combinedTech := CombineSkillStatements(techSkills, req.Options.AbbreviationStyle)
		skillEntries = append(skillEntries, ArtifactEntry{
			ID:              "entry:skills-technical",
			Kind:            "bullet",
			Text:            "Technical Skills: " + strings.TrimPrefix(combinedTech, "Utilized technologies: "),
			ClaimIDs:        claimIDsForClaims(techSkills),
			EvidenceIDs:     evidenceIDsForClaims(techSkills),
			SourceObjectIDs: sourceIDsForClaims(techSkills),
		})
	}
	if len(leadSkills) > 0 {
		combinedLead := CombineSkillStatements(leadSkills, req.Options.AbbreviationStyle)
		skillEntries = append(skillEntries, ArtifactEntry{
			ID:              "entry:skills-leadership",
			Kind:            "bullet",
			Text:            "Professional Competencies: " + strings.TrimPrefix(combinedLead, "Utilized technologies: "),
			ClaimIDs:        claimIDsForClaims(leadSkills),
			EvidenceIDs:     evidenceIDsForClaims(leadSkills),
			SourceObjectIDs: sourceIDsForClaims(leadSkills),
		})
	}

	if len(skillEntries) > 0 {
		sections = append(sections, ArtifactSection{
			ID:      "section:skills",
			Kind:    "Skills",
			Heading: "Core Skills",
			Entries: skillEntries,
		})
	}

	// 4. Education Section
	var eduEntries []ArtifactEntry
	eduIndex := 0
	for _, pc := range selected {
		if pc.Section == "Education" || pc.Claim.Kind == claims.KindEducation {
			eduEntries = append(eduEntries, ArtifactEntry{
				ID:              fmt.Sprintf("entry:education-%d", eduIndex),
				Kind:            "bullet",
				Text:            pc.Claim.Statement,
				ClaimIDs:        []claims.ClaimID{pc.Claim.ID},
				EvidenceIDs:     pc.Claim.EvidenceObjectIDs,
				SourceObjectIDs: pc.Claim.SourceObjectIDs,
			})
			eduIndex++
		}
	}
	if len(eduEntries) > 0 {
		sections = append(sections, ArtifactSection{
			ID:      "section:education",
			Kind:    "Education",
			Heading: "Education",
			Entries: eduEntries,
		})
	}

	// 5. Credentials / Certifications Section
	var credEntries []ArtifactEntry
	credIndex := 0
	for _, pc := range selected {
		if pc.Section == "Credentials" || pc.Claim.Kind == claims.KindCredential {
			credEntries = append(credEntries, ArtifactEntry{
				ID:              fmt.Sprintf("entry:credential-%d", credIndex),
				Kind:            "bullet",
				Text:            pc.Claim.Statement,
				ClaimIDs:        []claims.ClaimID{pc.Claim.ID},
				EvidenceIDs:     pc.Claim.EvidenceObjectIDs,
				SourceObjectIDs: pc.Claim.SourceObjectIDs,
			})
			credIndex++
		}
	}
	if len(credEntries) > 0 {
		sections = append(sections, ArtifactSection{
			ID:      "section:credentials",
			Kind:    "Credentials",
			Heading: "Certifications & Credentials",
			Entries: credEntries,
		})
	}

	return sections, warnings
}

func sanitizeFillerWords(text string) string {
	fillers := []string{"results-driven", "results-driven ", "rockstar", "guru", "expert-level"}
	res := text
	for _, f := range fillers {
		res = strings.ReplaceAll(res, f, "")
		res = strings.ReplaceAll(res, strings.Title(f), "")
	}
	// Clean double spaces
	res = strings.ReplaceAll(res, "  ", " ")
	return strings.TrimSpace(res)
}

func formatStatementMetricsCompact(statement string, metrics []claims.Metric) string {
	res := statement
	for _, m := range metrics {
		if m.IsAmbiguous || m.Status == "Ambiguous" || m.Status == "Target" || m.Status == "Estimate" {
			// Find approximate terms in the statement and replace with tilde
			res = strings.ReplaceAll(res, "approximately ", "~")
			res = strings.ReplaceAll(res, "approx ", "~")
			res = strings.ReplaceAll(res, "about ", "~")
		}
	}
	return res
}
