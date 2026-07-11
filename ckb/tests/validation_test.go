package tests

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/export"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/parser"
	"github.com/admbahm/theForge/ckb/planning"
	"github.com/admbahm/theForge/ckb/rendering"
	"github.com/admbahm/theForge/ckb/validation"
)

func TestValidFixtures(t *testing.T) {
	rootPath := filepath.Join("..", "tests", "fixtures", "valid")

	rootFiles, err := parser.DiscoverFiles(filepath.Join(".."))
	if err != nil {
		t.Fatalf("failed to discover root CKB files: %v", err)
	}

	t.Logf("Found %d root files", len(rootFiles))
	for _, rf := range rootFiles {
		t.Logf("  Root file: %s", rf)
	}

	files := append(rootFiles,
		filepath.Join(rootPath, "minimal_valid.md"),
		filepath.Join(rootPath, "complete_experience.md"),
		filepath.Join(rootPath, "complete_project.md"),
		filepath.Join(rootPath, "evidence_graph.md"),
		filepath.Join(rootPath, "private_evidence.md"),
		filepath.Join(rootPath, "many_to_many.md"),
	)

	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	}

	res := parser.ParseFiles(context.Background(), files, opts)

	// Assert zero error/fatal diagnostics
	for _, diag := range res.Diagnostics {
		if diag.Severity == model.SeverityFatal || diag.Severity == model.SeverityError {
			t.Errorf("Unexpected error diagnostic in valid fixtures: %s (%s)", diag.Message, diag.Code)
		}
	}

	kb := res.KnowledgeBase

	// Assert expected object count (6 files = 7 IDs, since evidence_graph declares ev:git-fixture-repo and ev:eval-fixture-2025 sub-ids, wait, pre-scan was run in previous test, but here it's parsed directly)
	// Let's check objects map
	expectedIDs := []string{
		"profile:minimal-profile",
		"exp:fictional-staff",
		"proj:fictional-titan",
		"ev:fictional-main",
		"proj:private-spec",
		"acc:many-links",
	}

	for _, id := range expectedIDs {
		if _, ok := kb.Objects[id]; !ok {
			t.Errorf("Expected object ID %q was not found in parsed knowledge base", id)
		}
	}

	// Verify complete_experience object metadata and fields
	expObj := kb.Objects["exp:fictional-staff"]
	if expObj != nil {
		if expObj.Type != model.TypeExperience {
			t.Errorf("Expected exp:fictional-staff type to be Experience, got %s", expObj.Type)
		}
		if expObj.Metadata.Confidence != 0.95 {
			t.Errorf("Expected confidence 0.95, got %.2f", expObj.Metadata.Confidence)
		}
		if expObj.Metadata.Verification != model.VerificationIndependentlyVerified {
			t.Errorf("Expected verification Independently-Verified, got %s", expObj.Metadata.Verification)
		}
	}

	// Verify duplicate JSON export determinism (running twice produces byte-identical output)
	var w1, w2 bytes.Buffer
	err1 := export.ExportJSON(kb, res.Diagnostics, &w1)
	err2 := export.ExportJSON(kb, res.Diagnostics, &w2)

	if err1 != nil || err2 != nil {
		t.Fatalf("JSON export failed: %v / %v", err1, err2)
	}

	if !bytes.Equal(w1.Bytes(), w2.Bytes()) {
		t.Error("JSON export determinism failure: successive runs produced different bytes")
	}
}

func TestInvalidFixtures(t *testing.T) {
	rootPath := filepath.Join("..", "tests", "fixtures", "invalid")

	// Mapping invalid fixtures to expected Diagnostic Code and properties
	type expectDiag struct {
		Code     model.DiagnosticCode
		Severity model.DiagnosticSeverity
		Field    string
	}

	expectedDiagnostics := map[string]expectDiag{
		"missing_id.md":                 {model.CodeMetadataInvalidOrder, model.SeverityError, "Type"},
		"malformed_id.md":               {model.CodeIdentityInvalidID, model.SeverityFatal, "ID"},
		"duplicate_id.md":               {model.CodeIdentityDuplicateID, model.SeverityFatal, ""},
		"unknown_type.md":               {model.CodeMetadataInvalidObjType, model.SeverityFatal, "Type"},
		"missing_required_field.md":     {model.CodeMetadataInvalidOrder, model.SeverityError, "Confidence"},
		"invalid_field_order.md":        {model.CodeMetadataInvalidOrder, model.SeverityError, "Verification Level"},
		"duplicate_metadata_row.md":     {model.CodeMetadataDuplicateField, model.SeverityFatal, ""},
		"unexpected_metadata_field.md":  {model.CodeMetadataUnknownField, model.SeverityError, "Superfluous Key"},
		"invalid_enum.md":               {model.CodeMetadataInvalidEnum, model.SeverityError, "Status"},
		"malformed_date.md":             {model.CodeMetadataInvalidDate, model.SeverityError, "Last Updated"},
		"malformed_float.md":            {model.CodeMetadataInvalidConfidence, model.SeverityError, "Confidence"},
		"broken_relationship.md":        {model.CodeGraphBrokenReference, model.SeverityError, "Related Projects"},
		"invalid_relationship_type.md":  {model.CodeRelationshipInvalidType, model.SeverityError, "Related Projects"},
		"duplicate_edge.md":             {model.CodeGraphDuplicateEdge, model.SeverityError, "Related Projects"},
		"self_reference.md":             {model.CodeGraphInvalidSelfRef, model.SeverityError, "Related Experience"},
		"orphan_evidence.md":            {model.CodeEvidenceOrphaned, model.SeverityWarning, ""},
		"unsupported_schema_version.md": {model.CodeVersionUnsupported, model.SeverityError, "Schema Version"},
		"prohibited_pii.md":             {model.CodePrivacyProhibitedPII, model.SeverityFatal, ""},
	}

	rootFiles, err := parser.DiscoverFiles(filepath.Join(".."))
	if err != nil {
		t.Fatalf("failed to discover root CKB files: %v", err)
	}

	for file, expected := range expectedDiagnostics {
		filePath := filepath.Join(rootPath, file)

		// Parse individual invalid file alongside mock helper nodes to resolve relations
		files := append([]string{
			filePath,
			filepath.Join("..", "tests", "fixtures", "valid", "complete_project.md"), // proj:fictional-titan
			filepath.Join("..", "tests", "fixtures", "valid", "evidence_graph.md"),   // ev:fictional-main
		}, rootFiles...)

		opts := parser.ParseOptions{
			Strict:          true,
			ValidatePrivacy: true,
			Limits:          parser.DefaultLimits(),
		}

		res := parser.ParseFiles(context.Background(), files, opts)

		// Assert expected diagnostic code and severity
		found := false
		for _, diag := range res.Diagnostics {
			// Normalize filepath comparisons
			if diag.Code == expected.Code {
				found = true
				if diag.Severity != expected.Severity {
					t.Errorf("File %s: expected severity %s, got %s", file, expected.Severity, diag.Severity)
				}
				if expected.Field != "" && diag.Field != expected.Field {
					t.Errorf("File %s: expected field %q, got %q", file, expected.Field, diag.Field)
				}
				if !strings.Contains(filepath.ToSlash(diag.Source.FilePath), file) {
					t.Errorf("File %s: diagnostic source path %s does not contain filename", file, diag.Source.FilePath)
				}
				// Assert complete english wording does not break parser checks
				if len(diag.Message) == 0 {
					t.Errorf("File %s: expected descriptive message string, got empty message", file)
				}
				break
			}
		}

		if !found {
			t.Errorf("File %s: expected to fail with Diagnostic Code %s, but code was not generated.\nDiagnostics generated: %+v", file, expected.Code, res.Diagnostics)
		}
	}
}

func TestAdditionalParserConstraints(t *testing.T) {
	// 1. Context Cancellation Check
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	files := []string{filepath.Join("..", "tests", "fixtures", "valid", "minimal_valid.md")}
	opts := parser.ParseOptions{Strict: true, Limits: parser.DefaultLimits()}
	res := parser.ParseFiles(ctx, files, opts)

	cancelled := false
	for _, diag := range res.Diagnostics {
		if diag.Code == model.CodeStructureMalformed && strings.Contains(diag.Message, "cancelled") {
			cancelled = true
		}
	}
	if !cancelled {
		t.Error("Expected parsing cancellation diagnostic when context is cancelled")
	}

	// 2. Parser Sizing Limits Enforcements
	restrictLimits := parser.Limits{
		MaxFileSize:          100, // extremely small size limit
		MaxLineLength:        50,
		MaxMetadataRows:      5,
		MaxObjectCount:       10,
		MaxRelationshipsNode: 5,
		MaxSectionCount:      5,
		MaxHeadingDepth:      3,
	}

	resRestrict := parser.ParseFiles(context.Background(), files, parser.ParseOptions{Limits: restrictLimits})
	limitExceeded := false
	for _, diag := range resRestrict.Diagnostics {
		if diag.Code == model.CodeLimitsExceeded {
			limitExceeded = true
		}
	}
	if !limitExceeded {
		t.Error("Expected CodeLimitsExceeded when file size exceeds MaxFileSize restriction")
	}

	// 3. PII safe redaction check
	piiFile := filepath.Join("..", "tests", "fixtures", "invalid", "prohibited_pii.md")
	resPII := parser.ParseFiles(context.Background(), []string{piiFile}, opts)
	for _, diag := range resPII.Diagnostics {
		if diag.Code == model.CodePrivacyProhibitedPII {
			if strings.Contains(diag.Message, "adam@fictional.com") {
				t.Error("Privacy Leak: complete prohibited email value leaked in diagnostic message")
			}
			if !strings.Contains(diag.Message, "[REDACTED]") {
				t.Error("Expected redacted notation inside privacy diagnostics report")
			}
		}
	}
}

func TestParseDirectory(t *testing.T) {
	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	}
	res := parser.ParseDirectory(context.Background(), "..", opts)
	if res.KnowledgeBase == nil {
		t.Fatal("Expected non-nil KnowledgeBase from ParseDirectory")
	}
	// Verify that it successfully discovered and parsed the root files
	if len(res.KnowledgeBase.Objects) == 0 {
		t.Error("Expected parsed objects in ParseDirectory, got 0")
	}
}

func TestIDPrefixMustMatchDeclaredType(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		id   string
		typ  model.ObjectType
	}{
		{name: "experience prefix project type", id: "exp:foo", typ: model.TypeProject},
		{name: "project prefix experience type", id: "proj:foo", typ: model.TypeExperience},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := writeCKBTestFile(t, dir, strings.ReplaceAll(tc.name, " ", "-")+".md", ckbTestDoc(tc.id, tc.typ, "", "## 1. Test\n"))
			res := parser.ParseFiles(context.Background(), []string{file}, parser.ParseOptions{Limits: parser.DefaultLimits()})

			if _, exists := res.KnowledgeBase.Objects[tc.id]; exists {
				t.Fatalf("Object %q with mismatched Type %q should not be registered", tc.id, tc.typ)
			}
			assertDiagnostic(t, res.Diagnostics, model.CodeIdentityPrefixTypeMismatch)
		})
	}
}

func TestRelationshipValidationUsesResolvedTargetType(t *testing.T) {
	kb := model.NewKnowledgeBase()
	source := &model.Object{
		ID:   "acc:test",
		Type: model.TypeAccomplishment,
		Relationships: []model.Relationship{
			{
				SourceID: "acc:test",
				TargetID: "proj:wrong-kind",
				Type:     model.RelProjects,
				Source:   model.SourceLocation{FilePath: "source.md", Line: 1},
			},
		},
	}
	target := &model.Object{
		ID:         "proj:wrong-kind",
		Type:       model.TypeEvidence,
		SourceFile: "target.md",
		Metadata: model.Metadata{
			ID:   "proj:wrong-kind",
			Type: model.TypeEvidence,
		},
	}
	kb.Objects[source.ID] = source
	kb.Objects[target.ID] = target

	diags := validation.ValidateGraph(kb)
	assertDiagnostic(t, diags, model.CodeRelationshipInvalidType)
}

func TestMaxRelationshipsNodeLimit(t *testing.T) {
	dir := t.TempDir()
	evA := writeCKBTestFile(t, dir, "ev-a.md", ckbTestDoc("ev:a", model.TypeEvidence, "", "## 1. Evidence\n"))
	evB := writeCKBTestFile(t, dir, "ev-b.md", ckbTestDoc("ev:b", model.TypeEvidence, "", "## 1. Evidence\n"))
	evC := writeCKBTestFile(t, dir, "ev-c.md", ckbTestDoc("ev:c", model.TypeEvidence, "", "## 1. Evidence\n"))
	targetFiles := []string{evA, evB, evC}

	t.Run("exactly at limit accepted", func(t *testing.T) {
		source := writeCKBTestFile(t, dir, "at-limit.md", ckbTestDoc("proj:at-limit", model.TypeProject, "Related Evidence|ev:a, ev:b", "## 1. Project\n"))
		limits := parser.DefaultLimits()
		limits.MaxRelationshipsNode = 2
		res := parser.ParseFiles(context.Background(), append([]string{source}, targetFiles...), parser.ParseOptions{Limits: limits})
		if _, exists := res.KnowledgeBase.Objects["proj:at-limit"]; !exists {
			t.Fatalf("Expected object at relationship limit to be registered; diagnostics: %+v", res.Diagnostics)
		}
		assertNoDiagnostic(t, res.Diagnostics, model.CodeLimitRelationshipsExceeded)
	})

	t.Run("one over limit rejected", func(t *testing.T) {
		source := writeCKBTestFile(t, dir, "over-limit.md", ckbTestDoc("proj:over-limit", model.TypeProject, "Related Evidence|ev:a, ev:b, ev:c", "## 1. Project\n"))
		limits := parser.DefaultLimits()
		limits.MaxRelationshipsNode = 2
		res := parser.ParseFiles(context.Background(), append([]string{source}, targetFiles...), parser.ParseOptions{Limits: limits})
		if _, exists := res.KnowledgeBase.Objects["proj:over-limit"]; exists {
			t.Fatal("Expected object over relationship limit not to be registered")
		}
		assertDiagnostic(t, res.Diagnostics, model.CodeLimitRelationshipsExceeded)
	})

	t.Run("aggregate across rows", func(t *testing.T) {
		exp := writeCKBTestFile(t, dir, "exp-a.md", ckbTestDoc("exp:a", model.TypeExperience, "", "## 1. Experience\n"))
		source := writeCKBTestFile(t, dir, "aggregate-limit.md", ckbTestDoc("acc:aggregate-limit", model.TypeAccomplishment, "Related Experience|exp:a\nRelated Evidence|ev:a, ev:b", "## 1. Accomplishment\n"))
		limits := parser.DefaultLimits()
		limits.MaxRelationshipsNode = 2
		res := parser.ParseFiles(context.Background(), append([]string{source, exp}, targetFiles...), parser.ParseOptions{Limits: limits})
		assertDiagnostic(t, res.Diagnostics, model.CodeLimitRelationshipsExceeded)
	})

	t.Run("duplicate declared targets count before deduplication", func(t *testing.T) {
		source := writeCKBTestFile(t, dir, "duplicate-limit.md", ckbTestDoc("proj:duplicate-limit", model.TypeProject, "Related Evidence|ev:a, ev:a, ev:b", "## 1. Project\n"))
		limits := parser.DefaultLimits()
		limits.MaxRelationshipsNode = 2
		res := parser.ParseFiles(context.Background(), append([]string{source}, targetFiles...), parser.ParseOptions{Limits: limits})
		assertDiagnostic(t, res.Diagnostics, model.CodeLimitRelationshipsExceeded)
	})
}

func TestSyntheticEvidenceCatalogTimestampsAreDeterministic(t *testing.T) {
	dir := t.TempDir()
	file := writeCKBTestFile(t, dir, "evidence-catalog.md", ckbTestDoc("ev:catalog", model.TypeEvidence, "", "## 1. Relational Evidence Catalog\n| Evidence ID | Notes |\n| :--- | :--- |\n| **ev:inline-one** | One |\n| **ev:inline-two** | Two |\n"))
	opts := parser.ParseOptions{Limits: parser.DefaultLimits()}

	first := parser.ParseFiles(context.Background(), []string{file}, opts)
	second := parser.ParseFiles(context.Background(), []string{file}, opts)

	var firstJSON, secondJSON bytes.Buffer
	if err := export.ExportJSON(first.KnowledgeBase, first.Diagnostics, &firstJSON); err != nil {
		t.Fatalf("first export failed: %v", err)
	}
	if err := export.ExportJSON(second.KnowledgeBase, second.Diagnostics, &secondJSON); err != nil {
		t.Fatalf("second export failed: %v", err)
	}
	if !bytes.Equal(firstJSON.Bytes(), secondJSON.Bytes()) {
		t.Fatal("Expected repeated parse/export of synthetic evidence catalog to be byte-identical")
	}

	inline := first.KnowledgeBase.Objects["ev:inline-one"]
	if inline == nil {
		t.Fatal("Expected synthetic inline evidence node to be registered")
	}
	if !inline.Metadata.LastUpdated.Equal(first.KnowledgeBase.Objects["ev:catalog"].Metadata.LastUpdated) {
		t.Fatalf("Expected synthetic evidence LastUpdated to inherit parent timestamp, got %s", inline.Metadata.LastUpdated)
	}
}

func TestParserCanonicalizesMarkdownListItemsForClaimsAndRendering(t *testing.T) {
	dir := t.TempDir()
	file := writeCKBTestFile(t, dir, "list-semantics.md", ckbTestDoc("exp:list-semantics", model.TypeExperience, "", `## 1. Role Context
- **Organization**: Example Systems
* **Role**: Platform Engineer
+ **Duration**: 2022-01 - 2024-01

## 2. Key Achievements
Intro achievement prose.
- Plain bullet
* Star bullet
+ Plus bullet
- **Stability**: Achieved 99.99% uptime

Closing achievement prose.
`))

	res := parser.ParseFiles(context.Background(), []string{file}, parser.ParseOptions{Limits: parser.DefaultLimits()})
	for _, diag := range res.Diagnostics {
		if diag.Severity == model.SeverityFatal || diag.Severity == model.SeverityError {
			t.Fatalf("Unexpected parser diagnostic: %+v", diag)
		}
	}

	obj := res.KnowledgeBase.Objects["exp:list-semantics"]
	if obj == nil {
		t.Fatalf("Expected parsed object, got IDs: %+v", res.KnowledgeBase.Objects)
	}
	var achievementsBody string
	for _, sec := range obj.Sections {
		if strings.Contains(sec.Heading, "Key Achievements") {
			achievementsBody = sec.Body
			break
		}
	}
	expectedLines := []string{
		"Intro achievement prose.",
		"- Plain bullet",
		"- Star bullet",
		"- Plus bullet",
		"- **Stability**: Achieved 99.99% uptime",
		"Closing achievement prose.",
	}
	for _, expected := range expectedLines {
		if !strings.Contains(achievementsBody, expected) {
			t.Fatalf("Expected section body to contain %q, got %q", expected, achievementsBody)
		}
	}
	if strings.Contains(achievementsBody, "* Star bullet") || strings.Contains(achievementsBody, "+ Plus bullet") {
		t.Fatalf("Expected unordered list markers to be canonicalized to '-', got %q", achievementsBody)
	}

	extracted, err := claims.ExtractClaims(res.KnowledgeBase)
	if err != nil {
		t.Fatalf("ExtractClaims failed: %v", err)
	}
	expectedClaims := map[string]bool{
		"Plain bullet":                          false,
		"Star bullet":                           false,
		"Plus bullet":                           false,
		"**Stability**: Achieved 99.99% uptime": false,
	}
	for _, c := range extracted {
		if c.Kind == claims.KindAccomplishment {
			if _, ok := expectedClaims[c.Statement]; ok {
				expectedClaims[c.Statement] = true
			}
			if strings.HasPrefix(c.Statement, "Stability**") {
				t.Fatalf("Bold-led bullet was corrupted during extraction: %q", c.Statement)
			}
		}
	}
	for stmt, found := range expectedClaims {
		if !found {
			t.Fatalf("Expected extracted accomplishment %q from canonical bullets; claims: %+v", stmt, extracted)
		}
	}

	planRes := planning.BuildPlan(context.Background(), res.KnowledgeBase, planning.PlanRequest{
		ArtifactType:           planning.TypeResume,
		PolicyID:               planning.PolicyStrictPublic,
		Target:                 &planning.TargetProfile{RoleTitle: "Platform Engineer"},
		MaxRoles:               1,
		MaxAchievementsPerRole: 4,
	})
	if planRes.Plan == nil {
		t.Fatalf("Expected plan for canonical bullet fixture, got diagnostics: %+v", planRes.Diagnostics)
	}
	renderRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: planRes.Plan})
	if renderRes.Artifact == nil {
		t.Fatalf("Expected rendered artifact, got diagnostics: %+v", renderRes.Diagnostics)
	}
	var text bytes.Buffer
	if err := export.ExportText(renderRes.Artifact, &text, export.TextOptions{IncludeHeadings: true}); err != nil {
		t.Fatalf("ExportText failed: %v", err)
	}
	renderedText := text.String()
	if !strings.Contains(renderedText, "Stability: Achieved 99.99% uptime") {
		t.Fatalf("Expected plain-text render to preserve normalized bold-led bullet, got %q", renderedText)
	}
	if strings.Contains(renderedText, "Stability**") {
		t.Fatalf("Rendered artifact contains corrupted bold delimiter: %q", renderedText)
	}
}

func TestEvidenceCatalogVisibilityControlsPublicArtifactEvidence(t *testing.T) {
	dir := t.TempDir()
	evidenceFile := writeCKBTestFile(t, dir, "evidence-catalog.md", ckbTestDoc("ev:catalog", model.TypeEvidence, "", `## 1. Relational Evidence Catalog
| Evidence ID | Evidence Type | Verification Level | Visibility | Storage Location (Fictionalized) | Description |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **ev:public-proof** | Repository | Independently-Verified | Public | example.invalid/public | Public proof. |
| **ev:internal-proof** | Incident | Independently-Verified | Internal | internal.invalid/incident | Internal proof. |
| **ev:confidential-proof** | Review | Independently-Verified | Confidential | local-secure-review | Confidential proof. |
`))
	skillFile := writeCKBTestFile(t, dir, "skill-privacy.md", ckbTestDoc("skill:privacy", model.TypeSkill, "Related Evidence|ev:public-proof, ev:internal-proof, ev:confidential-proof", "## 1. Technical Skills\n- Go reliability engineering\n"))

	res := parser.ParseFiles(context.Background(), []string{evidenceFile, skillFile}, parser.ParseOptions{Limits: parser.DefaultLimits()})
	for _, diag := range res.Diagnostics {
		if diag.Severity == model.SeverityFatal || diag.Severity == model.SeverityError {
			t.Fatalf("Unexpected parser diagnostic: %+v", diag)
		}
	}
	assertEvidenceVisibility(t, res.KnowledgeBase, "ev:public-proof", model.VisibilityPublic)
	assertEvidenceVisibility(t, res.KnowledgeBase, "ev:internal-proof", model.VisibilityInternal)
	assertEvidenceVisibility(t, res.KnowledgeBase, "ev:confidential-proof", model.VisibilityConfidential)

	publicPlan := buildSkillPlan(t, res.KnowledgeBase, planning.PolicyStrictPublic, planning.TypeResume)
	publicClaim := findSelectedClaimBySource(t, publicPlan, "skill:privacy")
	if strings.Join(publicClaim.EvidenceObjectIDs, ",") != "ev:public-proof" {
		t.Fatalf("StrictPublic should retain only public evidence IDs, got %+v", publicClaim.EvidenceObjectIDs)
	}

	var planJSON bytes.Buffer
	if err := export.ExportPlanJSON(publicPlan, &planJSON); err != nil {
		t.Fatalf("ExportPlanJSON failed: %v", err)
	}

	renderRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: publicPlan})
	if renderRes.Artifact == nil {
		t.Fatalf("Expected public artifact, got diagnostics: %+v", renderRes.Diagnostics)
	}
	if !containsString(renderRes.Artifact.Manifest.EvidenceReferences, "ev:public-proof") {
		t.Fatalf("Expected public evidence in manifest, got %+v", renderRes.Artifact.Manifest.EvidenceReferences)
	}
	if containsString(renderRes.Artifact.Manifest.EvidenceReferences, "ev:internal-proof") ||
		containsString(renderRes.Artifact.Manifest.EvidenceReferences, "ev:confidential-proof") {
		t.Fatalf("Restricted evidence leaked into manifest: %+v", renderRes.Artifact.Manifest.EvidenceReferences)
	}

	var artifactJSON, sidecarJSON, markdownOut, textOut bytes.Buffer
	if err := export.ExportArtifactJSON(renderRes.Artifact, &artifactJSON); err != nil {
		t.Fatalf("ExportArtifactJSON failed: %v", err)
	}
	if err := export.ExportProvenanceSidecar(renderRes.Artifact, &sidecarJSON); err != nil {
		t.Fatalf("ExportProvenanceSidecar failed: %v", err)
	}
	if err := export.ExportMarkdown(renderRes.Artifact, &markdownOut, export.MarkdownOptions{IncludeHeadings: true}); err != nil {
		t.Fatalf("ExportMarkdown failed: %v", err)
	}
	if err := export.ExportText(renderRes.Artifact, &textOut, export.TextOptions{IncludeHeadings: true}); err != nil {
		t.Fatalf("ExportText failed: %v", err)
	}

	restrictedIDs := []string{"ev:internal-proof", "ev:confidential-proof"}
	assertBytesDoNotContain(t, planJSON.Bytes(), restrictedIDs, "artifact plan JSON")
	assertBytesDoNotContain(t, artifactJSON.Bytes(), restrictedIDs, "artifact JSON")
	assertBytesDoNotContain(t, sidecarJSON.Bytes(), restrictedIDs, "provenance sidecar")
	assertBytesDoNotContain(t, markdownOut.Bytes(), restrictedIDs, "markdown output")
	assertBytesDoNotContain(t, textOut.Bytes(), restrictedIDs, "text output")

	internalPlan := buildSkillPlan(t, res.KnowledgeBase, planning.PolicyInternalRecord, planning.TypeSkillsSummary)
	internalClaim := findSelectedClaimBySource(t, internalPlan, "skill:privacy")
	for _, id := range []string{"ev:public-proof", "ev:internal-proof", "ev:confidential-proof"} {
		if !containsString(internalClaim.EvidenceObjectIDs, id) {
			t.Fatalf("InternalRecord should retain evidence ID %q, got %+v", id, internalClaim.EvidenceObjectIDs)
		}
	}
}

func TestEvidenceCatalogInvalidAndOmittedVisibility(t *testing.T) {
	dir := t.TempDir()

	invalidFile := writeCKBTestFile(t, dir, "invalid-evidence-catalog.md", ckbTestDoc("ev:invalid-catalog", model.TypeEvidence, "", `## 1. Relational Evidence Catalog
| Evidence ID | Evidence Type | Verification Level | Visibility | Description |
| :--- | :--- | :---: | :--- | :--- |
| **ev:bad-visibility** | Repository | Independently-Verified | Secret | Invalid visibility. |
`))
	invalidRes := parser.ParseFiles(context.Background(), []string{invalidFile}, parser.ParseOptions{Limits: parser.DefaultLimits()})
	assertDiagnostic(t, invalidRes.Diagnostics, model.CodeMetadataInvalidEnum)
	if _, exists := invalidRes.KnowledgeBase.Objects["ev:bad-visibility"]; exists {
		t.Fatal("Catalog row with invalid visibility must not be registered as a usable evidence object")
	}

	omittedFile := writeCKBTestFile(t, dir, "omitted-evidence-catalog.md", ckbTestDoc("ev:omitted-catalog", model.TypeEvidence, "", `## 1. Relational Evidence Catalog
| Evidence ID | Evidence Type | Verification Level | Description |
| :--- | :--- | :---: | :--- |
| **ev:omitted-visibility** | Repository | Independently-Verified | Omitted visibility defaults safely. |
`))
	omittedRes := parser.ParseFiles(context.Background(), []string{omittedFile}, parser.ParseOptions{Limits: parser.DefaultLimits()})
	assertEvidenceVisibility(t, omittedRes.KnowledgeBase, "ev:omitted-visibility", model.VisibilityConfidential)
}

func writeCKBTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test CKB file: %v", err)
	}
	return path
}

func ckbTestDoc(id string, typ model.ObjectType, optionalRows string, body string) string {
	var b strings.Builder
	b.WriteString("| Metadata | Value |\n")
	b.WriteString("| :--- | :--- |\n")
	b.WriteString("| **Schema Version** | 1.0 |\n")
	b.WriteString("| **ID** | " + id + " |\n")
	b.WriteString("| **Type** | " + string(typ) + " |\n")
	b.WriteString("| **Status** | Active |\n")
	b.WriteString("| **Verification Level** | Self-Attested |\n")
	b.WriteString("| **Confidence** | 0.90 |\n")
	b.WriteString("| **Visibility** | Public |\n")
	b.WriteString("| **Source** | Synthetic Test Fixture |\n")
	b.WriteString("| **Last Updated** | 2026-07-10 |\n")
	b.WriteString("| **Lifecycle State** | Active |\n")
	if optionalRows != "" {
		for _, row := range strings.Split(optionalRows, "\n") {
			row = strings.TrimSpace(row)
			if row == "" {
				continue
			}
			parts := strings.SplitN(row, "|", 2)
			if len(parts) != 2 {
				continue
			}
			b.WriteString("| **" + strings.TrimSpace(parts[0]) + "** | " + strings.TrimSpace(parts[1]) + " |\n")
		}
	}
	b.WriteString("\n---\n\n")
	b.WriteString(body)
	return b.String()
}

func assertDiagnostic(t *testing.T, diagnostics []model.Diagnostic, code model.DiagnosticCode) {
	t.Helper()
	for _, diag := range diagnostics {
		if diag.Code == code {
			return
		}
	}
	t.Fatalf("Expected diagnostic %s, got %+v", code, diagnostics)
}

func assertNoDiagnostic(t *testing.T, diagnostics []model.Diagnostic, code model.DiagnosticCode) {
	t.Helper()
	for _, diag := range diagnostics {
		if diag.Code == code {
			t.Fatalf("Did not expect diagnostic %s, got %+v", code, diagnostics)
		}
	}
}

func assertEvidenceVisibility(t *testing.T, kb *model.KnowledgeBase, id string, visibility model.Visibility) {
	t.Helper()
	obj := kb.Objects[id]
	if obj == nil {
		t.Fatalf("Expected evidence object %q to be registered", id)
	}
	if obj.Metadata.Visibility != visibility {
		t.Fatalf("Expected evidence %q visibility %s, got %s", id, visibility, obj.Metadata.Visibility)
	}
}

func buildSkillPlan(t *testing.T, kb *model.KnowledgeBase, policyID string, artifactType planning.ArtifactType) *planning.ArtifactPlan {
	t.Helper()
	planRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: artifactType,
		PolicyID:     policyID,
		Target:       &planning.TargetProfile{DesiredSkills: []string{"Go"}},
	})
	if planRes.Plan == nil {
		t.Fatalf("Expected plan for %s, got diagnostics: %+v", policyID, planRes.Diagnostics)
	}
	return planRes.Plan
}

func findSelectedClaimBySource(t *testing.T, plan *planning.ArtifactPlan, sourceID string) claims.Claim {
	t.Helper()
	for _, pc := range plan.SelectedClaims {
		for _, id := range pc.Claim.SourceObjectIDs {
			if id == sourceID {
				return pc.Claim
			}
		}
	}
	t.Fatalf("Expected selected claim from source %q, got %+v", sourceID, plan.SelectedClaims)
	return claims.Claim{}
}

func assertBytesDoNotContain(t *testing.T, data []byte, restrictedIDs []string, label string) {
	t.Helper()
	for _, id := range restrictedIDs {
		if bytes.Contains(data, []byte(id)) {
			t.Fatalf("%s leaked restricted evidence ID %q:\n%s", label, id, string(data))
		}
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
