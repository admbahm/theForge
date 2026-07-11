package planning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/model"
)

var evidenceIDFormatRegex = regexp.MustCompile(`^ev:[a-z0-9-]+$`)

// PlanResult holds the assembled planning result and diagnostic logs.
type PlanResult struct {
	Plan        *ArtifactPlan
	Diagnostics []model.Diagnostic
}

// BuildPlan coordinates claim extraction, eligibility filters, scoring, budgeting, and gaps analysis.
func BuildPlan(ctx context.Context, kb *model.KnowledgeBase, request PlanRequest) PlanResult {
	res := PlanResult{
		Diagnostics: make([]model.Diagnostic, 0),
	}

	// 1. Context Cancellation Check
	select {
	case <-ctx.Done():
		res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
			Code:     model.CodeStructureMalformed,
			Severity: model.SeverityFatal,
			Message:  "Planning cancelled by context cancellation.",
			Source:   model.SourceLocation{FilePath: "planner"},
		})
		return res
	default:
	}

	// 2. Resolve Policy
	policy, ok := GetPolicyByID(request.PolicyID)
	if !ok {
		res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
			Code:     "CKB-PLAN-INVALID-POLICY",
			Severity: model.SeverityError,
			Message:  fmt.Sprintf("Invalid Policy: Policy %q is not defined in the planner registry", request.PolicyID),
			Source:   model.SourceLocation{FilePath: "planner"},
		})
		return res
	}

	// 3. Extract Claims from CKB Objects
	allClaims, err := claims.ExtractClaims(kb)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
			Code:     "CKB-CLAIM-EXTRACTION-FAILED",
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("Claims Extraction Failure: %v", err),
			Source:   model.SourceLocation{FilePath: "planner"},
		})
		return res
	}

	// 3b. Filter linked Evidence IDs based on policy allowed visibilities to prevent leakage
	for idx, c := range allClaims {
		authorizedClaim, evidenceDiags := authorizeEvidenceReferences(kb, policy, c)
		allClaims[idx] = authorizedClaim
		res.Diagnostics = append(res.Diagnostics, evidenceDiags...)
	}

	// 4. Normalization and Deduplication
	for i, c := range allClaims {
		allClaims[i] = claims.NormalizeClaim(c)
	}
	allClaims = claims.DeduplicateClaims(allClaims)

	// 5. Conflict Detection
	conflicts := claims.DetectConflicts(allClaims)
	conflictMap := make(map[string]bool)
	for _, conf := range conflicts {
		if conf.Blocked {
			for _, cid := range conf.ClaimIDs {
				conflictMap[cid] = true
			}
			// Trigger a warning/error diagnostic about the conflict
			res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
				Code:     "CKB-CLAIM-CONFLICT",
				Severity: model.SeverityError,
				Message:  fmt.Sprintf("Claim Conflict Block: %s (Claims: %s)", conf.Explanation, strings.Join(conf.ClaimIDs, ", ")),
				Source:   model.SourceLocation{FilePath: "planner"},
				ObjectID: strings.Join(conf.SourceObjectIDs, ", "),
			})
		}
	}

	// 6. Relevance Scoring
	scores := make(map[string]int)
	for _, c := range allClaims {
		sc, _ := ScoreClaim(c, request.Target, conflictMap[string(c.ID)])
		scores[string(c.ID)] = sc
	}

	// 7. Selection and Budgeting
	selected, excluded := SelectClaimsForArtifact(allClaims, request, scores, policy)

	// 8. Gap Analysis
	gaps := AnalyzeGaps(selected, request.Target)
	for _, gap := range gaps {
		res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
			Code:     model.DiagnosticCode(gap.Code),
			Severity: model.DiagnosticSeverity(gap.Severity),
			Message:  gap.Description,
			Source:   model.SourceLocation{FilePath: "planner"},
			Field:    gap.TargetField,
		})
	}

	// 9. Provenance Collection & Invariant Enforcement
	var provenance []ProvenanceRecord
	for _, pc := range selected {
		// Provenance Invariant Check
		if len(pc.Claim.SourceObjectIDs) == 0 || len(pc.Claim.SourceLocations) == 0 || pc.Claim.SourceLocations[0].FilePath == "" || pc.Claim.SourceLocations[0].FilePath == "unknown" {
			res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
				Code:     "CKB-PLAN-PROVENANCE-INCOMPLETE",
				Severity: model.SeverityFatal,
				Message:  fmt.Sprintf("Provenance Incomplete: Selected claim %q is missing required source object graph link or file path", pc.Claim.ID),
				Source:   model.SourceLocation{FilePath: "planner"},
			})
			return res
		}

		sourceFile := getCleanSourcePath(pc.Claim.SourceLocations[0].FilePath)
		sourceLine := pc.Claim.SourceLocations[0].Line
		provenance = append(provenance, ProvenanceRecord{
			ClaimID:      pc.Claim.ID,
			SourceFile:   sourceFile,
			SourceLine:   sourceLine,
			EvidenceIDs:  pc.Claim.EvidenceObjectIDs,
			Confidence:   pc.Claim.Confidence,
			Verification: pc.Claim.Verification,
		})
	}

	// 10. Generate deterministic Plan ID
	planID := generatePlanID(request)

	res.Plan = &ArtifactPlan{
		ID:                     planID,
		ArtifactType:           request.ArtifactType,
		Policy:                 policy,
		Target:                 request.Target,
		SelectedClaims:         selected,
		ExcludedClaims:         excluded,
		Conflicts:              conflicts,
		Gaps:                   gaps,
		Provenance:             provenance,
		Diagnostics:            res.Diagnostics,
		SchemaVersion:          "1.0",
		Overrides:              request.Overrides,
		MaxRoles:               request.MaxRoles,
		MaxAchievementsPerRole: request.MaxAchievementsPerRole,
		MaxSkills:              request.MaxSkills,
		MaxEducation:           request.MaxEducation,
	}

	return res
}

func authorizeEvidenceReferences(kb *model.KnowledgeBase, policy claims.Policy, c claims.Claim) (claims.Claim, []model.Diagnostic) {
	var diags []model.Diagnostic
	seen := make(map[string]bool)

	for _, evID := range c.EvidenceObjectIDs {
		if evID == "" || seen[evID] {
			continue
		}
		seen[evID] = true

		evObj, ok := kb.Objects[evID]
		switch {
		case !evidenceIDFormatRegex.MatchString(evID):
			diags = append(diags, evidenceAuthorizationDiagnostic(c, policy, "A malformed evidence reference was excluded from output."))
			continue
		case !ok:
			diags = append(diags, evidenceAuthorizationDiagnostic(c, policy, "A referenced evidence object could not be resolved and was excluded from output."))
			continue
		case evObj.Type != model.TypeEvidence:
			diags = append(diags, evidenceAuthorizationDiagnostic(c, policy, "A referenced evidence object resolved to an invalid object type and was excluded from output."))
			continue
		case !evidenceVisibilityAllowed(evObj.Metadata.Visibility, policy):
			continue
		case claims.VerificationRank(evObj.Metadata.Verification) < claims.VerificationRank(policy.MinVerification):
			diags = append(diags, evidenceAuthorizationDiagnostic(c, policy, "A referenced evidence object is below the policy verification threshold and was excluded from output."))
			continue
		case evObj.Metadata.Verification == model.VerificationDisputed || evObj.Metadata.Verification == model.VerificationSuperseded:
			diags = append(diags, evidenceAuthorizationDiagnostic(c, policy, "A referenced evidence object is not eligible for authorized provenance and was excluded from output."))
			continue
		case evObj.Metadata.Status == model.StatusDeprecated || evObj.Metadata.Status == model.StatusDraft:
			diags = append(diags, evidenceAuthorizationDiagnostic(c, policy, "A referenced evidence object is not active and was excluded from output."))
			continue
		case evObj.Metadata.Lifecycle == model.LifecycleArchived:
			diags = append(diags, evidenceAuthorizationDiagnostic(c, policy, "A referenced evidence object is archived and was excluded from output."))
			continue
		}
	}

	var authorized []string
	for evID := range seen {
		evObj, ok := kb.Objects[evID]
		if ok &&
			evidenceIDFormatRegex.MatchString(evID) &&
			evObj.Type == model.TypeEvidence &&
			evidenceVisibilityAllowed(evObj.Metadata.Visibility, policy) &&
			claims.VerificationRank(evObj.Metadata.Verification) >= claims.VerificationRank(policy.MinVerification) &&
			evObj.Metadata.Verification != model.VerificationDisputed &&
			evObj.Metadata.Verification != model.VerificationSuperseded &&
			evObj.Metadata.Status != model.StatusDeprecated &&
			evObj.Metadata.Status != model.StatusDraft &&
			evObj.Metadata.Lifecycle != model.LifecycleArchived {
			authorized = append(authorized, evID)
		}
	}
	sort.Strings(authorized)
	c.EvidenceObjectIDs = authorized
	return c, diags
}

func evidenceVisibilityAllowed(visibility model.Visibility, policy claims.Policy) bool {
	for _, allowed := range policy.AllowedVisibilities {
		if visibility == allowed {
			return true
		}
	}
	return false
}

func evidenceAuthorizationDiagnostic(c claims.Claim, policy claims.Policy, message string) model.Diagnostic {
	source := model.SourceLocation{FilePath: "planner"}
	if len(c.SourceLocations) > 0 {
		source = c.SourceLocations[0]
	}
	objectID := ""
	if len(c.SourceObjectIDs) > 0 {
		objectID = c.SourceObjectIDs[0]
	}
	return model.Diagnostic{
		Code:        model.CodeEvidenceUnresolved,
		Severity:    model.SeverityWarning,
		Message:     fmt.Sprintf("%s Policy=%s; claim remained subject to standard eligibility checks.", message, policy.ID),
		Source:      source,
		ObjectID:    objectID,
		Remediation: "Fix the evidence reference or add a resolvable Evidence catalog row with policy-permitted visibility before relying on it as provenance.",
	}
}

func generatePlanID(req PlanRequest) string {
	combined := fmt.Sprintf("%s|%s", req.ArtifactType, req.PolicyID)
	if req.Target != nil {
		combined += fmt.Sprintf("|%s|%s", req.Target.RoleTitle, req.Target.RoleFamily)
	}
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("plan:%s", hex.EncodeToString(hash[:])[:12])
}

func getCleanSourcePath(absPath string) string {
	idx := strings.Index(absPath, "ckb/")
	if idx != -1 {
		return absPath[idx:]
	}
	// Fallback to base name if ckb/ is not found
	idxSlash := strings.LastIndex(absPath, "/")
	if idxSlash != -1 {
		return absPath[idxSlash+1:]
	}
	return absPath
}
