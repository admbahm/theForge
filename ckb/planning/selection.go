package planning

import (
	"fmt"
	"sort"

	"github.com/admbahm/theForge/ckb/claims"
)

// SelectClaimsForArtifact filters and segments claims into the planned artifact sections.
func SelectClaimsForArtifact(allClaims []claims.Claim, req PlanRequest, scores map[string]int, policy claims.Policy) ([]PlannedClaim, []ExcludedClaim) {
	overrideMap := make(map[claims.ClaimID]Override)
	for _, o := range req.Overrides {
		overrideMap[o.ClaimID] = o
	}

	var candidates []claims.Claim
	var excluded []ExcludedClaim

	for _, c := range allClaims {
		status, code, msg := claims.EvaluateEligibility(c, policy)

		// Honor Exclude override
		over, hasOverride := overrideMap[c.ID]
		if hasOverride && over.Action == "Exclude" {
			excluded = append(excluded, ExcludedClaim{
				ClaimID:              c.ID,
				ExclusionReason:      "Excluded by human override",
				ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
				HumanOverrideAllowed: true,
			})
			continue
		}

		if status == claims.StatusIneligible {
			// Allow overriding unverified or low-confidence status, but never privacy
			if hasOverride && over.Action == "Include" && code != claims.CodeClaimPrivate {
				status = claims.StatusEligible
			} else {
				excluded = append(excluded, ExcludedClaim{
					ClaimID:              c.ID,
					ExclusionReason:      msg,
					ExclusionReasonCode:  code,
					HumanOverrideAllowed: code != claims.CodeClaimPrivate,
				})
				continue
			}
		}

		candidates = append(candidates, c)
	}

	// Sort candidates by score descending, then alphabetically by claim ID
	sort.Slice(candidates, func(i, j int) bool {
		sI := scores[string(candidates[i].ID)]
		sJ := scores[string(candidates[j].ID)]
		if sI != sJ {
			return sI > sJ
		}
		return candidates[i].ID < candidates[j].ID
	})

	var selected []PlannedClaim

	switch req.ArtifactType {
	case TypeResume:
		selected = selectForResume(candidates, req, scores, overrideMap, &excluded)
	case TypeCV:
		selected = selectForCV(candidates, req, scores, overrideMap, &excluded)
	case TypeBiography:
		selected = selectForBiography(candidates, req, scores, overrideMap, &excluded)
	case TypeSTARStory:
		selected = selectForSTAR(candidates, req, scores, overrideMap, &excluded)
	case TypeSkillsSummary:
		selected = selectForSkillsSummary(candidates, req, scores, overrideMap, &excluded)
	default:
		// Default CV style fallback
		selected = selectForCV(candidates, req, scores, overrideMap, &excluded)
	}

	// Assign ranks and label inactive claims
	for i := range selected {
		selected[i].Rank = i + 1

		c := selected[i].Claim
		isInactiveStatus := c.Status != "Active"
		isInactiveLifecycle := c.Lifecycle != "Active" && c.Lifecycle != "Completed"
		if isInactiveStatus || isInactiveLifecycle {
			label := ""
			if isInactiveStatus && isInactiveLifecycle {
				label = fmt.Sprintf("[INACTIVE: Status=%s, Lifecycle=%s]", c.Status, c.Lifecycle)
			} else if isInactiveStatus {
				label = fmt.Sprintf("[INACTIVE: Status=%s]", c.Status)
			} else {
				label = fmt.Sprintf("[INACTIVE: Lifecycle=%s]", c.Lifecycle)
			}
			selected[i].SelectionReason = fmt.Sprintf("%s %s", label, selected[i].SelectionReason)
			selected[i].Warnings = append(selected[i].Warnings, fmt.Sprintf("Claim is inactive: status=%s, lifecycle=%s", c.Status, c.Lifecycle))
		}
	}

	// Sort excluded claims for determinism
	sort.Slice(excluded, func(i, j int) bool {
		return excluded[i].ClaimID < excluded[j].ClaimID
	})

	return selected, excluded
}

func selectForResume(candidates []claims.Claim, req PlanRequest, scores map[string]int, overrides map[claims.ClaimID]Override, excluded *[]ExcludedClaim) []PlannedClaim {
	var selected []PlannedClaim
	roleCount := 0
	bulletsPerRole := make(map[string]int)
	skillCount := 0
	eduCount := 0

	maxRoles := req.MaxRoles
	if maxRoles <= 0 {
		maxRoles = 3
	}
	maxAchievements := req.MaxAchievementsPerRole
	if maxAchievements <= 0 {
		maxAchievements = 4
	}
	maxSkills := req.MaxSkills
	if maxSkills <= 0 {
		maxSkills = 10
	}
	maxEdu := req.MaxEducation
	if maxEdu <= 0 {
		maxEdu = 2
	}

	// Track selected roles to associate achievements
	selectedRoles := make(map[string]bool)
	roleSources := roleSourceIndex(candidates)
	employmentBySource := employmentClaimIndex(candidates)

	for _, c := range candidates {
		over, hasOver := overrides[c.ID]

		// Force include if override dictates
		if hasOver && over.Action == "Include" {
			sec := "General"
			if over.Section != "" {
				sec = over.Section
			}
			selected = append(selected, PlannedClaim{
				Claim:           c,
				SelectionReason: "Included via human override",
				RelevanceScore:  scores[string(c.ID)],
				Section:         sec,
			})
			continue
		}

		switch c.Kind {
		case claims.KindRole, claims.KindEmployment:
			srcID := primarySourceID(c)
			if srcID != "" && selectedRoles[srcID] {
				excludeDuplicateRoleAnchor(excluded, c)
				continue
			}
			if c.Kind == claims.KindEmployment && srcID != "" && roleSources[srcID] {
				excludeDuplicateRoleAnchor(excluded, c)
				continue
			}
			if roleCount < maxRoles {
				selectedClaim := c
				if c.Kind == claims.KindRole && srcID != "" {
					selectedClaim = mergeRoleAnchorClaim(c, employmentBySource[srcID])
				}
				selected = append(selected, PlannedClaim{
					Claim:           selectedClaim,
					SelectionReason: fmt.Sprintf("High-relevance chronology element (Score: %d)", scores[string(selectedClaim.ID)]),
					RelevanceScore:  scores[string(c.ID)],
					Section:         "Experience",
				})
				if srcID != "" {
					selectedRoles[srcID] = true
				}
				roleCount++
			} else {
				*excluded = append(*excluded, ExcludedClaim{
					ClaimID:              c.ID,
					ExclusionReason:      fmt.Sprintf("Omitted due to page limits (Max %d roles)", maxRoles),
					ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
					SectionConsidered:    "Experience",
					HumanOverrideAllowed: true,
				})
			}

		case claims.KindAccomplishment, claims.KindMeasurableResult:
			// Ensure it relates to a selected role/project
			hasParent := false
			parentID := ""
			for _, sid := range c.SourceObjectIDs {
				if selectedRoles[sid] {
					hasParent = true
					parentID = sid
					break
				}
			}
			if hasParent {
				if bulletsPerRole[parentID] < maxAchievements {
					selected = append(selected, PlannedClaim{
						Claim:           c,
						SelectionReason: fmt.Sprintf("High-relevance outcome (Score: %d)", scores[string(c.ID)]),
						RelevanceScore:  scores[string(c.ID)],
						Section:         "Achievements",
					})
					bulletsPerRole[parentID]++
				} else {
					*excluded = append(*excluded, ExcludedClaim{
						ClaimID:              c.ID,
						ExclusionReason:      fmt.Sprintf("Omitted due to page limits (Max %d accomplishments per role)", maxAchievements),
						ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
						SectionConsidered:    "Achievements",
						HumanOverrideAllowed: true,
					})
				}
			} else {
				*excluded = append(*excluded, ExcludedClaim{
					ClaimID:              c.ID,
					ExclusionReason:      "Omitted because the associated role was not selected",
					ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
					SectionConsidered:    "Achievements",
					HumanOverrideAllowed: true,
				})
			}

		case claims.KindTechnicalSkill, claims.KindLeadershipSkill:
			if skillCount < maxSkills {
				selected = append(selected, PlannedClaim{
					Claim:           c,
					SelectionReason: fmt.Sprintf("High-relevance technology/skill (Score: %d)", scores[string(c.ID)]),
					RelevanceScore:  scores[string(c.ID)],
					Section:         "Skills",
				})
				skillCount++
			} else {
				*excluded = append(*excluded, ExcludedClaim{
					ClaimID:              c.ID,
					ExclusionReason:      fmt.Sprintf("Omitted due to page limits (Max %d skills)", maxSkills),
					ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
					SectionConsidered:    "Skills",
					HumanOverrideAllowed: true,
				})
			}

		case claims.KindEducation:
			if eduCount < maxEdu {
				selected = append(selected, PlannedClaim{
					Claim:           c,
					SelectionReason: fmt.Sprintf("Academic credentials (Score: %d)", scores[string(c.ID)]),
					RelevanceScore:  scores[string(c.ID)],
					Section:         "Education",
				})
				eduCount++
			} else {
				*excluded = append(*excluded, ExcludedClaim{
					ClaimID:              c.ID,
					ExclusionReason:      fmt.Sprintf("Omitted due to page limits (Max %d education items)", maxEdu),
					ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
					SectionConsidered:    "Education",
					HumanOverrideAllowed: true,
				})
			}

		case claims.KindCredential:
			selected = append(selected, PlannedClaim{
				Claim:           c,
				SelectionReason: fmt.Sprintf("Professional credential (Score: %d)", scores[string(c.ID)]),
				RelevanceScore:  scores[string(c.ID)],
				Section:         "Credentials",
			})

		default:
			*excluded = append(*excluded, ExcludedClaim{
				ClaimID:              c.ID,
				ExclusionReason:      "Omitted due to one-page resume space limits",
				ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
				HumanOverrideAllowed: true,
			})
		}
	}
	return selected
}

func selectForCV(candidates []claims.Claim, req PlanRequest, scores map[string]int, overrides map[claims.ClaimID]Override, excluded *[]ExcludedClaim) []PlannedClaim {
	var selected []PlannedClaim
	selectedRoleSources := make(map[string]bool)
	roleSources := roleSourceIndex(candidates)
	employmentBySource := employmentClaimIndex(candidates)

	for _, c := range candidates {
		if c.Kind == claims.KindRole || c.Kind == claims.KindEmployment {
			srcID := primarySourceID(c)
			if srcID != "" && selectedRoleSources[srcID] {
				excludeDuplicateRoleAnchor(excluded, c)
				continue
			}
			if c.Kind == claims.KindEmployment && srcID != "" && roleSources[srcID] {
				excludeDuplicateRoleAnchor(excluded, c)
				continue
			}
			if c.Kind == claims.KindRole && srcID != "" {
				c = mergeRoleAnchorClaim(c, employmentBySource[srcID])
			}
			if srcID != "" {
				selectedRoleSources[srcID] = true
			}
		}

		sec := "Experience"
		switch c.Kind {
		case claims.KindEducation:
			sec = "Education"
		case claims.KindCredential:
			sec = "Credentials"
		case claims.KindTechnicalSkill, claims.KindLeadershipSkill:
			sec = "Skills"
		case claims.KindPublication, claims.KindSpeaking, claims.KindMentoring:
			sec = "Contributions"
		case claims.KindAccomplishment, claims.KindMeasurableResult:
			sec = "Accomplishments"
		}

		selected = append(selected, PlannedClaim{
			Claim:           c,
			SelectionReason: "Included in comprehensive CV timeline",
			RelevanceScore:  scores[string(c.ID)],
			Section:         sec,
		})
	}
	return selected
}

func primarySourceID(c claims.Claim) string {
	if len(c.SourceObjectIDs) == 0 {
		return ""
	}
	return c.SourceObjectIDs[0]
}

func roleSourceIndex(candidates []claims.Claim) map[string]bool {
	roleSources := make(map[string]bool)
	for _, c := range candidates {
		if c.Kind == claims.KindRole {
			if srcID := primarySourceID(c); srcID != "" {
				roleSources[srcID] = true
			}
		}
	}
	return roleSources
}

func employmentClaimIndex(candidates []claims.Claim) map[string]claims.Claim {
	employment := make(map[string]claims.Claim)
	for _, c := range candidates {
		if c.Kind == claims.KindEmployment {
			if srcID := primarySourceID(c); srcID != "" {
				employment[srcID] = c
			}
		}
	}
	return employment
}

func mergeRoleAnchorClaim(role claims.Claim, employment claims.Claim) claims.Claim {
	if role.TimeRange == nil && employment.TimeRange != nil {
		role.TimeRange = employment.TimeRange
	}
	if len(role.Organizations) == 0 && len(employment.Organizations) > 0 {
		role.Organizations = append([]string(nil), employment.Organizations...)
	}
	return role
}

func excludeDuplicateRoleAnchor(excluded *[]ExcludedClaim, c claims.Claim) {
	*excluded = append(*excluded, ExcludedClaim{
		ClaimID:              c.ID,
		ExclusionReason:      "Omitted because another role anchor from the same source was selected",
		ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
		SectionConsidered:    "Experience",
		HumanOverrideAllowed: true,
	})
}

func selectForBiography(candidates []claims.Claim, req PlanRequest, scores map[string]int, overrides map[claims.ClaimID]Override, excluded *[]ExcludedClaim) []PlannedClaim {
	var selected []PlannedClaim
	bioCount := 0
	achCount := 0

	for _, c := range candidates {
		switch c.Kind {
		case claims.KindRole, claims.KindEmployment, claims.KindCareerObjective:
			if bioCount < 2 {
				selected = append(selected, PlannedClaim{
					Claim:           c,
					SelectionReason: "Core narrative thread",
					RelevanceScore:  scores[string(c.ID)],
					Section:         "Summary",
				})
				bioCount++
			} else {
				*excluded = append(*excluded, ExcludedClaim{
					ClaimID:              c.ID,
					ExclusionReason:      "Biography limits reached (Max 2 roles)",
					ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
					SectionConsidered:    "Summary",
					HumanOverrideAllowed: true,
				})
			}
		case claims.KindAccomplishment, claims.KindMeasurableResult, claims.KindAward:
			if achCount < 3 {
				selected = append(selected, PlannedClaim{
					Claim:           c,
					SelectionReason: "Notable career milestone",
					RelevanceScore:  scores[string(c.ID)],
					Section:         "Highlights",
				})
				achCount++
			} else {
				*excluded = append(*excluded, ExcludedClaim{
					ClaimID:              c.ID,
					ExclusionReason:      "Biography milestone limits reached (Max 3 highlights)",
					ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
					SectionConsidered:    "Highlights",
					HumanOverrideAllowed: true,
				})
			}
		default:
			*excluded = append(*excluded, ExcludedClaim{
				ClaimID:              c.ID,
				ExclusionReason:      "Omitted due to narrative focus constraints",
				ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
				HumanOverrideAllowed: true,
			})
		}
	}
	return selected
}

func selectForSTAR(candidates []claims.Claim, req PlanRequest, scores map[string]int, overrides map[claims.ClaimID]Override, excluded *[]ExcludedClaim) []PlannedClaim {
	var selected []PlannedClaim
	for _, c := range candidates {
		if c.Kind == claims.KindSTARSituation || c.Kind == claims.KindSTARTask || c.Kind == claims.KindSTARAction || c.Kind == claims.KindSTARResult || c.Kind == claims.KindProjectContribution || c.Kind == claims.KindAccomplishment || c.Kind == claims.KindMeasurableResult {
			selected = append(selected, PlannedClaim{
				Claim:           c,
				SelectionReason: "STAR narrative block element",
				RelevanceScore:  scores[string(c.ID)],
				Section:         "STAR",
			})
		} else {
			*excluded = append(*excluded, ExcludedClaim{
				ClaimID:              c.ID,
				ExclusionReason:      "Omitted because it is not a direct STAR candidate",
				ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
				HumanOverrideAllowed: true,
			})
		}
	}
	return selected
}

func selectForSkillsSummary(candidates []claims.Claim, req PlanRequest, scores map[string]int, overrides map[claims.ClaimID]Override, excluded *[]ExcludedClaim) []PlannedClaim {
	var selected []PlannedClaim
	for _, c := range candidates {
		if c.Kind == claims.KindTechnicalSkill || c.Kind == claims.KindLeadershipSkill {
			selected = append(selected, PlannedClaim{
				Claim:           c,
				SelectionReason: "Skill mapping node",
				RelevanceScore:  scores[string(c.ID)],
				Section:         "Skills",
			})
		} else {
			*excluded = append(*excluded, ExcludedClaim{
				ClaimID:              c.ID,
				ExclusionReason:      "Non-skill node omitted from skills inventory plan",
				ExclusionReasonCode:  "CKB-CLAIM-OUTSIDE-TARGET-SCOPE",
				HumanOverrideAllowed: true,
			})
		}
	}
	return selected
}
