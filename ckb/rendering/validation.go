package rendering

import (
	"fmt"
	"strings"

	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/planning"
)

// ValidatePlan checks the ArtifactPlan for security, completeness, and boundary violations.
func ValidatePlan(req RenderRequest) []model.Diagnostic {
	var diags []model.Diagnostic

	plan := req.Plan
	if plan == nil {
		diags = append(diags, model.Diagnostic{
			Code:     "CKB-RENDER-INVALID-PLAN",
			Severity: model.SeverityFatal,
			Message:  "Rendering failed: ArtifactPlan is nil",
			Source:   model.SourceLocation{FilePath: "renderer"},
		})
		return diags
	}

	if plan.SchemaVersion != "1.0" {
		diags = append(diags, model.Diagnostic{
			Code:     "CKB-RENDER-INVALID-PLAN",
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("Rendering failed: Unsupported plan schema version %q (expected 1.0)", plan.SchemaVersion),
			Source:   model.SourceLocation{FilePath: "renderer"},
		})
	}

	switch plan.ArtifactType {
	case planning.TypeResume, planning.TypeCV, planning.TypeBiography, planning.TypeSTARStory, planning.TypeSkillsSummary:
		// Supported
	default:
		diags = append(diags, model.Diagnostic{
			Code:     "CKB-RENDER-UNSUPPORTED-ARTIFACT",
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("Rendering failed: Unsupported artifact type %q", plan.ArtifactType),
			Source:   model.SourceLocation{FilePath: "renderer"},
		})
	}

	// Build map of override inclusions to resolve conflicts
	overrideMap := make(map[string]bool)
	for _, over := range req.Plan.Overrides {
		if over.Action == "Include" {
			overrideMap[string(over.ClaimID)] = true
		}
	}
	for _, over := range req.Plan.SelectedClaims {
		// Also look at plan-level overrides if passed
		_ = over
	}

	// Build map of active conflict claim IDs
	conflictClaims := make(map[string]string)
	for _, conf := range plan.Conflicts {
		if !conf.Blocked {
			diags = append(diags, model.Diagnostic{
				Code:     "CKB-RENDER-NONBLOCKING-CONFLICT",
				Severity: model.SeverityWarning,
				Message:  fmt.Sprintf("Non-blocking conflict retained for audit: %s", conf.Explanation),
				Source:   model.SourceLocation{FilePath: "renderer"},
				ObjectID: strings.Join(conf.ClaimIDs, ","),
			})
			continue
		}
		for _, cid := range conf.ClaimIDs {
			conflictClaims[cid] = conf.Explanation
		}
	}

	// Validate each selected claim
	for _, pc := range plan.SelectedClaims {
		c := pc.Claim

		// 1. Provenance completeness invariant
		if len(c.SourceObjectIDs) == 0 || len(c.SourceLocations) == 0 || c.SourceLocations[0].FilePath == "" || c.SourceLocations[0].FilePath == "unknown" {
			diags = append(diags, model.Diagnostic{
				Code:     "CKB-RENDER-MISSING-PROVENANCE",
				Severity: model.SeverityFatal,
				Message:  fmt.Sprintf("Provenance Incomplete: Selected claim %q is missing required source ID or file path", c.ID),
				Source:   model.SourceLocation{FilePath: "renderer"},
				ObjectID: string(c.ID),
			})
		}

		// 2. Section assignment check
		if pc.Section == "" {
			diags = append(diags, model.Diagnostic{
				Code:     "CKB-RENDER-INVALID-PLAN",
				Severity: model.SeverityError,
				Message:  fmt.Sprintf("Claim assignment error: Selected claim %q has no target section assigned in the plan", c.ID),
				Source:   model.SourceLocation{FilePath: "renderer"},
				ObjectID: string(c.ID),
			})
		}

		// 3. Visibility boundaries
		allowedVis := false
		for _, vis := range plan.Policy.AllowedVisibilities {
			if c.Visibility == vis {
				allowedVis = true
				break
			}
		}
		if !allowedVis {
			diags = append(diags, model.Diagnostic{
				Code:     "CKB-RENDER-VISIBILITY-VIOLATION",
				Severity: model.SeverityFatal,
				Message:  fmt.Sprintf("Visibility Violation: Selected claim %q has visibility %q which is prohibited by policy %q", c.ID, c.Visibility, plan.Policy.ID),
				Source:   model.SourceLocation{FilePath: "renderer"},
				ObjectID: string(c.ID),
			})
		}

		// 4. Blocking conflicts (unless explicitly resolved via override)
		if expl, inConflict := conflictClaims[string(c.ID)]; inConflict {
			if !overrideMap[string(c.ID)] {
				diags = append(diags, model.Diagnostic{
					Code:     "CKB-RENDER-BLOCKING-CONFLICT",
					Severity: model.SeverityFatal,
					Message:  fmt.Sprintf("Blocking Conflict: Claim %q has unresolved conflict and cannot render: %s", c.ID, expl),
					Source:   model.SourceLocation{FilePath: "renderer"},
					ObjectID: string(c.ID),
				})
			}
		}
	}

	return diags
}
