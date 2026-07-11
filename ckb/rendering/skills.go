package rendering

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/planning"
)

// RenderSkillsSummary processes the plan to build the structured SkillsSummary artifact.
func RenderSkillsSummary(req RenderRequest, policy claims.Policy) ([]ArtifactSection, []string) {
	var sections []ArtifactSection
	var warnings []string

	selected := req.Plan.SelectedClaims

	// Group skills by category
	var techSkills []claims.Claim
	var leadSkills []claims.Claim

	for _, pc := range selected {
		if pc.Claim.Kind == claims.KindTechnicalSkill {
			techSkills = append(techSkills, pc.Claim)
		} else if pc.Claim.Kind == claims.KindLeadershipSkill {
			leadSkills = append(leadSkills, pc.Claim)
		}
	}

	renderSkillSection := func(skills []claims.Claim, kind string, heading string) ArtifactSection {
		var entries []ArtifactEntry
		for sIdx, sc := range skills {
			duration := CalculateSkillDuration(sc.Statement, selected)
			durationStr := ""
			if duration > 0 {
				durationStr = fmt.Sprintf(" (Used for %.1f months)", duration)
			}

			evidenceCount := len(sc.EvidenceObjectIDs)
			evidenceStr := ""
			if evidenceCount > 0 {
				evidenceStr = fmt.Sprintf(" [Backed by %d evidence source(s)]", evidenceCount)
			}

			text := fmt.Sprintf("%s%s%s", sc.Statement, durationStr, evidenceStr)
			skillClaims := []claims.Claim{sc}

			entries = append(entries, ArtifactEntry{
				ID:              fmt.Sprintf("entry:skills-summary-%s-%d", strings.ToLower(kind), sIdx),
				Kind:            "bullet",
				Text:            text,
				ClaimIDs:        claimIDsForClaims(skillClaims),
				EvidenceIDs:     evidenceIDsForClaims(skillClaims),
				SourceObjectIDs: sourceIDsForClaims(skillClaims),
			})
		}
		return ArtifactSection{
			ID:      "section:skills-summary-" + strings.ToLower(kind),
			Kind:    "SkillsSummary",
			Heading: heading,
			Entries: entries,
		}
	}

	if len(techSkills) > 0 {
		sections = append(sections, renderSkillSection(techSkills, "Technical", "Technical Skills & Competencies"))
	}
	if len(leadSkills) > 0 {
		sections = append(sections, renderSkillSection(leadSkills, "Leadership", "Methodologies & Leadership Competencies"))
	}

	return sections, warnings
}

// CalculateSkillDuration computes the non-overlapping timeline sum in months for a given skill.
func CalculateSkillDuration(skillName string, allSelected []planning.PlannedClaim) float64 {
	var ranges []claims.TimeRange
	referenceEnd := deterministicSkillReferenceEnd(allSelected)
	// Extract basic skill word (e.g. "Go (Golang)**" -> "go")
	skillClean := strings.ToLower(skillName)
	skillClean = strings.TrimSuffix(skillClean, "**")
	skillClean = strings.Split(skillClean, " ")[0] // Take first word for fuzzy match

	for _, pc := range allSelected {
		c := pc.Claim
		if (c.Kind == claims.KindRole || c.Kind == claims.KindEmployment) && c.TimeRange != nil {
			hasSkill := false
			// Check explicit skill lists
			for _, s := range c.Skills {
				if strings.Contains(strings.ToLower(s), skillClean) {
					hasSkill = true
					break
				}
			}
			// Check statement text
			if strings.Contains(strings.ToLower(c.Statement), skillClean) {
				hasSkill = true
			}
			if hasSkill {
				r := *c.TimeRange
				if r.End.IsZero() {
					r.End = referenceEnd
					if r.End.Before(r.Start) {
						r.End = r.Start
					}
				}
				ranges = append(ranges, r)
			}
		}
	}

	if len(ranges) == 0 {
		return 0
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start.Before(ranges[j].Start)
	})

	var unioned []claims.TimeRange
	curr := ranges[0]
	for i := 1; i < len(ranges); i++ {
		next := ranges[i]
		currEnd := curr.End
		if next.Start.Before(currEnd) || next.Start.Equal(currEnd) {
			nextEnd := next.End
			if nextEnd.After(currEnd) {
				curr.End = next.End
			}
		} else {
			unioned = append(unioned, curr)
			curr = next
		}
	}
	unioned = append(unioned, curr)

	var totalMonths float64
	for _, r := range unioned {
		months := r.End.Sub(r.Start).Hours() / (24 * 30)
		totalMonths += months
	}

	return totalMonths
}

func deterministicSkillReferenceEnd(allSelected []planning.PlannedClaim) time.Time {
	var latestEnd time.Time
	var latestStart time.Time

	for _, pc := range allSelected {
		if pc.Claim.TimeRange == nil {
			continue
		}
		r := pc.Claim.TimeRange
		if r.Start.After(latestStart) {
			latestStart = r.Start
		}
		if !r.End.IsZero() && r.End.After(latestEnd) {
			latestEnd = r.End
		}
	}

	if !latestEnd.IsZero() {
		return latestEnd
	}
	return latestStart
}
