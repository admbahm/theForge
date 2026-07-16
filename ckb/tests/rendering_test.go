package tests

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/export"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/parser"
	"github.com/admbahm/theForge/ckb/planning"
	"github.com/admbahm/theForge/ckb/rendering"
)

func TestArtifactRendering(t *testing.T) {
	// 1. Setup mock KnowledgeBase
	kb := model.NewKnowledgeBase()
	obj1 := &model.Object{
		ID:         "exp:stark-devops",
		Type:       model.TypeExperience,
		SourceFile: "/Users/example/dev/example-project/ckb/experience/stark-devops.md",
		Metadata: model.Metadata{
			Visibility:   model.VisibilityPublic,
			Verification: model.VerificationIndependentlyVerified,
			Source:       "Stark Industries",
			Confidence:   0.90,
		},
		Sections: []model.Section{
			{
				Heading: "Role Context",
				Body:    "Organization: Stark Industries\nRole: Principal DevOps Architect\nDuration: 2025-06 - 2026-06\nLocation: Remote",
			},
			{
				Heading: "Achievements",
				Body:    "- Cost Reduction: Saved $1.2M in annual cloud spend by implementing autoscaling configurations.\n- Velocity: Reduced onboarding from 4 days to 30 minutes.",
			},
		},
	}
	obj2 := &model.Object{
		ID:         "skill:go",
		Type:       model.TypeSkill,
		SourceFile: "/Users/example/dev/example-project/ckb/skills/go.md",
		Metadata: model.Metadata{
			Visibility:   model.VisibilityPublic,
			Verification: model.VerificationIndependentlyVerified,
			Confidence:   0.90,
		},
		Sections: []model.Section{
			{
				Heading: "Metadata",
				Body:    "Name: Go\nCategory: Languages",
			},
		},
	}
	kb.Objects[obj1.ID] = obj1
	kb.Objects[obj2.ID] = obj2

	// Build plan
	req := planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     "StrictPublic",
		Target: &planning.TargetProfile{
			RoleTitle: "DevOps Engineer",
		},
	}
	planRes := planning.BuildPlan(context.Background(), kb, req)
	if planRes.Plan == nil {
		t.Fatalf("Plan construction failed: %+v", planRes.Diagnostics)
	}

	// 2. Render Resume
	renderReq := rendering.RenderRequest{
		Plan: planRes.Plan,
		Options: rendering.RenderOptions{
			Subtype:   "technical",
			DateStyle: "YearOnly",
			Contact: rendering.ContactInfo{
				Name:  "Tony Stark",
				Email: "tony@stark.com",
			},
		},
	}

	res := rendering.Render(context.Background(), renderReq)
	if len(res.Diagnostics) > 0 {
		for _, d := range res.Diagnostics {
			if d.Severity == model.SeverityFatal {
				t.Fatalf("Fatal rendering diagnostic: %s", d.Message)
			}
		}
	}

	art := res.Artifact
	if art == nil {
		t.Fatalf("Artifact rendering returned nil artifact")
	}

	if art.Title != "Tony Stark" {
		t.Errorf("Expected title 'Tony Stark', got %q", art.Title)
	}

	// Verify Experience section is present
	foundExp := false
	for _, sec := range art.Sections {
		if sec.Kind == "Experience" {
			foundExp = true
			if len(sec.Entries) < 2 {
				t.Errorf("Expected organization and role entries in experience, got %d", len(sec.Entries))
			}
		}
	}
	if !foundExp {
		t.Error("Expected 'Experience' section in rendered resume, not found")
	}

	// 3. Export Markdown
	var mdBuf bytes.Buffer
	err := export.ExportMarkdown(art, &mdBuf, export.MarkdownOptions{IncludeHeadings: true})
	if err != nil {
		t.Fatalf("Markdown export failed: %v", err)
	}
	mdStr := mdBuf.String()

	if !strings.Contains(mdStr, "# Tony Stark") {
		t.Errorf("Expected Markdown to contain title '# Tony Stark', got %q", mdStr)
	}
	if !strings.Contains(mdStr, "Stark Industries") {
		t.Error("Expected Markdown to contain organization name 'Stark Industries'")
	}

	// 4. Export Text
	var textBuf bytes.Buffer
	err = export.ExportText(art, &textBuf, export.TextOptions{IncludeHeadings: true})
	if err != nil {
		t.Fatalf("Text export failed: %v", err)
	}
	textStr := textBuf.String()

	if strings.Contains(textStr, "**") {
		t.Errorf("Plain text export contains Markdown bold artifacts: %q", textStr)
	}

	// 5. Export Deterministic JSON and verify content digest
	var jsonBuf bytes.Buffer
	err = export.ExportArtifactJSON(art, &jsonBuf)
	if err != nil {
		t.Fatalf("JSON export failed: %v", err)
	}
	if art.Manifest.ContentDigest == "" {
		t.Error("Manifest content digest is empty")
	}

	// 6. Export Provenance Sidecar
	var sidecarBuf bytes.Buffer
	err = export.ExportProvenanceSidecar(art, &sidecarBuf)
	if err != nil {
		t.Fatalf("Sidecar export failed: %v", err)
	}
	sidecarStr := sidecarBuf.String()
	if !strings.Contains(sidecarStr, "whitespace-normalization") {
		t.Error("Sidecar transformations missing whitespace-normalization record")
	}
}

func TestExperienceClaimsPopulateExplicitOrganization(t *testing.T) {
	obj := experienceObjectForOrganization("exp:org", "Synthetic Test Fixture", "Example Systems")
	extracted, err := claims.ExtractClaims(&model.KnowledgeBase{
		Objects: map[string]*model.Object{obj.ID: obj},
	})
	if err != nil {
		t.Fatalf("ExtractClaims failed: %v", err)
	}

	foundRole := false
	foundEmployment := false
	for _, c := range extracted {
		switch c.Kind {
		case claims.KindRole:
			foundRole = true
			if len(c.Organizations) != 1 || c.Organizations[0] != "Example Systems" {
				t.Fatalf("Expected role claim organization from explicit Role Context field, got %+v", c.Organizations)
			}
			if strings.Contains(c.Statement, "Synthetic Test Fixture") {
				t.Fatalf("Role statement used generic metadata source as employer: %q", c.Statement)
			}
		case claims.KindEmployment:
			foundEmployment = true
			if len(c.Organizations) != 1 || c.Organizations[0] != "Example Systems" {
				t.Fatalf("Expected employment claim organization from explicit Role Context field, got %+v", c.Organizations)
			}
		}
	}
	if !foundRole || !foundEmployment {
		t.Fatalf("Expected role and employment claims, got role=%t employment=%t claims=%+v", foundRole, foundEmployment, extracted)
	}
}

func TestExperienceWithoutOrganizationFallsBackSafely(t *testing.T) {
	obj := experienceObjectForOrganization("exp:no-org", "Synthetic Test Fixture", "")
	extracted, err := claims.ExtractClaims(&model.KnowledgeBase{
		Objects: map[string]*model.Object{obj.ID: obj},
	})
	if err != nil {
		t.Fatalf("ExtractClaims failed: %v", err)
	}

	var role *claims.Claim
	for idx := range extracted {
		if extracted[idx].Kind == claims.KindRole {
			role = &extracted[idx]
			break
		}
	}
	if role == nil {
		t.Fatalf("Expected role claim, got %+v", extracted)
	}
	if len(role.Organizations) != 0 {
		t.Fatalf("Missing explicit organization should not populate generic metadata source, got %+v", role.Organizations)
	}
	if strings.Contains(role.Statement, "Synthetic Test Fixture") {
		t.Fatalf("Missing explicit organization should not render generic metadata source as employer: %q", role.Statement)
	}

	res := rendering.Render(context.Background(), rendering.RenderRequest{Plan: resumePlanWithClaims([]claims.Claim{*role})})
	if res.Artifact == nil {
		t.Fatalf("Expected resume render with missing organization to succeed, got diagnostics: %+v", res.Diagnostics)
	}
	if !experienceHeaderTexts(res.Artifact)["Other Experience"] {
		t.Fatalf("Expected missing organization to fall back to Other Experience, got headers %+v", experienceHeaderTexts(res.Artifact))
	}
	if experienceHeaderTexts(res.Artifact)["Synthetic Test Fixture"] {
		t.Fatal("Generic metadata provenance was rendered as employer header")
	}
}

func TestResumeOrganizationGrouping(t *testing.T) {
	sameOrgPlan := resumePlanWithClaims([]claims.Claim{
		resumeRoleClaim("claim:role:one", "Principal Engineer", "Example Systems", "exp:one"),
		resumeRoleClaim("claim:role:two", "Staff Engineer", "Example Systems", "exp:two"),
	})
	sameOrgRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: sameOrgPlan})
	if sameOrgRes.Artifact == nil {
		t.Fatalf("Expected same-org resume render, got diagnostics: %+v", sameOrgRes.Diagnostics)
	}
	if countExperienceHeader(sameOrgRes.Artifact, "Example Systems") != 1 {
		t.Fatalf("Expected consecutive promotions at same org to share one header, got headers %+v", experienceHeaderTexts(sameOrgRes.Artifact))
	}

	differentOrgPlan := resumePlanWithClaims([]claims.Claim{
		resumeRoleClaim("claim:role:alpha", "Platform Engineer", "Alpha Systems", "exp:alpha"),
		resumeRoleClaim("claim:role:beta", "Infrastructure Engineer", "Beta Labs", "exp:beta"),
	})
	differentOrgRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: differentOrgPlan})
	if differentOrgRes.Artifact == nil {
		t.Fatalf("Expected different-org resume render, got diagnostics: %+v", differentOrgRes.Diagnostics)
	}
	headers := experienceHeaderTexts(differentOrgRes.Artifact)
	if !headers["Alpha Systems"] || !headers["Beta Labs"] {
		t.Fatalf("Expected different organizations to render separate headers, got %+v", headers)
	}
}

func TestResumeSkillEntriesCarryUnionedProvenance(t *testing.T) {
	skillA := skillClaim("claim:skill:a", claims.KindTechnicalSkill, "Utilized technologies: Go", []string{"skill:b", "skill:a", "skill:a"}, []string{"ev:b", "ev:a", "ev:a"})
	skillB := skillClaim("claim:skill:b", claims.KindTechnicalSkill, "Utilized technologies: Kubernetes", []string{"skill:c"}, []string{"ev:c", "ev:a"})
	leadership := skillClaim("claim:skill:lead", claims.KindLeadershipSkill, "Utilized technologies: Mentoring", []string{"skill:lead"}, []string{"ev:public"})

	plan := skillArtifactPlan(planning.TypeResume, []claims.Claim{skillB, leadership, skillA})
	first := rendering.Render(context.Background(), rendering.RenderRequest{Plan: plan})
	second := rendering.Render(context.Background(), rendering.RenderRequest{Plan: plan})
	if first.Artifact == nil || second.Artifact == nil {
		t.Fatalf("Expected rendered artifacts, got diagnostics: %+v / %+v", first.Diagnostics, second.Diagnostics)
	}

	techEntry := findArtifactEntry(t, first.Artifact, "entry:skills-technical")
	assertClaimIDsEqual(t, techEntry.ClaimIDs, []claims.ClaimID{"claim:skill:a", "claim:skill:b"})
	assertStringsEqual(t, techEntry.SourceObjectIDs, []string{"skill:a", "skill:b", "skill:c"})
	assertStringsEqual(t, techEntry.EvidenceIDs, []string{"ev:a", "ev:b", "ev:c"})

	leadEntry := findArtifactEntry(t, first.Artifact, "entry:skills-leadership")
	assertStringsEqual(t, leadEntry.EvidenceIDs, []string{"ev:public"})
	if containsString(leadEntry.EvidenceIDs, "ev:restricted") {
		t.Fatalf("Restricted evidence unexpectedly appeared in skill provenance: %+v", leadEntry.EvidenceIDs)
	}

	assertStringsEqual(t, first.Artifact.Manifest.SourceReferences, []string{"skill:a", "skill:b", "skill:c", "skill:lead"})
	assertStringsEqual(t, first.Artifact.Manifest.EvidenceReferences, []string{"ev:a", "ev:b", "ev:c", "ev:public"})

	var firstSidecar, secondSidecar bytes.Buffer
	if err := export.ExportProvenanceSidecar(first.Artifact, &firstSidecar); err != nil {
		t.Fatalf("first sidecar export failed: %v", err)
	}
	if err := export.ExportProvenanceSidecar(second.Artifact, &secondSidecar); err != nil {
		t.Fatalf("second sidecar export failed: %v", err)
	}
	if !bytes.Equal(firstSidecar.Bytes(), secondSidecar.Bytes()) {
		t.Fatalf("Expected repeated sidecar exports to be byte-identical\nfirst=%s\nsecond=%s", firstSidecar.String(), secondSidecar.String())
	}
	for _, expected := range []string{"entry:skills-technical", "claim:skill:a", "skill:a", "ev:a"} {
		if !strings.Contains(firstSidecar.String(), expected) {
			t.Fatalf("Expected sidecar to include %q, got %s", expected, firstSidecar.String())
		}
	}

	var firstArtifactJSON, secondArtifactJSON bytes.Buffer
	if err := export.ExportArtifactJSON(first.Artifact, &firstArtifactJSON); err != nil {
		t.Fatalf("first artifact JSON export failed: %v", err)
	}
	if err := export.ExportArtifactJSON(second.Artifact, &secondArtifactJSON); err != nil {
		t.Fatalf("second artifact JSON export failed: %v", err)
	}
	if !bytes.Equal(firstArtifactJSON.Bytes(), secondArtifactJSON.Bytes()) {
		t.Fatal("Expected repeated artifact JSON exports to be byte-identical")
	}
}

func TestCVRoleHeadingDoesNotDuplicateOrganization(t *testing.T) {
	role := resumeRoleClaim("claim:cv:role:org", "Platform Engineer", "Example Systems", "exp:cv-org")
	role.Value = "Platform Engineer"
	plan := skillArtifactPlan(planning.TypeCV, []claims.Claim{role})

	first := rendering.Render(context.Background(), rendering.RenderRequest{Plan: plan})
	second := rendering.Render(context.Background(), rendering.RenderRequest{Plan: plan})
	if first.Artifact == nil || second.Artifact == nil {
		t.Fatalf("Expected CV artifacts, got diagnostics: %+v / %+v", first.Diagnostics, second.Diagnostics)
	}
	rendered := renderedEntryTexts(first.Artifact)
	if !containsString(rendered, "Platform Engineer at Example Systems (2022-01 – 2024-01)") {
		t.Fatalf("Expected structured role heading with one organization reference, got %+v", rendered)
	}
	for _, text := range rendered {
		if strings.Contains(text, "Example Systems at Example Systems") {
			t.Fatalf("CV heading duplicated organization: %q", text)
		}
	}
	if strings.Join(renderedEntryTexts(first.Artifact), "\n") != strings.Join(renderedEntryTexts(second.Artifact), "\n") {
		t.Fatal("Expected deterministic CV role heading output")
	}
}

func TestCVRoleHeadingFallbacksRenderSafely(t *testing.T) {
	orgless := resumeRoleClaim("claim:cv:role:no-org", "Independent Engineer", "", "exp:no-org")
	orgless.Statement = "Served as Independent Engineer"
	orgless.Value = "Independent Engineer"
	orgless.Organizations = nil
	employmentOnly := resumeRoleClaim("claim:cv:employment", "Example Systems", "Example Systems", "exp:employment")
	employmentOnly.Kind = claims.KindEmployment
	employmentOnly.Statement = "Employed at Example Systems"
	employmentOnly.Value = ""

	res := rendering.Render(context.Background(), rendering.RenderRequest{Plan: skillArtifactPlan(planning.TypeCV, []claims.Claim{orgless, employmentOnly})})
	if res.Artifact == nil {
		t.Fatalf("Expected CV artifact, got diagnostics: %+v", res.Diagnostics)
	}
	rendered := renderedEntryTexts(res.Artifact)
	if !containsString(rendered, "Independent Engineer (2022-01 – 2024-01)") {
		t.Fatalf("Expected organization-less role fallback without dangling separator, got %+v", rendered)
	}
	if !containsString(rendered, "Example Systems (2022-01 – 2024-01)") {
		t.Fatalf("Expected employment-only fallback with one organization reference, got %+v", rendered)
	}
	for _, text := range rendered {
		if strings.Contains(text, " at  ") || strings.Contains(text, " at (") || strings.Contains(text, "Example Systems at Example Systems") {
			t.Fatalf("Unsafe CV heading fallback: %q", text)
		}
	}
}

func TestCVSkillEntriesCarryProvenance(t *testing.T) {
	skill := skillClaim("claim:skill:cv", claims.KindTechnicalSkill, "Utilized technologies: Go", []string{"skill:cv"}, []string{"ev:cv"})
	res := rendering.Render(context.Background(), rendering.RenderRequest{Plan: skillArtifactPlan(planning.TypeCV, []claims.Claim{skill})})
	if res.Artifact == nil {
		t.Fatalf("Expected CV artifact, got diagnostics: %+v", res.Diagnostics)
	}
	entry := findArtifactEntry(t, res.Artifact, "entry:cv-skill-0")
	assertClaimIDsEqual(t, entry.ClaimIDs, []claims.ClaimID{"claim:skill:cv"})
	assertStringsEqual(t, entry.SourceObjectIDs, []string{"skill:cv"})
	assertStringsEqual(t, entry.EvidenceIDs, []string{"ev:cv"})
	assertStringsEqual(t, res.Artifact.Manifest.SourceReferences, []string{"skill:cv"})
	assertStringsEqual(t, res.Artifact.Manifest.EvidenceReferences, []string{"ev:cv"})
}

func TestStrengthPreservationVerification(t *testing.T) {
	// 1. Ownership Strength Upgrade (contributed -> led)
	claim1 := claims.Claim{
		Statement: "Contributed to the migrations of gateways.",
	}
	err := rendering.CheckStrengthPreservation(claim1, "Led the migrations of gateways.")
	if err == nil {
		t.Error("Expected ownership strength upgrade to return error, got nil")
	}

	// 2. Skill Strength Upgrade (familiar with -> expert)
	claim2 := claims.Claim{
		Statement: "Familiar with Go programming.",
	}
	err = rendering.CheckStrengthPreservation(claim2, "Expert Go programming.")
	if err == nil {
		t.Error("Expected skill strength upgrade to return error, got nil")
	}

	// 3. Certainty Strength Upgrade (planned -> achieved)
	claim3 := claims.Claim{
		Statement: "Planned outage resolution.",
	}
	err = rendering.CheckStrengthPreservation(claim3, "Achieved outage resolution.")
	if err == nil {
		t.Error("Expected certainty strength upgrade to return error, got nil")
	}

	// 4. Scope Strength Upgrade (team -> enterprise)
	claim4 := claims.Claim{
		Statement: "Managed team gateway migrations.",
	}
	err = rendering.CheckStrengthPreservation(claim4, "Managed enterprise gateway migrations.")
	if err == nil {
		t.Error("Expected scope strength upgrade to return error, got nil")
	}

	// 5. Permitted weakening (led -> contributed)
	claimLed := claims.Claim{
		Statement: "Led the migration of gateways.",
	}
	err = rendering.CheckStrengthPreservation(claimLed, "Contributed to migration of gateways.")
	if err != nil {
		t.Errorf("Expected weakening to be permitted, got error: %v", err)
	}

	// 6. Cross-dimension non-interference
	// A source containing "expert" (skill level 7) but no ownership verb
	// destination containing "implemented" (ownership level 5) but no skill verb
	// Since implemented is not compared directly to expert, this should succeed.
	claimSkill := claims.Claim{
		Statement: "Expert in Go.",
	}
	err = rendering.CheckStrengthPreservation(claimSkill, "Implemented service.")
	if err != nil {
		t.Errorf("Expected cross-dimension non-interference to pass, got error: %v", err)
	}
}

func TestRenderValidationNonBlockingConflictsWarnAndRender(t *testing.T) {
	plan := renderConflictPlan(false, nil)
	res := rendering.Render(context.Background(), rendering.RenderRequest{Plan: plan})

	if res.Artifact == nil {
		t.Fatalf("Expected non-blocking conflict plan to render, got diagnostics: %+v", res.Diagnostics)
	}
	if !hasDiagnostic(res.Diagnostics, "CKB-RENDER-NONBLOCKING-CONFLICT", model.SeverityWarning) {
		t.Fatalf("Expected non-blocking conflict warning diagnostic, got %+v", res.Diagnostics)
	}
	if hasDiagnostic(res.Diagnostics, "CKB-RENDER-BLOCKING-CONFLICT", model.SeverityFatal) {
		t.Fatalf("Non-blocking conflict incorrectly produced blocking diagnostic: %+v", res.Diagnostics)
	}
}

func TestRenderValidationBlockingConflictsFailUnlessOverridden(t *testing.T) {
	blockingPlan := renderConflictPlan(true, nil)
	blockingRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: blockingPlan})
	if blockingRes.Artifact != nil {
		t.Fatal("Expected blocking conflict to prevent artifact rendering")
	}
	if !hasDiagnostic(blockingRes.Diagnostics, "CKB-RENDER-BLOCKING-CONFLICT", model.SeverityFatal) {
		t.Fatalf("Expected blocking conflict fatal diagnostic, got %+v", blockingRes.Diagnostics)
	}

	overriddenPlan := renderConflictPlan(true, []planning.Override{
		{ClaimID: "claim:conflict-a", Action: "Include"},
		{ClaimID: "claim:conflict-b", Action: "Include"},
	})
	overriddenRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: overriddenPlan})
	if overriddenRes.Artifact == nil {
		t.Fatalf("Expected override to allow blocking conflict render, got diagnostics: %+v", overriddenRes.Diagnostics)
	}
	if hasDiagnostic(overriddenRes.Diagnostics, "CKB-RENDER-BLOCKING-CONFLICT", model.SeverityFatal) {
		t.Fatalf("Override should suppress blocking conflict fatal diagnostics: %+v", overriddenRes.Diagnostics)
	}
}

func renderConflictPlan(blocked bool, overrides []planning.Override) *planning.ArtifactPlan {
	claimA := renderConflictClaim("claim:conflict-a", "Served as Platform Engineer", "exp:a")
	claimB := renderConflictClaim("claim:conflict-b", "Served as Systems Engineer", "exp:b")
	return &planning.ArtifactPlan{
		ID:            "plan:conflict-test",
		ArtifactType:  planning.TypeResume,
		SchemaVersion: "1.0",
		Policy: claims.Policy{
			ID:                  "StrictPublic",
			AllowedVisibilities: []model.Visibility{model.VisibilityPublic},
		},
		SelectedClaims: []planning.PlannedClaim{
			{Claim: claimA, Section: "Experience"},
			{Claim: claimB, Section: "Experience"},
		},
		Conflicts: []claims.ClaimConflict{
			{
				ID:          "conflict:test",
				Type:        "DateOverlap",
				Severity:    "Warning",
				Explanation: "overlapping employment retained as non-blocking audit context",
				ClaimIDs:    []string{string(claimA.ID), string(claimB.ID)},
				Blocked:     blocked,
			},
		},
		Overrides: overrides,
	}
}

func renderConflictClaim(id claims.ClaimID, statement string, sourceID string) claims.Claim {
	return claims.Claim{
		ID:              id,
		Kind:            claims.KindRole,
		Statement:       statement,
		SourceObjectIDs: []string{sourceID},
		SourceLocations: []model.SourceLocation{{FilePath: sourceID + ".md", Line: 1}},
		Visibility:      model.VisibilityPublic,
		Verification:    model.VerificationSelfAttested,
		Confidence:      0.9,
		Organizations:   []string{"Example Org"},
		Status:          "Active",
	}
}

func hasDiagnostic(diags []model.Diagnostic, code model.DiagnosticCode, severity model.DiagnosticSeverity) bool {
	for _, diag := range diags {
		if diag.Code == code && diag.Severity == severity {
			return true
		}
	}
	return false
}

func experienceObjectForOrganization(id string, metadataSource string, organization string) *model.Object {
	body := "*   **Role**: Platform Engineer\n*   **Duration**: 2022-01 - 2024-01\n*   **Location**: Remote\n"
	if organization != "" {
		body = "*   **Organization**: " + organization + "\n" + body
	}
	return &model.Object{
		ID:         id,
		Type:       model.TypeExperience,
		SourceFile: id + ".md",
		Metadata: model.Metadata{
			ID:           id,
			Type:         model.TypeExperience,
			Status:       model.StatusActive,
			Verification: model.VerificationSelfAttested,
			Confidence:   0.9,
			Visibility:   model.VisibilityPublic,
			Source:       metadataSource,
		},
		Sections: []model.Section{
			{Heading: "## 1. Role Context", Body: body},
		},
	}
}

func resumeRoleClaim(id claims.ClaimID, title string, organization string, sourceID string) claims.Claim {
	return claims.Claim{
		ID:              id,
		Kind:            claims.KindRole,
		Statement:       "Served as " + title + " at " + organization,
		SourceObjectIDs: []string{sourceID},
		SourceLocations: []model.SourceLocation{{FilePath: sourceID + ".md", Line: 1}},
		Visibility:      model.VisibilityPublic,
		Verification:    model.VerificationSelfAttested,
		Confidence:      0.9,
		Organizations:   []string{organization},
		TimeRange: &claims.TimeRange{
			Start: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Status: "Active",
	}
}

func resumePlanWithClaims(roleClaims []claims.Claim) *planning.ArtifactPlan {
	selected := make([]planning.PlannedClaim, 0, len(roleClaims))
	for idx, c := range roleClaims {
		selected = append(selected, planning.PlannedClaim{
			Claim:   c,
			Section: "Experience",
			Rank:    idx + 1,
		})
	}
	return &planning.ArtifactPlan{
		ID:            "plan:resume-org-test",
		ArtifactType:  planning.TypeResume,
		SchemaVersion: "1.0",
		Policy: claims.Policy{
			ID:                  "StrictPublic",
			AllowedVisibilities: []model.Visibility{model.VisibilityPublic},
		},
		SelectedClaims: selected,
	}
}

func skillClaim(id claims.ClaimID, kind claims.ClaimKind, statement string, sourceIDs []string, evidenceIDs []string) claims.Claim {
	return claims.Claim{
		ID:                id,
		Kind:              kind,
		Statement:         statement,
		SourceObjectIDs:   sourceIDs,
		EvidenceObjectIDs: evidenceIDs,
		SourceLocations:   []model.SourceLocation{{FilePath: string(id) + ".md", Line: 1}},
		Visibility:        model.VisibilityPublic,
		Verification:      model.VerificationSelfAttested,
		Confidence:        0.9,
		Status:            "Active",
	}
}

func renderedEntryTexts(artifact *rendering.Artifact) []string {
	var out []string
	for _, section := range artifact.Sections {
		for _, entry := range section.Entries {
			out = append(out, entry.Text)
		}
	}
	return out
}

func skillArtifactPlan(artifactType planning.ArtifactType, skillClaims []claims.Claim) *planning.ArtifactPlan {
	selected := make([]planning.PlannedClaim, 0, len(skillClaims))
	for idx, c := range skillClaims {
		selected = append(selected, planning.PlannedClaim{
			Claim:   c,
			Section: "Skills",
			Rank:    idx + 1,
		})
	}
	return &planning.ArtifactPlan{
		ID:            "plan:skill-provenance-test",
		ArtifactType:  artifactType,
		SchemaVersion: "1.0",
		Policy: claims.Policy{
			ID:                  "StrictPublic",
			AllowedVisibilities: []model.Visibility{model.VisibilityPublic},
		},
		SelectedClaims: selected,
	}
}

func findArtifactEntry(t *testing.T, artifact *rendering.Artifact, id string) rendering.ArtifactEntry {
	t.Helper()
	for _, sec := range artifact.Sections {
		for _, entry := range sec.Entries {
			if entry.ID == id {
				return entry
			}
		}
	}
	t.Fatalf("Expected artifact entry %q, got sections %+v", id, artifact.Sections)
	return rendering.ArtifactEntry{}
}

func assertClaimIDsEqual(t *testing.T, got []claims.ClaimID, want []claims.ClaimID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Expected claim IDs %+v, got %+v", want, got)
	}
	for idx := range got {
		if got[idx] != want[idx] {
			t.Fatalf("Expected claim IDs %+v, got %+v", want, got)
		}
	}
}

func assertStringsEqual(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Expected strings %+v, got %+v", want, got)
	}
	for idx := range got {
		if got[idx] != want[idx] {
			t.Fatalf("Expected strings %+v, got %+v", want, got)
		}
	}
}

func experienceHeaderTexts(artifact *rendering.Artifact) map[string]bool {
	headers := make(map[string]bool)
	for _, sec := range artifact.Sections {
		if sec.Kind != "Experience" {
			continue
		}
		for _, entry := range sec.Entries {
			if entry.Kind == "header" {
				headers[entry.Text] = true
			}
		}
	}
	return headers
}

func countExperienceHeader(artifact *rendering.Artifact, header string) int {
	count := 0
	for _, sec := range artifact.Sections {
		if sec.Kind != "Experience" {
			continue
		}
		for _, entry := range sec.Entries {
			if entry.Kind == "header" && entry.Text == header {
				count++
			}
		}
	}
	return count
}

func TestTenseConjugationAdaptation(t *testing.T) {
	stmt := "Lead infrastructure updates."
	adapted := rendering.AdaptVerbTense(stmt, "past")
	if adapted != "Led infrastructure updates." {
		t.Errorf("Expected leading verb 'Lead' to adapt to 'Led', got %q", adapted)
	}

	stmt2 := "served as DevOps Architect"
	adapted2 := rendering.AdaptVerbTense(stmt2, "present")
	if adapted2 != "serve as DevOps Architect" {
		t.Errorf("Expected leading verb 'served' to adapt to 'serve', got %q", adapted2)
	}
}

func TestSTARStoryCompletenessChecking(t *testing.T) {
	// Verify biography pronoun formatting
	plan := &planning.ArtifactPlan{
		SelectedClaims: []planning.PlannedClaim{
			{
				Claim: claims.Claim{
					ID:        "claim:test",
					Kind:      claims.KindCareerObjective,
					Statement: "Transition into cloud architecture.",
					SourceLocations: []model.SourceLocation{
						{FilePath: "bio.md", Line: 1},
					},
					SourceObjectIDs: []string{"bio"},
					Visibility:      model.VisibilityPublic,
					Confidence:      0.90,
					Verification:    model.VerificationSelfAttested,
				},
				Section: "Summary",
			},
			{
				Claim: claims.Claim{
					ID:        "claim:test2",
					Kind:      claims.KindRole,
					Statement: "Served as DevOps Architect at Stark.",
					SourceLocations: []model.SourceLocation{
						{FilePath: "bio.md", Line: 5},
					},
					SourceObjectIDs: []string{"bio"},
					Visibility:      model.VisibilityPublic,
					Confidence:      0.90,
					Verification:    model.VerificationSelfAttested,
				},
				Section: "Experience",
			},
		},
		ArtifactType:  planning.TypeBiography,
		SchemaVersion: "1.0",
		Policy: claims.Policy{
			AllowedVisibilities: []model.Visibility{model.VisibilityPublic},
		},
	}

	req := rendering.RenderRequest{
		Plan: plan,
		Options: rendering.RenderOptions{
			PronounStyle:    "neutral",
			BiographyLength: "short",
			Contact: rendering.ContactInfo{
				Name: "Tony Stark",
			},
		},
	}

	res := rendering.Render(context.Background(), req)
	if len(res.Diagnostics) > 0 {
		for _, d := range res.Diagnostics {
			if d.Severity == model.SeverityFatal {
				t.Fatalf("Render failed with fatal diagnostics: %s", d.Message)
			}
		}
	}

	text := res.Artifact.Sections[0].Entries[0].Text
	if !strings.Contains(text, "Tony Stark served as DevOps Architect at Stark.") {
		t.Errorf("Expected name substitution in biography sentence, got %q", text)
	}
}

func TestSkillsSummaryDurationOverlaps(t *testing.T) {
	// Create two overlapping role claims
	now := time.Now()
	pc1 := planning.PlannedClaim{
		Claim: claims.Claim{
			Kind:   claims.KindRole,
			Skills: []string{"Go"},
			TimeRange: &claims.TimeRange{
				Start: now.AddDate(-2, 0, 0),
				End:   now.AddDate(-1, 0, 0), // 12 months
			},
		},
	}
	pc2 := planning.PlannedClaim{
		Claim: claims.Claim{
			Kind:   claims.KindRole,
			Skills: []string{"Go"},
			TimeRange: &claims.TimeRange{
				Start: now.AddDate(-1, -6, 0), // overlaps pc1 by 6 months
				End:   now.AddDate(0, 0, 0),   // ends now
			},
		},
	}

	selected := []planning.PlannedClaim{pc1, pc2}
	duration := rendering.CalculateSkillDuration("Go", selected)

	// Total non-overlapping duration: now-2years to now -> 24 months
	// Summing overlapping would be 12 + 18 = 30 months!
	if duration > 25 || duration < 23 {
		t.Errorf("Expected non-overlapping duration to resolve to ~24 months, got %.1f", duration)
	}
}

func TestBiographyProvenanceTracksOnlyRenderedClaims(t *testing.T) {
	plan := &planning.ArtifactPlan{
		ID:            "plan:bio-provenance",
		ArtifactType:  planning.TypeBiography,
		SchemaVersion: "1.0",
		Policy: claims.Policy{
			ID:                  "StrictPublic",
			AllowedVisibilities: []model.Visibility{model.VisibilityPublic},
		},
		SelectedClaims: []planning.PlannedClaim{
			{
				Claim: claims.Claim{
					ID:                "claim:summary",
					Kind:              claims.KindCareerObjective,
					Statement:         "Builds reliable platforms.",
					Visibility:        model.VisibilityPublic,
					Verification:      model.VerificationSelfAttested,
					Confidence:        0.9,
					SourceObjectIDs:   []string{"profile"},
					EvidenceObjectIDs: []string{"ev:summary"},
					SourceLocations:   []model.SourceLocation{{FilePath: "profile.md", Line: 1}},
				},
				Section: "Summary",
			},
			{
				Claim: claims.Claim{
					ID:                "claim:role",
					Kind:              claims.KindRole,
					Statement:         "Served as platform engineer.",
					Visibility:        model.VisibilityPublic,
					Verification:      model.VerificationSelfAttested,
					Confidence:        0.9,
					SourceObjectIDs:   []string{"exp:platform"},
					EvidenceObjectIDs: []string{"ev:role"},
					SourceLocations:   []model.SourceLocation{{FilePath: "experience.md", Line: 3}},
				},
				Section: "Experience",
			},
			{
				Claim: claims.Claim{
					ID:                "claim:achievement",
					Kind:              claims.KindAccomplishment,
					Statement:         "Improved deployment reliability.",
					Visibility:        model.VisibilityPublic,
					Verification:      model.VerificationSelfAttested,
					Confidence:        0.9,
					SourceObjectIDs:   []string{"exp:platform"},
					EvidenceObjectIDs: []string{"ev:achievement"},
					SourceLocations:   []model.SourceLocation{{FilePath: "experience.md", Line: 9}},
				},
				Section: "Achievements",
			},
			{
				Claim: claims.Claim{
					ID:                "claim:education",
					Kind:              claims.KindEducation,
					Statement:         "Completed computer science coursework.",
					Visibility:        model.VisibilityPublic,
					Verification:      model.VerificationSelfAttested,
					Confidence:        0.9,
					SourceObjectIDs:   []string{"edu:cs"},
					EvidenceObjectIDs: []string{"ev:education"},
					SourceLocations:   []model.SourceLocation{{FilePath: "education.md", Line: 4}},
				},
				Section: "Education",
			},
			{
				Claim: claims.Claim{
					ID:              "claim:credential-not-rendered",
					Kind:            claims.KindCredential,
					Statement:       "Holds unrelated credential.",
					Visibility:      model.VisibilityPublic,
					Verification:    model.VerificationSelfAttested,
					Confidence:      0.9,
					SourceObjectIDs: []string{"cred:unused"},
					SourceLocations: []model.SourceLocation{{FilePath: "credentials.md", Line: 2}},
				},
				Section: "Credentials",
			},
		},
	}

	res := rendering.Render(context.Background(), rendering.RenderRequest{
		Plan: plan,
		Options: rendering.RenderOptions{
			BiographyLength: "long",
			PronounStyle:    "neutral",
			Contact:         rendering.ContactInfo{Name: "Candidate"},
		},
	})
	if res.Artifact == nil {
		t.Fatalf("Expected biography artifact, got diagnostics: %+v", res.Diagnostics)
	}

	entries := res.Artifact.Sections[0].Entries
	if len(entries) != 2 {
		t.Fatalf("Expected 2 biography paragraphs, got %d", len(entries))
	}

	firstClaims := entries[0].ClaimIDs
	if len(firstClaims) != 2 || firstClaims[0] != "claim:summary" || firstClaims[1] != "claim:role" {
		t.Fatalf("First paragraph claim mapping included unrendered claims: %+v", firstClaims)
	}
	secondClaims := entries[1].ClaimIDs
	if len(secondClaims) != 2 || secondClaims[0] != "claim:achievement" || secondClaims[1] != "claim:education" {
		t.Fatalf("Second paragraph claim mapping included unrendered claims: %+v", secondClaims)
	}
	if strings.Contains(strings.Join(claimIDsToStrings(firstClaims), ","), "credential") ||
		strings.Contains(strings.Join(claimIDsToStrings(secondClaims), ","), "credential") {
		t.Fatalf("Biography provenance leaked unrendered credential claim: first=%+v second=%+v", firstClaims, secondClaims)
	}
}

func TestSkillsSummaryDurationOpenEndedRangesAreDeterministic(t *testing.T) {
	selected := []planning.PlannedClaim{
		{
			Claim: claims.Claim{
				Kind:   claims.KindRole,
				Skills: []string{"Go"},
				TimeRange: &claims.TimeRange{
					Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			Claim: claims.Claim{
				Kind:   claims.KindRole,
				Skills: []string{"Rust"},
				TimeRange: &claims.TimeRange{
					Start: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	first := rendering.CalculateSkillDuration("Go", selected)
	second := rendering.CalculateSkillDuration("Go", selected)
	if first != second {
		t.Fatalf("Expected deterministic open-ended skill duration, got %.4f and %.4f", first, second)
	}
	if first < 12 || first > 13 {
		t.Fatalf("Expected open-ended skill duration to use deterministic latest plan date, got %.2f months", first)
	}
}

func TestSTARStorySectionOrderIsDeterministic(t *testing.T) {
	plan := &planning.ArtifactPlan{
		ID:            "plan:star-order",
		ArtifactType:  planning.TypeSTARStory,
		SchemaVersion: "1.0",
		Policy: claims.Policy{
			ID:                  "StrictPublic",
			AllowedVisibilities: []model.Visibility{model.VisibilityPublic},
		},
		SelectedClaims: append(starClaimsForSource("src:b"), starClaimsForSource("src:a")...),
	}

	res := rendering.Render(context.Background(), rendering.RenderRequest{Plan: plan})
	if res.Artifact == nil {
		t.Fatalf("Expected STAR artifact, got diagnostics: %+v", res.Diagnostics)
	}
	if len(res.Artifact.Sections) != 2 {
		t.Fatalf("Expected 2 STAR sections, got %d", len(res.Artifact.Sections))
	}
	if res.Artifact.Sections[0].Heading != "STAR Story - src:a" || res.Artifact.Sections[1].Heading != "STAR Story - src:b" {
		t.Fatalf("Expected sorted STAR sections, got %q then %q", res.Artifact.Sections[0].Heading, res.Artifact.Sections[1].Heading)
	}
}

func claimIDsToStrings(ids []claims.ClaimID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	return out
}

func starClaimsForSource(sourceID string) []planning.PlannedClaim {
	kinds := []claims.ClaimKind{
		claims.KindSTARSituation,
		claims.KindSTARTask,
		claims.KindSTARAction,
		claims.KindSTARResult,
	}
	labels := []string{"Situation", "Task", "Action", "Result"}
	out := make([]planning.PlannedClaim, 0, len(kinds))
	for idx, kind := range kinds {
		out = append(out, planning.PlannedClaim{
			Claim: claims.Claim{
				ID:              claims.ClaimID(sourceID + ":" + strings.ToLower(labels[idx])),
				Kind:            kind,
				Statement:       labels[idx] + " for " + sourceID + ".",
				Visibility:      model.VisibilityPublic,
				Verification:    model.VerificationSelfAttested,
				Confidence:      0.9,
				SourceObjectIDs: []string{sourceID},
				SourceLocations: []model.SourceLocation{{FilePath: sourceID + ".md", Line: idx + 1}},
			},
			Section: "STAR",
		})
	}
	return out
}

func TestParserLimitsDefaults(t *testing.T) {
	lim := parser.DefaultLimits()
	if lim.MaxFileSize != 1048576 {
		t.Errorf("Expected MaxFileSize default of 1MB, got %d", lim.MaxFileSize)
	}
}

func TestExpandedRenderersAndDiagnostics(t *testing.T) {
	// 1. Setup mock plan
	plan := &planning.ArtifactPlan{
		ID:            "plan:mock-test",
		ArtifactType:  planning.TypeCV,
		SchemaVersion: "1.0",
		Policy: claims.Policy{
			ID:                  "ComprehensiveCV",
			AllowedVisibilities: []model.Visibility{model.VisibilityPublic, model.VisibilityInternal},
		},
		SelectedClaims: []planning.PlannedClaim{
			{
				Claim: claims.Claim{
					ID:              "claim:cv-1",
					Kind:            claims.KindCareerObjective,
					Statement:       "Career direction statement.",
					Visibility:      model.VisibilityPublic,
					Confidence:      0.90,
					Verification:    model.VerificationSelfAttested,
					SourceObjectIDs: []string{"bio"},
					SourceLocations: []model.SourceLocation{{FilePath: "bio.md", Line: 1}},
				},
				Section: "Summary",
			},
			{
				Claim: claims.Claim{
					ID:              "claim:cv-2",
					Kind:            claims.KindRole,
					Statement:       "Served as Principal Architect.",
					Visibility:      model.VisibilityPublic,
					Confidence:      0.90,
					Verification:    model.VerificationIndependentlyVerified,
					SourceObjectIDs: []string{"exp:acme"},
					SourceLocations: []model.SourceLocation{{FilePath: "bio.md", Line: 5}},
				},
				Section: "Experience",
			},
			{
				Claim: claims.Claim{
					ID:              "claim:cv-3",
					Kind:            claims.KindSTARSituation,
					Statement:       "We faced gateway outages.",
					Visibility:      model.VisibilityPublic,
					Confidence:      0.90,
					Verification:    model.VerificationSelfAttested,
					SourceObjectIDs: []string{"exp:acme"},
					SourceLocations: []model.SourceLocation{{FilePath: "bio.md", Line: 10}},
				},
				Section: "STAR",
			},
			{
				Claim: claims.Claim{
					ID:              "claim:cv-4",
					Kind:            claims.KindSTARTask,
					Statement:       "Task was to resolve gateway outages.",
					Visibility:      model.VisibilityPublic,
					Confidence:      0.90,
					Verification:    model.VerificationSelfAttested,
					SourceObjectIDs: []string{"exp:acme"},
					SourceLocations: []model.SourceLocation{{FilePath: "bio.md", Line: 11}},
				},
				Section: "STAR",
			},
			{
				Claim: claims.Claim{
					ID:              "claim:cv-5",
					Kind:            claims.KindSTARAction,
					Statement:       "Implemented failover controls.",
					Visibility:      model.VisibilityPublic,
					Confidence:      0.90,
					Verification:    model.VerificationSelfAttested,
					SourceObjectIDs: []string{"exp:acme"},
					SourceLocations: []model.SourceLocation{{FilePath: "bio.md", Line: 12}},
				},
				Section: "STAR",
			},
			{
				Claim: claims.Claim{
					ID:              "claim:cv-6",
					Kind:            claims.KindSTARResult,
					Statement:       "Restored gateway uptime to 99.9%.",
					Visibility:      model.VisibilityPublic,
					Confidence:      0.90,
					Verification:    model.VerificationSelfAttested,
					SourceObjectIDs: []string{"exp:acme"},
					SourceLocations: []model.SourceLocation{{FilePath: "bio.md", Line: 13}},
				},
				Section: "STAR",
			},
		},
	}

	// 2. Test CV technical Subtype
	reqCV := rendering.RenderRequest{
		Plan: plan,
		Options: rendering.RenderOptions{
			Subtype: "technical",
		},
	}
	resCV := rendering.Render(context.Background(), reqCV)
	if len(resCV.Diagnostics) > 0 {
		t.Errorf("Expected zero diagnostics for valid CV, got %d", len(resCV.Diagnostics))
	}

	// 3. Test Biography medium/long and pronoun styles
	reqBio := rendering.RenderRequest{
		Plan: plan,
	}
	reqBio.Plan.ArtifactType = planning.TypeBiography
	reqBio.Options.BiographyLength = "medium"
	reqBio.Options.PronounStyle = "she/her"
	resBio := rendering.Render(context.Background(), reqBio)
	if len(resBio.Diagnostics) > 0 {
		t.Errorf("Expected zero diagnostics for Biography medium she/her, got %d", len(resBio.Diagnostics))
	}

	reqBioLong := reqBio
	reqBioLong.Options.BiographyLength = "long"
	reqBioLong.Options.PronounStyle = "they/them"
	resBioLong := rendering.Render(context.Background(), reqBioLong)
	if len(resBioLong.Diagnostics) > 0 {
		t.Errorf("Expected zero diagnostics for Biography long, got %d", len(resBioLong.Diagnostics))
	}

	// 4. Test STAR story configurations
	reqSTAR := rendering.RenderRequest{
		Plan: plan,
	}
	reqSTAR.Plan.ArtifactType = planning.TypeSTARStory
	reqSTAR.Options.Subtype = "paragraph"
	resSTAR := rendering.Render(context.Background(), reqSTAR)
	if len(resSTAR.Diagnostics) > 0 {
		t.Errorf("Expected zero diagnostics for STAR paragraph, got %d", len(resSTAR.Diagnostics))
	}

	reqSTARBullet := reqSTAR
	reqSTARBullet.Options.Subtype = "bullet"
	resSTARBullet := rendering.Render(context.Background(), reqSTARBullet)
	if len(resSTARBullet.Diagnostics) > 0 {
		t.Errorf("Expected zero diagnostics for STAR bullet, got %d", len(resSTARBullet.Diagnostics))
	}

	// 5. Test Skills Summary rendering
	reqSkills := rendering.RenderRequest{
		Plan: plan,
	}
	reqSkills.Plan.ArtifactType = planning.TypeSkillsSummary
	resSkills := rendering.Render(context.Background(), reqSkills)
	if len(resSkills.Diagnostics) > 0 {
		t.Errorf("Expected zero diagnostics for SkillsSummary, got %d", len(resSkills.Diagnostics))
	}

	// 6. Test Validation constraints & fatal diagnostics
	// Nil plan
	resNil := rendering.Render(context.Background(), rendering.RenderRequest{Plan: nil})
	if len(resNil.Diagnostics) == 0 || resNil.Diagnostics[0].Code != "CKB-RENDER-INVALID-PLAN" {
		t.Errorf("Expected CKB-RENDER-INVALID-PLAN warning, got: %+v", resNil.Diagnostics)
	}

	// Visibility violation
	plan.SelectedClaims[0].Claim.Visibility = model.VisibilityConfidential
	resVis := rendering.Render(context.Background(), reqCV)
	hasVisError := false
	for _, d := range resVis.Diagnostics {
		if d.Code == "CKB-RENDER-VISIBILITY-VIOLATION" {
			hasVisError = true
		}
	}
	if !hasVisError {
		t.Errorf("Expected visibility violation error, got: %+v", resVis.Diagnostics)
	}
}

func TestArtifactRendering_CoverLetter(t *testing.T) {
	// 1. Setup mock KnowledgeBase
	kb := model.NewKnowledgeBase()
	obj1 := &model.Object{
		ID:         "exp:stark-devops",
		Type:       model.TypeExperience,
		SourceFile: "/Users/example/dev/example-project/ckb/experience/stark-devops.md",
		Metadata: model.Metadata{
			Visibility:   model.VisibilityPublic,
			Verification: model.VerificationIndependentlyVerified,
			Source:       "Stark Industries",
			Confidence:   0.90,
		},
		Sections: []model.Section{
			{
				Heading: "Role Context",
				Body:    "Organization: Stark Industries\nRole: Principal DevOps Architect\nDuration: 2025-06 - 2026-06\nLocation: Remote",
			},
			{
				Heading: "Achievements",
				Body:    "- Cost Reduction: Saved $1.2M in annual cloud spend.",
			},
		},
	}
	kb.Objects[obj1.ID] = obj1

	// Build plan
	req := planning.PlanRequest{
		ArtifactType: planning.TypeCoverLetter,
		PolicyID:     "StrictPublic",
		Target: &planning.TargetProfile{
			RoleTitle: "Principal DevOps Architect",
		},
	}
	planRes := planning.BuildPlan(context.Background(), kb, req)
	if planRes.Plan == nil {
		t.Fatalf("Plan construction failed: %+v", planRes.Diagnostics)
	}

	// 2. Render Cover Letter
	renderReq := rendering.RenderRequest{
		Plan: planRes.Plan,
		Options: rendering.RenderOptions{
			Contact: rendering.ContactInfo{
				Name:    "Tony Stark",
				Email:   "tony@stark.com",
				Address: "Malibu, CA",
				Phone:   "123-456-7890",
			},
		},
	}

	res := rendering.Render(context.Background(), renderReq)
	if len(res.Diagnostics) > 0 {
		for _, d := range res.Diagnostics {
			if d.Severity == model.SeverityFatal {
				t.Fatalf("Fatal rendering diagnostic: %s", d.Message)
			}
		}
	}

	art := res.Artifact
	if art == nil {
		t.Fatalf("Artifact rendering returned nil artifact")
	}

	if art.Title != "Tony Stark" {
		t.Errorf("Expected title 'Tony Stark', got %q", art.Title)
	}

	// Export Markdown
	var mdBuf bytes.Buffer
	err := export.ExportMarkdown(art, &mdBuf, export.MarkdownOptions{IncludeHeadings: true})
	if err != nil {
		t.Fatalf("Markdown export failed: %v", err)
	}
	mdStr := mdBuf.String()

	if !strings.Contains(mdStr, "Dear Hiring Manager at Stark Industries,") {
		t.Errorf("Expected Markdown to contain salutation, got:\n%s", mdStr)
	}
	if !strings.Contains(mdStr, "Cost Reduction: Saved $1.2M in annual cloud spend.") {
		t.Errorf("Expected Markdown to contain accomplishment, got:\n%s", mdStr)
	}
}
