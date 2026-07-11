package rendering

import (
	"fmt"
	"sort"
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/planning"
)

// RenderCV processes the plan to build the comprehensive CV structure.
func RenderCV(req RenderRequest, policy claims.Policy) ([]ArtifactSection, []string) {
	var sections []ArtifactSection
	var warnings []string

	selected := req.Plan.SelectedClaims

	// 1. Professional Profile (Summary)
	var summaryEntries []ArtifactEntry
	var summaryClaimIDs []claims.ClaimID
	sumIndex := 0
	for _, pc := range selected {
		if pc.Section == "Summary" || pc.Claim.Kind == claims.KindCareerObjective {
			stmt := pc.Claim.Statement
			stmt = sanitizeFillerWords(stmt)
			summaryEntries = append(summaryEntries, ArtifactEntry{
				ID:              fmt.Sprintf("entry:cv-summary-%d", sumIndex),
				Kind:            "paragraph",
				Text:            stmt,
				ClaimIDs:        []claims.ClaimID{pc.Claim.ID},
				SourceObjectIDs: pc.Claim.SourceObjectIDs,
			})
			summaryClaimIDs = append(summaryClaimIDs, pc.Claim.ID)
			sumIndex++
		}
	}

	// 2. Experience Section
	var expClaims []planning.PlannedClaim
	for _, pc := range selected {
		if pc.Section == "Experience" || pc.Claim.Kind == claims.KindRole || pc.Claim.Kind == claims.KindEmployment {
			expClaims = append(expClaims, pc)
		}
	}

	sort.Slice(expClaims, func(i, j int) bool {
		rI := expClaims[i].Claim.TimeRange
		rJ := expClaims[j].Claim.TimeRange
		if rI != nil && rJ != nil {
			return rI.Start.After(rJ.Start)
		}
		return expClaims[i].Claim.ID < expClaims[j].Claim.ID
	})

	var expEntries []ArtifactEntry
	roleIndex := 0
	for _, pc := range expClaims {
		c := pc.Claim
		orgName := "Other Experience"
		if len(c.Organizations) > 0 {
			orgName = c.Organizations[0]
		}
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

		expEntries = append(expEntries, ArtifactEntry{
			ID:              fmt.Sprintf("entry:cv-experience-role-%d", roleIndex),
			Kind:            "subheader",
			Text:            fmt.Sprintf("%s at %s%s", roleTitle, orgName, dates),
			ClaimIDs:        []claims.ClaimID{c.ID},
			SourceObjectIDs: c.SourceObjectIDs,
		})

		achCount := 0
		for _, apc := range selected {
			ac := apc.Claim
			if apc.Section == "Achievements" || ac.Kind == claims.KindAccomplishment || ac.Kind == claims.KindMeasurableResult {
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
					stmt := ac.Statement
					stmt = ExpandAbbreviations(stmt, req.Options.AbbreviationStyle)

					if req.Options.MetricStyle == "Compact" {
						stmt = formatStatementMetricsCompact(stmt, ac.Metrics)
					}

					if err := CheckStrengthPreservation(ac, stmt); err != nil {
						warnings = append(warnings, fmt.Sprintf("Claim %s: %v", ac.ID, err))
					}

					expEntries = append(expEntries, ArtifactEntry{
						ID:              fmt.Sprintf("entry:cv-experience-role-%d-ach-%d", roleIndex, achCount),
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
		roleIndex++
	}

	// 3. Technical Skills
	var skillsList []claims.Claim
	for _, pc := range selected {
		if pc.Section == "Skills" || pc.Claim.Kind == claims.KindTechnicalSkill || pc.Claim.Kind == claims.KindLeadershipSkill {
			skillsList = append(skillsList, pc.Claim)
		}
	}

	var skillEntries []ArtifactEntry
	for sIdx, sc := range skillsList {
		skillClaims := []claims.Claim{sc}
		skillEntries = append(skillEntries, ArtifactEntry{
			ID:              fmt.Sprintf("entry:cv-skill-%d", sIdx),
			Kind:            "bullet",
			Text:            ExpandAbbreviations(sc.Statement, req.Options.AbbreviationStyle),
			ClaimIDs:        claimIDsForClaims(skillClaims),
			EvidenceIDs:     evidenceIDsForClaims(skillClaims),
			SourceObjectIDs: sourceIDsForClaims(skillClaims),
		})
	}

	// 4. Education
	var eduEntries []ArtifactEntry
	eduIndex := 0
	for _, pc := range selected {
		if pc.Section == "Education" || pc.Claim.Kind == claims.KindEducation {
			eduEntries = append(eduEntries, ArtifactEntry{
				ID:              fmt.Sprintf("entry:cv-education-%d", eduIndex),
				Kind:            "bullet",
				Text:            pc.Claim.Statement,
				ClaimIDs:        []claims.ClaimID{pc.Claim.ID},
				EvidenceIDs:     pc.Claim.EvidenceObjectIDs,
				SourceObjectIDs: pc.Claim.SourceObjectIDs,
			})
			eduIndex++
		}
	}

	// 5. Credentials / Certifications
	var credEntries []ArtifactEntry
	credIndex := 0
	for _, pc := range selected {
		if pc.Section == "Credentials" || pc.Claim.Kind == claims.KindCredential {
			credEntries = append(credEntries, ArtifactEntry{
				ID:              fmt.Sprintf("entry:cv-credential-%d", credIndex),
				Kind:            "bullet",
				Text:            pc.Claim.Statement,
				ClaimIDs:        []claims.ClaimID{pc.Claim.ID},
				EvidenceIDs:     pc.Claim.EvidenceObjectIDs,
				SourceObjectIDs: pc.Claim.SourceObjectIDs,
			})
			credIndex++
		}
	}

	// 6. Contributions (Publications, speaking, mentoring)
	var contribEntries []ArtifactEntry
	contribIndex := 0
	for _, pc := range selected {
		k := pc.Claim.Kind
		if pc.Section == "Contributions" || k == claims.KindPublication || k == claims.KindSpeaking || k == claims.KindMentoring {
			contribEntries = append(contribEntries, ArtifactEntry{
				ID:              fmt.Sprintf("entry:cv-contribution-%d", contribIndex),
				Kind:            "bullet",
				Text:            pc.Claim.Statement,
				ClaimIDs:        []claims.ClaimID{pc.Claim.ID},
				SourceObjectIDs: pc.Claim.SourceObjectIDs,
			})
			contribIndex++
		}
	}

	// Compile sections map
	sectionsMap := make(map[string]ArtifactSection)
	if len(summaryEntries) > 0 {
		sectionsMap["Summary"] = ArtifactSection{
			ID:      "section:cv-summary",
			Kind:    "Summary",
			Heading: "Professional Profile",
			Entries: summaryEntries,
		}
	}
	if len(expEntries) > 0 {
		sectionsMap["Experience"] = ArtifactSection{
			ID:      "section:cv-experience",
			Kind:    "Experience",
			Heading: "Detailed Experience",
			Entries: expEntries,
		}
	}
	if len(skillEntries) > 0 {
		sectionsMap["Skills"] = ArtifactSection{
			ID:      "section:cv-skills",
			Kind:    "Skills",
			Heading: "Technical & Professional Skills",
			Entries: skillEntries,
		}
	}
	if len(eduEntries) > 0 {
		sectionsMap["Education"] = ArtifactSection{
			ID:      "section:cv-education",
			Kind:    "Education",
			Heading: "Education History",
			Entries: eduEntries,
		}
	}
	if len(credEntries) > 0 {
		sectionsMap["Credentials"] = ArtifactSection{
			ID:      "section:cv-credentials",
			Kind:    "Credentials",
			Heading: "Professional Certifications",
			Entries: credEntries,
		}
	}
	if len(contribEntries) > 0 {
		sectionsMap["Contributions"] = ArtifactSection{
			ID:      "section:cv-contributions",
			Kind:    "Contributions",
			Heading: "Professional Contributions",
			Entries: contribEntries,
		}
	}

	// Order sections based on CV subtype
	var order []string
	switch req.Options.Subtype {
	case "technical":
		order = []string{"Summary", "Skills", "Experience", "Education", "Credentials", "Contributions"}
	case "executive":
		order = []string{"Summary", "Experience", "Skills", "Education", "Credentials", "Contributions"}
	case "academic-adjacent":
		order = []string{"Education", "Contributions", "Summary", "Experience", "Skills", "Credentials"}
	default: // "professional" or general standard
		order = []string{"Summary", "Experience", "Skills", "Education", "Credentials", "Contributions"}
	}

	for _, sName := range order {
		if sec, exists := sectionsMap[sName]; exists {
			sections = append(sections, sec)
		}
	}

	return sections, warnings
}
