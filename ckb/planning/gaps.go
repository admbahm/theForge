package planning

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/admbahm/theForge/ckb/claims"
)

// AnalyzeGaps scans selected planned claims and target profiles to identify gaps.
func AnalyzeGaps(selected []PlannedClaim, target *TargetProfile) []Gap {
	var gaps []Gap

	gaps = append(gaps, checkSkillGaps(selected, target)...)
	gaps = append(gaps, checkChronologyGaps(selected)...)
	gaps = append(gaps, checkMetricGaps(selected)...)
	gaps = append(gaps, checkSTARStoryCompleteness(selected)...)

	return gaps
}

func checkSkillGaps(selected []PlannedClaim, target *TargetProfile) []Gap {
	var gaps []Gap
	if target == nil {
		return nil
	}

	matchedSkills := make(map[string]bool)
	for _, pc := range selected {
		for _, sk := range pc.Claim.Skills {
			matchedSkills[strings.ToLower(sk)] = true
		}
		stmtLower := strings.ToLower(pc.Claim.Statement)
		valLower := strings.ToLower(string(pc.Claim.Value))
		for _, targetSkill := range target.DesiredSkills {
			ts := strings.ToLower(targetSkill)
			if strings.Contains(stmtLower, ts) || strings.Contains(valLower, ts) {
				matchedSkills[ts] = true
			}
		}
		for _, targetTech := range target.DesiredTechnologies {
			tt := strings.ToLower(targetTech)
			if strings.Contains(stmtLower, tt) || strings.Contains(valLower, tt) {
				matchedSkills[tt] = true
			}
		}
	}

	for _, ts := range target.DesiredSkills {
		if !matchedSkills[strings.ToLower(ts)] {
			gaps = append(gaps, Gap{
				Code:        "CKB-PLAN-TARGET-GAP",
				Severity:    "Warning",
				Description: fmt.Sprintf("Missing target skill: Desired skill %q is not backed by any eligible claim", ts),
				TargetField: "Skills",
				Context:     ts,
			})
		}
	}
	for _, tt := range target.DesiredTechnologies {
		if !matchedSkills[strings.ToLower(tt)] {
			gaps = append(gaps, Gap{
				Code:        "CKB-PLAN-TARGET-GAP",
				Severity:    "Warning",
				Description: fmt.Sprintf("Missing target technology: Desired technology %q is not backed by any eligible claim", tt),
				TargetField: "Technologies",
				Context:     tt,
			})
		}
	}
	return gaps
}

func checkChronologyGaps(selected []PlannedClaim) []Gap {
	var gaps []Gap
	var ranges []claims.TimeRange
	for _, pc := range selected {
		if (pc.Claim.Kind == claims.KindEmployment || pc.Claim.Kind == claims.KindRole) && pc.Claim.TimeRange != nil {
			ranges = append(ranges, *pc.Claim.TimeRange)
		}
	}

	if len(ranges) == 0 {
		return nil
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start.Before(ranges[j].Start)
	})

	var unioned []claims.TimeRange
	curr := ranges[0]
	for i := 1; i < len(ranges); i++ {
		next := ranges[i]
		currEnd := curr.End
		if currEnd.IsZero() {
			currEnd = time.Now()
		}
		if next.Start.Before(currEnd) || next.Start.Equal(currEnd) {
			nextEnd := next.End
			if nextEnd.IsZero() {
				nextEnd = time.Now()
			}
			if nextEnd.After(currEnd) {
				curr.End = next.End
			}
		} else {
			unioned = append(unioned, curr)
			curr = next
		}
	}
	unioned = append(unioned, curr)

	for i := 0; i < len(unioned)-1; i++ {
		currEnd := unioned[i].End
		if currEnd.IsZero() {
			continue
		}
		nextStart := unioned[i+1].Start
		gapDuration := nextStart.Sub(currEnd)
		if gapDuration > (30 * 24 * time.Hour * 6) { // > 6 months
			gaps = append(gaps, Gap{
				Code:        "CKB-PLAN-CHRONOLOGY-GAP",
				Severity:    "Warning",
				Description: fmt.Sprintf("Timeline interval: No selected employment activity recorded for a duration of %.1f months (%s to %s)", gapDuration.Hours()/(24*30), currEnd.Format("2006-01"), nextStart.Format("2006-01")),
				TargetField: "Timeline",
			})
		}
	}
	return gaps
}

func checkMetricGaps(selected []PlannedClaim) []Gap {
	var gaps []Gap
	for _, pc := range selected {
		if pc.Claim.Kind == claims.KindAccomplishment {
			gaps = append(gaps, Gap{
				Code:        "CKB-PLAN-METRIC-OPPORTUNITY",
				Severity:    "Warning",
				Description: fmt.Sprintf("Accomplishment opportunity: Outcome %q has no supporting quantifiable metrics", pc.Claim.Statement),
				TargetField: "Metrics",
				Context:     string(pc.Claim.ID),
			})
		}
	}
	return gaps
}

func checkSTARStoryCompleteness(selected []PlannedClaim) []Gap {
	var gaps []Gap
	// Group STAR claims by source experience/project ID
	starGroups := make(map[string]map[claims.ClaimKind]bool)
	for _, pc := range selected {
		k := pc.Claim.Kind
		if k == claims.KindSTARSituation || k == claims.KindSTARTask || k == claims.KindSTARAction || k == claims.KindSTARResult {
			srcID := "unknown"
			if len(pc.Claim.SourceObjectIDs) > 0 {
				srcID = pc.Claim.SourceObjectIDs[0]
			}
			if _, exists := starGroups[srcID]; !exists {
				starGroups[srcID] = make(map[claims.ClaimKind]bool)
			}
			starGroups[srcID][k] = true
		}
	}

	// For each STAR story group, verify completeness of STAR components
	for srcID, components := range starGroups {
		var missing []string
		if !components[claims.KindSTARSituation] {
			missing = append(missing, "Situation")
		}
		if !components[claims.KindSTARTask] {
			missing = append(missing, "Task")
		}
		if !components[claims.KindSTARAction] {
			missing = append(missing, "Action")
		}
		if !components[claims.KindSTARResult] {
			missing = append(missing, "Result")
		}

		if len(missing) > 0 {
			gaps = append(gaps, Gap{
				Code:        "CKB-PLAN-STAR-INCOMPLETE",
				Severity:    "Warning",
				Description: fmt.Sprintf("Incomplete STAR Story: Experience %q is missing components: %s", srcID, strings.Join(missing, ", ")),
				TargetField: "STAR",
				Context:     srcID,
			})
		}
	}

	return gaps
}
