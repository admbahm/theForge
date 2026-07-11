package planning

import (
	"strings"
	"time"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/model"
)

// ScoreClaim evaluates the relevance of a claim against a target profile.
func ScoreClaim(c claims.Claim, target *TargetProfile, inConflict bool) (int, map[string]int) {
	score := 50
	components := map[string]int{"Base Score": 50}

	if target == nil {
		return score, components
	}

	// 1. Skill Match (+15)
	hasSkillMatch := false
	for _, skill := range c.Skills {
		for _, targetSkill := range target.DesiredSkills {
			if strings.EqualFold(skill, targetSkill) {
				hasSkillMatch = true
				break
			}
		}
		for _, targetTech := range target.DesiredTechnologies {
			if strings.EqualFold(skill, targetTech) {
				hasSkillMatch = true
				break
			}
		}
	}

	// Case-insensitive text keyword scan
	stmtLower := strings.ToLower(c.Statement)
	valLower := strings.ToLower(string(c.Value))
	for _, targetSkill := range target.DesiredSkills {
		ts := strings.ToLower(targetSkill)
		if strings.Contains(stmtLower, ts) || strings.Contains(valLower, ts) {
			hasSkillMatch = true
			break
		}
	}
	for _, targetTech := range target.DesiredTechnologies {
		tt := strings.ToLower(targetTech)
		if strings.Contains(stmtLower, tt) || strings.Contains(valLower, tt) {
			hasSkillMatch = true
			break
		}
	}

	if hasSkillMatch {
		score += 15
		components["Skill Match Boost"] = 15
	}

	// 2. Role Match (+15)
	hasRoleMatch := false
	if target.RoleTitle != "" {
		rtLower := strings.ToLower(target.RoleTitle)
		if strings.Contains(stmtLower, rtLower) || strings.Contains(valLower, rtLower) {
			hasRoleMatch = true
		}
		if c.Kind == claims.KindRole && strings.Contains(strings.ToLower(string(c.Value)), rtLower) {
			hasRoleMatch = true
		}
	}
	if target.RoleFamily != "" {
		rfLower := strings.ToLower(target.RoleFamily)
		if strings.Contains(stmtLower, rfLower) || strings.Contains(valLower, rfLower) {
			hasRoleMatch = true
		}
	}
	if hasRoleMatch {
		score += 15
		components["Role Relevance Boost"] = 15
	}

	// 3. Recency Boost (+10)
	isRecent := false
	prefYears := target.RecencyPreferenceYears
	if prefYears == 0 {
		prefYears = 3 // default preference
	}

	if c.TimeRange != nil {
		end := c.TimeRange.End
		if end.IsZero() {
			// Ongoing / current is always recent
			isRecent = true
		} else {
			cutoff := time.Now().AddDate(-prefYears, 0, 0)
			if end.After(cutoff) {
				isRecent = true
			}
		}
	} else {
		// No time range is assumed historic or neutral
	}

	if isRecent {
		score += 10
		components["Recency Boost"] = 10
	}

	// 4. Evidence Strength (+10)
	if c.Verification == model.VerificationIndependentlyVerified || c.Verification == model.VerificationArtifactSupported {
		score += 10
		components["Evidence Verification Boost"] = 10
	}

	// 5. Private / Caution Penalty (-10)
	if c.Visibility == model.VisibilityConfidential || c.Visibility == model.VisibilityInternal {
		score -= 10
		components["Visibility Penalty"] = -10
	}

	// 6. Conflict Penalty (-20)
	if inConflict {
		score -= 20
		components["Conflict Penalty"] = -20
	}

	// Clamp to 0 - 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score, components
}
