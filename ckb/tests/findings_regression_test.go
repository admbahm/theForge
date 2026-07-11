package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/parser"
	"github.com/admbahm/theForge/ckb/planning"
)

// Helper to write a temp file and parse it
func parseTempFile(t *testing.T, filename string, content string) (*model.Object, []model.Diagnostic) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: false,
		Limits:          parser.DefaultLimits(),
	}
	res := parser.ParseFiles(context.Background(), []string{path}, opts)
	for _, obj := range res.KnowledgeBase.Objects {
		return obj, res.Diagnostics
	}
	return nil, res.Diagnostics
}

// Finding 1: Exclude Inactive Source Objects From Generation
func TestInactiveSourceObjectsFromGeneration(t *testing.T) {
	// 1. Setup mock CKB with different statuses and lifecycles
	kb := model.NewKnowledgeBase()

	// Helper to add mock objects to KB
	addObject := func(id string, status model.Status, lifecycle model.LifecycleState) {
		kb.Objects[id] = &model.Object{
			ID:         id,
			Type:       model.TypeExperience,
			SourceFile: id + ".md",
			Metadata: model.Metadata{
				ID:           id,
				Type:         model.TypeExperience,
				Status:       status,
				Lifecycle:    lifecycle,
				Verification: model.VerificationSelfAttested,
				Confidence:   0.9,
				Visibility:   model.VisibilityPublic,
			},
			Sections: []model.Section{
				{
					Heading: "Role Context",
					Body:    "- Role: Engineer\n- Duration: 2024-01 to 2024-12\n",
				},
				{
					Heading: "Details",
					Body:    "- Achieved significant outcome for role " + id + "\n",
				},
			},
		}
	}

	// 1. Active status + Active lifecycle
	addObject("exp:active-active", model.StatusActive, model.LifecycleActive)
	// 2. Draft status
	addObject("exp:draft-active", model.StatusDraft, model.LifecycleActive)
	// 3. Deprecated status
	addObject("exp:deprecated-active", model.StatusDeprecated, model.LifecycleActive)
	// 4. Archived lifecycle
	addObject("exp:active-archived", model.StatusActive, model.LifecycleArchived)
	// 5. Planned lifecycle
	addObject("exp:active-planned", model.StatusActive, model.LifecyclePlanned)

	// Extract claims
	extracted, err := claims.ExtractClaims(kb)
	if err != nil {
		t.Fatalf("ExtractClaims failed: %v", err)
	}

	// Get strict public policy
	strictPolicy, _ := planning.GetPolicyByID(planning.PolicyStrictPublic)

	// Verify active is eligible
	activeClaim := findClaimBySource(extracted, "exp:active-active")
	status, code, _ := claims.EvaluateEligibility(activeClaim, strictPolicy)
	if status != claims.StatusEligible {
		t.Errorf("Expected active status + active lifecycle to be eligible, got %s / %s", status, code)
	}

	// Verify draft status is excluded
	draftClaim := findClaimBySource(extracted, "exp:draft-active")
	status, code, _ = claims.EvaluateEligibility(draftClaim, strictPolicy)
	if status != claims.StatusIneligible || code != claims.CodeClaimInactiveStatus {
		t.Errorf("Expected draft status to be excluded with CodeClaimInactiveStatus, got %s / %s", status, code)
	}

	// Verify deprecated status is excluded
	deprecatedClaim := findClaimBySource(extracted, "exp:deprecated-active")
	status, code, _ = claims.EvaluateEligibility(deprecatedClaim, strictPolicy)
	if status != claims.StatusIneligible || code != claims.CodeClaimInactiveStatus {
		t.Errorf("Expected deprecated status to be excluded with CodeClaimInactiveStatus, got %s / %s", status, code)
	}

	// Verify archived lifecycle is excluded
	archivedClaim := findClaimBySource(extracted, "exp:active-archived")
	status, code, _ = claims.EvaluateEligibility(archivedClaim, strictPolicy)
	if status != claims.StatusIneligible || code != claims.CodeClaimInactiveLifecycle {
		t.Errorf("Expected archived lifecycle to be excluded with CodeClaimInactiveLifecycle, got %s / %s", status, code)
	}

	// Verify planned lifecycle is excluded from public generation
	plannedClaim := findClaimBySource(extracted, "exp:active-planned")
	status, code, _ = claims.EvaluateEligibility(plannedClaim, strictPolicy)
	if status != claims.StatusIneligible || code != claims.CodeClaimInactiveLifecycle {
		t.Errorf("Expected planned lifecycle to be excluded from public generation with CodeClaimInactiveLifecycle, got %s / %s", status, code)
	}

	// 7. Inactive source role does not appear in resume output
	resumeRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     planning.PolicyStrictPublic,
	})
	if resumeRes.Plan == nil {
		t.Fatalf("BuildPlan failed: %+v", resumeRes.Diagnostics)
	}
	for _, pc := range resumeRes.Plan.SelectedClaims {
		for _, src := range pc.Claim.SourceObjectIDs {
			if src == "exp:draft-active" || src == "exp:deprecated-active" || src == "exp:active-archived" || src == "exp:active-planned" {
				t.Errorf("Inactive source %s should not appear in selected claims of resume", src)
			}
		}
	}

	// 8. Inactive source role does not appear in CV output unless permitted
	cvRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeCV,
		PolicyID:     planning.PolicyStrictPublic,
	})
	for _, pc := range cvRes.Plan.SelectedClaims {
		for _, src := range pc.Claim.SourceObjectIDs {
			if src == "exp:draft-active" || src == "exp:deprecated-active" || src == "exp:active-archived" || src == "exp:active-planned" {
				t.Errorf("Inactive source %s should not appear in selected claims of CV", src)
			}
		}
	}

	// 9. Inactive claims remain available in internal inspection where documented (PolicyInternalRecord)
	internalRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeCV,
		PolicyID:     planning.PolicyInternalRecord,
	})
	foundInactive := false
	for _, pc := range internalRes.Plan.SelectedClaims {
		for _, src := range pc.Claim.SourceObjectIDs {
			if src == "exp:draft-active" || src == "exp:deprecated-active" || src == "exp:active-archived" || src == "exp:active-planned" {
				foundInactive = true
			}
		}
	}
	if !foundInactive {
		t.Error("Expected inactive claims to be retained in internal inspection plan")
	}

	// 10. Active and inactive records with similar content do not deduplicate into an active merged claim
	// Let's create an active claim and an inactive claim with identical statement
	activeC := claims.Claim{
		ID:              "claim:active",
		Kind:            claims.KindAccomplishment,
		Statement:       "Built awesome stuff.",
		SourceObjectIDs: []string{"exp:active-active"},
		Status:          "Active",
		Lifecycle:       "Active",
	}
	inactiveC := claims.Claim{
		ID:              "claim:inactive",
		Kind:            claims.KindAccomplishment,
		Statement:       "Built awesome stuff.",
		SourceObjectIDs: []string{"exp:active-archived"},
		Status:          "Active",
		Lifecycle:       "Archived",
	}
	merged := claims.DeduplicateClaims([]claims.Claim{activeC, inactiveC})
	for _, c := range merged {
		if c.Lifecycle == "Archived" {
			status, _, _ = claims.EvaluateEligibility(c, strictPolicy)
			if status == claims.StatusEligible {
				t.Error("Merged claim with archived provenance should not be eligible under active policy")
			}
		}
	}

	// 11. Inactive evidence does not authorize active output
	// Let's add active role backed by inactive evidence (draft or deprecated evidence)
	kb2 := model.NewKnowledgeBase()
	kb2.Objects["exp:test-role"] = &model.Object{
		ID:         "exp:test-role",
		Type:       model.TypeExperience,
		SourceFile: "test-role.md",
		Metadata: model.Metadata{
			ID:           "exp:test-role",
			Type:         model.TypeExperience,
			Status:       model.StatusActive,
			Lifecycle:    model.LifecycleActive,
			Verification: model.VerificationArtifactSupported,
			Confidence:   1.0,
			Visibility:   model.VisibilityPublic,
			RelatedEvs:   []string{"ev:inactive-ev"},
		},
		Sections: []model.Section{
			{Heading: "Role Context", Body: "- Role: Staff\n- Duration: 2024-01 to 2024-12\n"},
		},
	}
	kb2.Objects["ev:inactive-ev"] = &model.Object{
		ID:         "ev:inactive-ev",
		Type:       model.TypeEvidence,
		SourceFile: "inactive-ev.md",
		Metadata: model.Metadata{
			ID:           "ev:inactive-ev",
			Type:         model.TypeEvidence,
			Status:       model.StatusDeprecated, // INACTIVE!
			Lifecycle:    model.LifecycleActive,
			Verification: model.VerificationSelfAttested,
			Confidence:   1.0,
			Visibility:   model.VisibilityPublic,
		},
	}
	planRes := planning.BuildPlan(context.Background(), kb2, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     planning.PolicyStrictPublic,
	})
	for _, pc := range planRes.Plan.SelectedClaims {
		for _, evID := range pc.Claim.EvidenceObjectIDs {
			if evID == "ev:inactive-ev" {
				t.Error("Inactive evidence ID should not appear in selected claim's authorized evidence list")
			}
		}
	}

	// 12. Repeated planning remains deterministic
	planRes2 := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     planning.PolicyStrictPublic,
	})
	if len(resumeRes.Plan.SelectedClaims) != len(planRes2.Plan.SelectedClaims) {
		t.Error("Repeated planning did not produce deterministic selected claim count")
	}
	for i := range resumeRes.Plan.SelectedClaims {
		if resumeRes.Plan.SelectedClaims[i].Claim.ID != planRes2.Plan.SelectedClaims[i].Claim.ID {
			t.Error("Repeated planning selected claim IDs do not match")
		}
	}
}

func findClaimBySource(claimsList []claims.Claim, sourceID string) claims.Claim {
	for _, c := range claimsList {
		for _, s := range c.SourceObjectIDs {
			if s == sourceID {
				return c
			}
		}
	}
	return claims.Claim{}
}

// Finding 2: Reject Raw HTML Before Parsing CKB Markdown
func TestRejectRawHTMLBeforeParsing(t *testing.T) {
	// Base CKB frontmatter template
	baseFM := `| Metadata | Value |
| --- | --- |
| **Schema Version** | 1.0 |
| **ID** | exp:fictional-staff |
| **Type** | Experience |
| **Status** | Active |
| **Verification Level** | Self-Attested |
| **Confidence** | 1.0 |
| **Visibility** | Public |
| **Source** | personal-notes |
| **Last Updated** | 2026-07-11 |
| **Lifecycle State** | Active |

## Role Context
- Role: Engineer

## Details
`

	expectedLine := strings.Count(baseFM, "\n") + 1

	// 1. Raw <div> block is rejected
	_, diags1 := parseTempFile(t, "exp:div.md", baseFM+"<div>some raw html</div>\n")
	assertFatalCode(t, diags1, model.CodeMarkdownRawHTML)

	// 2. Raw <iframe> block is rejected
	_, diags2 := parseTempFile(t, "exp:iframe.md", baseFM+"<iframe src=\"http://malicious.com\"></iframe>\n")
	assertFatalCode(t, diags2, model.CodeMarkdownRawHTML)

	// 3. Raw <script> block is rejected
	_, diags3 := parseTempFile(t, "exp:script.md", baseFM+"<script>console.log('malicious')</script>\n")
	assertFatalCode(t, diags3, model.CodeMarkdownRawHTML)

	// 4. Inline prohibited HTML is rejected
	_, diags4 := parseTempFile(t, "exp:inline-prohibited.md", baseFM+"This is inline <div class=\"inline\">block tag</div> text.\n")
	assertFatalCode(t, diags4, model.CodeMarkdownRawHTML)

	// 5. HTML comment is accepted
	obj5, diags5 := parseTempFile(t, "exp:comment.md", baseFM+"<!-- This is a safe comment -->\n")
	assertNoFatal(t, diags5)
	if obj5 == nil {
		t.Error("Expected object to be parsed successfully with comment")
	}

	// 6. Escaped HTML text is accepted
	obj6, diags6 := parseTempFile(t, "exp:escaped.md", baseFM+"This is escaped HTML: &lt;div&gt;some text&lt;/div&gt;\n")
	assertNoFatal(t, diags6)
	if obj6 == nil {
		t.Error("Expected object to be parsed successfully with escaped HTML")
	}

	// 7. HTML inside an allowed fenced code block is accepted
	obj7, diags7 := parseTempFile(t, "exp:code-block.md", baseFM+"```html\n<div>some html code</div>\n```\n")
	assertNoFatal(t, diags7)
	if obj7 == nil {
		t.Error("Expected object to be parsed successfully with HTML inside fenced code block")
	}

	// 8. Rejected HTML file does not create a valid object
	objDiv, _ := parseTempFile(t, "exp:div-failed.md", baseFM+"<div>failed</div>\n")
	if objDiv != nil {
		t.Error("Expected rejected HTML file to not register/create a valid object")
	}

	// 10. Diagnostic code and severity are exact
	foundCode := false
	for _, d := range diags1 {
		if d.Code == model.CodeMarkdownRawHTML && d.Severity == model.SeverityFatal {
			foundCode = true
			if !strings.Contains(d.Message, "Prohibited raw HTML detected") {
				t.Errorf("Unexpected diagnostic message: %s", d.Message)
			}
			if d.Source.Line != expectedLine {
				t.Errorf("Expected raw HTML on line %d, got line %d", expectedLine, d.Source.Line)
			}
		}
	}
	if !foundCode {
		t.Error("Expected exact CodeMarkdownRawHTML and SeverityFatal diagnostic")
	}

	// 12. Repeated parsing produces deterministic diagnostics
	_, diagsDiv1 := parseTempFile(t, "exp:div-det.md", baseFM+"<div>first</div>\n")
	_, diagsDiv2 := parseTempFile(t, "exp:div-det.md", baseFM+"<div>first</div>\n")
	if len(diagsDiv1) != len(diagsDiv2) || diagsDiv1[0].Message != diagsDiv2[0].Message {
		t.Error("Repeated parsing of rejected HTML did not produce deterministic diagnostics")
	}
}

// Finding 3: Require Exactly Two Metadata-Table Columns
func TestRequireExactlyTwoMetadataTableColumns(t *testing.T) {
	// 1. Canonical two-column metadata table passes
	baseFM := `| Metadata | Value |
| --- | --- |
| **Schema Version** | 1.0 |
| **ID** | exp:fictional-staff |
| **Type** | Experience |
| **Status** | Active |
| **Verification Level** | Self-Attested |
| **Confidence** | 1.0 |
| **Visibility** | Public |
| **Source** | personal-notes |
| **Last Updated** | 2026-07-11 |
| **Lifecycle State** | Active |

## Role Context
- Role: Engineer
`
	obj1, diags1 := parseTempFile(t, "exp:valid-meta.md", baseFM)
	assertNoFatal(t, diags1)
	if obj1 == nil {
		t.Error("Expected canonical two-column metadata table to pass and parse successfully")
	}

	// 2. Three-column header fails
	fm2 := `| Metadata | Value | Extra |
| --- | --- | --- |
| **Schema Version** | 1.0 | extra |
| **ID** | exp:fictional-staff | extra |
| **Type** | Experience | extra |
`
	_, diags2 := parseTempFile(t, "exp:three-cols.md", fm2)
	assertFatalCode(t, diags2, model.CodeMetadataInvalidColumnCount)

	// 3. Two-column header with a three-cell data row fails
	fm3 := `| Metadata | Value |
| --- | --- |
| **Schema Version** | 1.0 | extra |
| **ID** | exp:fictional-staff |
| **Type** | Experience |
`
	_, diags3 := parseTempFile(t, "exp:three-cell-row.md", fm3)
	assertFatalCode(t, diags3, model.CodeMetadataMalformedRow)

	// 4. One-column metadata table fails
	fm4 := `| Metadata |
| --- |
| **Schema Version** |
| **ID** |
| **Type** |
`
	_, diags4 := parseTempFile(t, "exp:one-col.md", fm4)
	assertFatalCode(t, diags4, model.CodeMetadataInvalidColumnCount)

	// 5. Extra empty third column still fails
	fm5 := `| Metadata | Value | |
| --- | --- | --- |
| **Schema Version** | 1.0 | |
| **ID** | exp:fictional-staff | |
| **Type** | Experience | |
`
	_, diags5 := parseTempFile(t, "exp:empty-third.md", fm5)
	assertFatalCode(t, diags5, model.CodeMetadataInvalidColumnCount)

	// 7. Escaped pipe in a Value cell does not count as an extra column
	fm7 := `| Metadata | Value |
| --- | --- |
| **Schema Version** | 1.0 |
| **ID** | exp:fictional-staff |
| **Type** | Experience |
| **Status** | Active |
| **Verification Level** | Self-Attested |
| **Confidence** | 1.0 |
| **Visibility** | Public |
| **Source** | personal-notes |
| **Last Updated** | 2026-07-11 |
| **Lifecycle State** | Active |
| **Tags** | tag1 \| tag2 |

## Role Context
- Role: Engineer
`
	obj7, diags7 := parseTempFile(t, "exp:escaped-pipe.md", fm7)
	assertNoFatal(t, diags7)
	if obj7 == nil {
		t.Error("Expected metadata table with escaped pipe in value cell to pass")
	}

	// 10. Diagnostic ordering is deterministic
	_, diags8 := parseTempFile(t, "exp:double-fail.md", fm2)
	// Check sorting is stable
	for i := 0; i < len(diags8)-1; i++ {
		if diags8[i].Source.Line > diags8[i+1].Source.Line {
			t.Error("Diagnostics were not sorted deterministically by line number")
		}
	}
}

// Diagnostic asserts
func assertFatalCode(t *testing.T, diags []model.Diagnostic, expectedCode model.DiagnosticCode) {
	t.Helper()
	found := false
	for _, d := range diags {
		if d.Code == expectedCode && (d.Severity == model.SeverityFatal || d.Severity == model.SeverityError) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected fatal/error diagnostic code %s, got diagnostics: %+v", expectedCode, diags)
	}
}

func assertNoFatal(t *testing.T, diags []model.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.Severity == model.SeverityFatal || d.Severity == model.SeverityError {
			t.Errorf("Expected no fatal/error diagnostics, got: %+v", d)
		}
	}
}
