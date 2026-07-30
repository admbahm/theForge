package tests

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
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

func TestClaimExtraction(t *testing.T) {
	rootFiles, err := parser.DiscoverFiles(filepath.Join(".."))
	if err != nil {
		t.Fatalf("failed to discover root files: %v", err)
	}

	files := append([]string{
		filepath.Join("..", "tests", "fixtures", "valid", "complete_experience.md"),
		filepath.Join("..", "tests", "fixtures", "valid", "complete_project.md"),
		filepath.Join("..", "tests", "fixtures", "valid", "evidence_graph.md"),
	}, rootFiles...)

	res := parser.ParseFiles(context.Background(), files, parser.ParseOptions{Limits: parser.DefaultLimits()})
	for _, d := range res.Diagnostics {
		t.Logf("Diagnostic: Code=%s, Severity=%s, Message=%s, File=%s", d.Code, d.Severity, d.Message, d.Source.FilePath)
	}
	if len(res.Diagnostics) > 0 {
		for _, d := range res.Diagnostics {
			if d.Severity == model.SeverityFatal || d.Severity == model.SeverityError {
				t.Fatalf("failed to parse experience fixture: %v", d.Message)
			}
		}
	}

	extracted, err := claims.ExtractClaims(res.KnowledgeBase)
	if err != nil {
		t.Fatalf("failed to extract claims: %v", err)
	}

	for _, obj := range res.KnowledgeBase.Objects {
		t.Logf("Object ID: %s, Sections: %d", obj.ID, len(obj.Sections))
		for _, sec := range obj.Sections {
			t.Logf("  Heading: %q, Body length: %d", sec.Heading, len(sec.Body))
			if strings.Contains(strings.ToLower(sec.Heading), "role context") {
				t.Logf("    Body: %q", sec.Body)
			}
		}
	}

	// Verify we extracted Role, Employment, Responsibilities, and Technical Skills
	hasRole := false
	hasEmployment := false
	hasResp := false
	hasSkill := false
	hasEducation := false

	for _, c := range extracted {
		switch c.Kind {
		case claims.KindRole:
			hasRole = true
			if len(c.SourceObjectIDs) > 0 && c.SourceObjectIDs[0] == "exp:acme-lead" && c.Value != "Lead Infrastructure Engineer" {
				t.Errorf("Expected role 'Lead Infrastructure Engineer', got %q", c.Value)
			}
		case claims.KindEmployment:
			hasEmployment = true
			if c.TimeRange == nil {
				t.Error("Expected TimeRange to be parsed for employment claim")
			}
		case claims.KindResponsibility:
			hasResp = true
		case claims.KindTechnicalSkill:
			hasSkill = true
		case claims.KindEducation:
			hasEducation = true
			if len(c.SourceObjectIDs) > 0 && c.SourceObjectIDs[0] == "edu:metro-cs" {
				if !strings.Contains(c.Statement, "Bachelor of Science in Computer Science") {
					t.Errorf("Expected education claim to preserve degree heading, got %q", c.Statement)
				}
				if !strings.Contains(c.Statement, "Metropolis University") {
					t.Errorf("Expected education claim to preserve institution, got %q", c.Statement)
				}
				if c.TimeRange == nil {
					t.Error("Expected education claim to preserve Timeline as TimeRange")
				}
			}
		}
	}

	if !hasRole || !hasEmployment || !hasResp || !hasSkill || !hasEducation {
		t.Errorf("Missing expected claim kinds: role=%t, employment=%t, resp=%t, skill=%t, education=%t", hasRole, hasEmployment, hasResp, hasSkill, hasEducation)
	}
}

func TestEducationClaimExtractionFromParsedSubheadings(t *testing.T) {
	obj := &model.Object{
		ID:         "edu:multi-degree",
		Type:       model.TypeEducation,
		SourceFile: "education.md",
		Metadata: model.Metadata{
			ID:           "edu:multi-degree",
			Type:         model.TypeEducation,
			Status:       model.StatusActive,
			Verification: model.VerificationIndependentlyVerified,
			Confidence:   0.95,
			Visibility:   model.VisibilityPublic,
			RelatedEvs:   []string{"ev:transcript"},
		},
		Sections: []model.Section{
			{
				Heading: "## 1. Academic Credentials",
			},
			{
				Heading: "### Bachelor of Science in Computer Science",
				Body:    "*   **Institution**: Example University\n*   **Program**: B.S. in Computer Science\n*   **Timeline**: 2013-09 - 2017-05\n",
			},
			{
				Heading: "#### Core Coursework",
				Body:    "* Distributed Systems\n",
			},
			{
				Heading: "### Master of Science in Software Engineering",
				Body:    "*   **Institution**: Example Institute\n*   **Status**: In Progress\n*   **Timeline**: 2025-01 - Present\n",
			},
		},
	}

	extracted, err := claims.ExtractClaims(&model.KnowledgeBase{
		Objects: map[string]*model.Object{obj.ID: obj},
	})
	if err != nil {
		t.Fatalf("ExtractClaims failed: %v", err)
	}

	var educationClaims []claims.Claim
	for _, c := range extracted {
		if c.Kind == claims.KindEducation {
			educationClaims = append(educationClaims, c)
		}
	}

	if len(educationClaims) != 2 {
		t.Fatalf("Expected 2 education claims for two degree subheadings, got %d: %+v", len(educationClaims), educationClaims)
	}
	if !strings.Contains(educationClaims[0].Statement, "Bachelor of Science in Computer Science") ||
		!strings.Contains(educationClaims[0].Statement, "Example University") ||
		!strings.Contains(educationClaims[0].Statement, "2013-09 - 2017-05") {
		t.Fatalf("First education claim did not preserve degree, institution, and dates: %q", educationClaims[0].Statement)
	}
	if !strings.Contains(educationClaims[1].Statement, "Master of Science in Software Engineering") ||
		!strings.Contains(educationClaims[1].Statement, "Status: In Progress") {
		t.Fatalf("Second education claim did not preserve degree and status: %q", educationClaims[1].Statement)
	}
	if educationClaims[0].Verification != model.VerificationIndependentlyVerified ||
		educationClaims[0].Visibility != model.VisibilityPublic ||
		len(educationClaims[0].EvidenceObjectIDs) != 1 ||
		educationClaims[0].EvidenceObjectIDs[0] != "ev:transcript" {
		t.Fatalf("Education claim did not preserve metadata/provenance fields: %+v", educationClaims[0])
	}
}

func TestSparseAccomplishmentObjectsDoNotPanicPlanning(t *testing.T) {
	cases := []struct {
		name     string
		sections []model.Section
	}{
		{name: "zero sections", sections: nil},
		{name: "empty section", sections: []model.Section{{Heading: "## 1. Standalone Accomplishments"}}},
		{name: "metadata only", sections: []model.Section{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kb := model.NewKnowledgeBase()
			kb.Objects["acc:sparse"] = accomplishmentObject("acc:sparse", tc.sections)

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("BuildPlan panicked for sparse accomplishment object: %v", r)
				}
			}()

			res := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
				ArtifactType: planning.TypeResume,
				PolicyID:     planning.PolicyStrictPublic,
			})
			if res.Plan == nil {
				t.Fatalf("Expected sparse accomplishment planning to return a plan, got diagnostics: %+v", res.Diagnostics)
			}
			for _, pc := range res.Plan.SelectedClaims {
				if pc.Claim.Kind == claims.KindAccomplishment {
					t.Fatalf("Sparse accomplishment should not emit metadata-only claim: %+v", pc.Claim)
				}
			}
		})
	}
}

func TestNormalAccomplishmentExtractionRemainsUnchanged(t *testing.T) {
	kb := model.NewKnowledgeBase()
	kb.Objects["acc:normal"] = accomplishmentObject("acc:normal", []model.Section{
		{
			Heading: "## 1. Standalone Accomplishments",
			Body:    "* Improved release hygiene with documented rollback drills.\n",
		},
	})

	extracted, err := claims.ExtractClaims(kb)
	if err != nil {
		t.Fatalf("ExtractClaims failed: %v", err)
	}

	var accomplishments []claims.Claim
	for _, c := range extracted {
		if c.Kind == claims.KindAccomplishment {
			accomplishments = append(accomplishments, c)
		}
	}
	if len(accomplishments) != 1 {
		t.Fatalf("Expected one accomplishment claim, got %d: %+v", len(accomplishments), accomplishments)
	}
	if accomplishments[0].Statement != "Improved release hygiene with documented rollback drills." {
		t.Fatalf("Accomplishment statement changed unexpectedly: %q", accomplishments[0].Statement)
	}
}

func TestSkillTableRowsExtractClaimsAndPlanWithoutFalseGap(t *testing.T) {
	kb := model.NewKnowledgeBase()
	kb.Objects["skill:matrix"] = skillMatrixObject("skill:matrix", []model.Section{
		{
			Heading: "### Platform Skills",
			Body: `Intro prose before the table.
| Skill Name | Proficiency | Confidence | Years | Last Used | Related Experience | Related Projects | Supporting Evidence |
| :--- | :--- | :---: | :---: | :---: | :--- | :--- | :--- |
| **Go** | Expert | 0.99 | 7 | Present | [exp:platform](./experience.md) | [proj:platform](./project.md) | [ev:go-public](./evidence.md), [ev:go-internal](./evidence.md) |
| **Kubernetes** | Advanced | 0.90 | 5 | 2025 | [exp:platform](./experience.md) | [proj:platform](./project.md) | [ev:k8s-public](./evidence.md) |
| **AWS** |  |  |  |  | None | None | None |
| **Terraform** | Expert | 0.95 | 6 | Present | [exp:platform](./experience.md) | [proj:infra](./project.md) | [ev:terraform-public](./evidence.md) |
|  | Expert | 0.95 | 6 | Present | [exp:platform](./experience.md) | [proj:bad](./project.md) | [ev:bad](./evidence.md) |
Prose after the table.`,
		},
		{
			Heading: "### Secondary Skills",
			Body: `| Skill Name | Proficiency | Confidence | Years | Last Used | Related Experience | Related Projects | Supporting Evidence |
| :--- | :--- | :---: | :---: | :---: | :--- | :--- | :--- |
| **Go** | Novice | 0.10 | 1 | 2018 | None | None | [ev:duplicate](./evidence.md) |`,
		},
	})
	kb.Objects["ev:go-public"] = evidenceObject("ev:go-public", model.VisibilityPublic)
	kb.Objects["ev:go-internal"] = evidenceObject("ev:go-internal", model.VisibilityInternal)
	kb.Objects["ev:k8s-public"] = evidenceObject("ev:k8s-public", model.VisibilityPublic)
	kb.Objects["ev:terraform-public"] = evidenceObject("ev:terraform-public", model.VisibilityPublic)

	extracted, err := claims.ExtractClaims(kb)
	if err != nil {
		t.Fatalf("ExtractClaims failed: %v", err)
	}
	skillClaims := claimsByKind(extracted, claims.KindTechnicalSkill)
	if len(skillClaims) != 4 {
		t.Fatalf("Expected four distinct skill table claims with duplicate and malformed rows ignored, got %d: %+v", len(skillClaims), skillClaims)
	}

	goClaim := claimWithValue(t, skillClaims, "Go")
	if goClaim.Confidence != 0.99 {
		t.Fatalf("Expected row confidence to be retained, got %.2f", goClaim.Confidence)
	}
	if len(goClaim.SourceObjectIDs) != 1 || goClaim.SourceObjectIDs[0] != "skill:matrix" || len(goClaim.SourceLocations) == 0 {
		t.Fatalf("Expected source object and location provenance, got %+v / %+v", goClaim.SourceObjectIDs, goClaim.SourceLocations)
	}
	if !strings.Contains(goClaim.Statement, "Proficiency: Expert") || !strings.Contains(goClaim.Statement, "Years: 7") || !strings.Contains(goClaim.Statement, "Last Used: Present") {
		t.Fatalf("Expected proficiency, years, and last-used fields in statement, got %q", goClaim.Statement)
	}
	if strings.Join(goClaim.EvidenceObjectIDs, ",") != "ev:go-internal,ev:go-public" {
		t.Fatalf("Expected row evidence IDs sorted before planning policy filter, got %+v", goClaim.EvidenceObjectIDs)
	}

	awsClaim := claimWithValue(t, skillClaims, "AWS")
	if strings.Contains(awsClaim.Statement, "Proficiency:") || strings.Contains(awsClaim.Statement, "Years:") || strings.Contains(awsClaim.Statement, "Last Used:") {
		t.Fatalf("Blank optional cells should not be inferred into the statement, got %q", awsClaim.Statement)
	}

	planRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     planning.PolicyStrictPublic,
		Target:       &planning.TargetProfile{DesiredSkills: []string{"Go", "Kubernetes", "AWS", "Terraform"}},
		MaxSkills:    10,
	})
	if planRes.Plan == nil {
		t.Fatalf("Expected resume plan, got diagnostics: %+v", planRes.Diagnostics)
	}
	if hasTargetGap(planRes.Plan.Gaps, "Go") {
		t.Fatalf("Extracted Go skill should not produce a false target gap: %+v", planRes.Plan.Gaps)
	}
	plannedGo := claimWithValue(t, selectedClaims(planRes.Plan), "Go")
	if strings.Join(plannedGo.EvidenceObjectIDs, ",") != "ev:go-public" {
		t.Fatalf("StrictPublic should filter restricted skill-row evidence IDs, got %+v", plannedGo.EvidenceObjectIDs)
	}

	summaryRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeSkillsSummary,
		PolicyID:     planning.PolicyStrictPublic,
		Target:       &planning.TargetProfile{DesiredSkills: []string{"Terraform"}},
		MaxSkills:    10,
	})
	if summaryRes.Plan == nil {
		t.Fatalf("Expected skills summary plan, got diagnostics: %+v", summaryRes.Diagnostics)
	}
	if len(claimsByKind(selectedClaims(summaryRes.Plan), claims.KindTechnicalSkill)) == 0 {
		t.Fatalf("Expected standalone skills summary to include table-derived skills, got %+v", summaryRes.Plan.SelectedClaims)
	}

	repeatRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     planning.PolicyStrictPublic,
		Target:       &planning.TargetProfile{DesiredSkills: []string{"Go", "Kubernetes", "AWS", "Terraform"}},
		MaxSkills:    10,
	})
	if repeatRes.Plan == nil || strings.Join(claimIDsToStrings(planClaimIDs(planRes.Plan)), ",") != strings.Join(claimIDsToStrings(planClaimIDs(repeatRes.Plan)), ",") {
		t.Fatalf("Expected repeated skill planning to be deterministic")
	}
}

func TestUnsupportedAWSRemainsGapWithOnlyTransferableCloudEvidence(t *testing.T) {
	kb := model.NewKnowledgeBase()
	kb.Objects["skill:cloud-transferable"] = skillMatrixObject("skill:cloud-transferable", []model.Section{
		{
			Heading: "### Cloud Platform Skills",
			Body: `| Skill Name | Proficiency | Confidence | Years | Last Used | Related Experience | Related Projects | Supporting Evidence |
| :--- | :--- | :---: | :---: | :---: | :--- | :--- | :--- |
| **GCP** | Advanced | 0.95 | 5 | Present | None | None | [ev:gcp-public](./evidence.md) |
| **Kubernetes** | Advanced | 0.95 | 5 | Present | None | None | [ev:k8s-public](./evidence.md) |
| **Terraform** | Advanced | 0.95 | 5 | Present | None | None | [ev:terraform-public](./evidence.md) |`,
		},
	})
	kb.Objects["ev:gcp-public"] = evidenceObject("ev:gcp-public", model.VisibilityPublic)
	kb.Objects["ev:k8s-public"] = evidenceObject("ev:k8s-public", model.VisibilityPublic)
	kb.Objects["ev:terraform-public"] = evidenceObject("ev:terraform-public", model.VisibilityPublic)

	planRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     planning.PolicyStrictPublic,
		Target: &planning.TargetProfile{
			DesiredTechnologies: []string{"AWS"},
		},
		MaxSkills: 10,
	})
	if planRes.Plan == nil {
		t.Fatalf("BuildPlan() returned nil: %+v", planRes.Diagnostics)
	}
	if !hasTargetGap(planRes.Plan.Gaps, "AWS") {
		t.Fatalf("AWS requirement did not remain an explicit gap: %+v", planRes.Plan.Gaps)
	}
	for _, planned := range planRes.Plan.SelectedClaims {
		if strings.Contains(strings.ToLower(planned.Claim.Statement), "aws") || strings.EqualFold(string(planned.Claim.Value), "AWS") {
			t.Fatalf("unsupported AWS became a selected direct claim: %+v", planned.Claim)
		}
	}

	renderRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: planRes.Plan})
	if renderRes.Artifact == nil {
		t.Fatalf("Render() returned nil: %+v", renderRes.Diagnostics)
	}
	var output bytes.Buffer
	if err := export.ExportMarkdown(renderRes.Artifact, &output, export.MarkdownOptions{IncludeHeadings: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(output.String()), "aws") {
		t.Fatalf("unsupported AWS leaked into candidate-facing output:\n%s", output.String())
	}
}

func TestMissingSourceMetricIsNotInventedDuringPlanningOrRendering(t *testing.T) {
	kb := model.NewKnowledgeBase()
	kb.Objects["exp:metricless"] = &model.Object{
		ID:         "exp:metricless",
		Type:       model.TypeExperience,
		SourceFile: "experience/metricless.md",
		Metadata: model.Metadata{
			Status:       model.StatusActive,
			Visibility:   model.VisibilityPublic,
			Verification: model.VerificationIndependentlyVerified,
			Confidence:   0.95,
			Lifecycle:    model.LifecycleCompleted,
		},
		Sections: []model.Section{
			{Heading: "## 1. Role Context", Body: "* **Organization**: Example Systems\n* **Role**: Platform Engineer\n* **Duration**: Present"},
			{Heading: "## 2. Key Achievements", Body: "- Improved deployment reliability through safer release automation."},
		},
	}

	planRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType:           planning.TypeCoverLetter,
		PolicyID:               planning.PolicyStrictPublic,
		MaxAchievementsPerRole: 3,
	})
	if planRes.Plan == nil {
		t.Fatalf("BuildPlan() returned nil: %+v", planRes.Diagnostics)
	}
	metricGap := false
	for _, gap := range planRes.Plan.Gaps {
		if gap.Code == "CKB-PLAN-METRIC-OPPORTUNITY" {
			metricGap = true
		}
	}
	if !metricGap {
		t.Fatalf("metricless accomplishment was not identified: gaps=%+v selected=%+v excluded=%+v", planRes.Plan.Gaps, planRes.Plan.SelectedClaims, planRes.Plan.ExcludedClaims)
	}
	renderRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: planRes.Plan})
	if renderRes.Artifact == nil {
		t.Fatalf("Render() returned nil: %+v", renderRes.Diagnostics)
	}
	found := false
	for _, section := range renderRes.Artifact.Sections {
		for _, entry := range section.Entries {
			if strings.Contains(entry.Text, "Improved deployment reliability") {
				found = true
				if strings.ContainsAny(entry.Text, "%$") {
					t.Fatalf("renderer invented a metric: %q", entry.Text)
				}
			}
		}
	}
	if !found {
		t.Fatal("metricless source accomplishment was not rendered")
	}
}

func TestCredentialSubheadingsExtractIndividualClaimsAndRenderOnce(t *testing.T) {
	kb := model.NewKnowledgeBase()
	kb.Objects["cred:matrix"] = credentialObject("cred:matrix", []model.Section{
		{Heading: "## 1. Professional Certifications"},
		{
			Heading: "### Fictional Cloud Architect Professional",
			Body:    "- **Issuer**: Example Cloud\n- **Issue Status**: Active\n- **Issue Date**: 2024-01\n- **Expiration Date**: 2027-01\n- **Related Projects**: [proj:cloud](./projects.md)\n- **Evidence Ref**: [ev:cloud-cert](./evidence.md)\n",
		},
		{
			Heading: "### Fictional Platform Associate",
			Body:    "- **Issuer**: Example Platform\n- **Issue Status**: Expired\n- **Issue Date**: 2020-01\n- **Expiration Date**: 2023-01\n- **Evidence Ref**: [ev:platform-cert](./evidence.md)\n",
		},
		{Heading: "## 2. Professional Training Log"},
		{
			Heading: "### Course: Planned Reliability Training",
			Body:    "- **Provider**: Example Training\n- **Issue Status**: Planned\n- **Type**: Online Learning\n- **Evidence Ref**: [ev:planned-training](./evidence.md)\n",
		},
		{Heading: "### Empty Credential Section"},
	})
	kb.Objects["ev:cloud-cert"] = evidenceObject("ev:cloud-cert", model.VisibilityPublic)
	kb.Objects["ev:platform-cert"] = evidenceObject("ev:platform-cert", model.VisibilityPublic)
	kb.Objects["ev:planned-training"] = evidenceObject("ev:planned-training", model.VisibilityPublic)

	extracted, err := claims.ExtractClaims(kb)
	if err != nil {
		t.Fatalf("ExtractClaims failed: %v", err)
	}
	credentialClaims := claimsByKind(extracted, claims.KindCredential)
	if len(credentialClaims) != 4 {
		t.Fatalf("Expected one credential claim per H3 heading, got %d: %+v", len(credentialClaims), credentialClaims)
	}
	for _, c := range credentialClaims {
		if strings.Contains(c.Statement, "Professional Certifications") || strings.Contains(c.Statement, "Professional Training Log") {
			t.Fatalf("Container heading should not become credential claim: %+v", c)
		}
	}

	active := claimWithValue(t, credentialClaims, "Fictional Cloud Architect Professional")
	if !strings.Contains(active.Statement, "Issuer: Example Cloud") ||
		!strings.Contains(active.Statement, "Status: Active") ||
		!strings.Contains(active.Statement, "Issue Date: 2024-01") ||
		!strings.Contains(active.Statement, "Expiration Date: 2027-01") {
		t.Fatalf("Expected credential fields preserved in statement, got %q", active.Statement)
	}
	if active.Verification != model.VerificationIndependentlyVerified || active.Visibility != model.VisibilityPublic || active.Status != "Active" {
		t.Fatalf("Expected metadata and status preservation, got %+v", active)
	}
	if strings.Join(active.EvidenceObjectIDs, ",") != "ev:cloud-cert" || strings.Join(active.Projects, ",") != "proj:cloud" {
		t.Fatalf("Expected row evidence and project IDs, got evidence=%+v projects=%+v", active.EvidenceObjectIDs, active.Projects)
	}

	expired := claimWithValue(t, credentialClaims, "Fictional Platform Associate")
	if expired.Status != "Expired" || !strings.Contains(expired.Statement, "Status: Expired") {
		t.Fatalf("Expired credential status not preserved: %+v", expired)
	}
	planned := claimWithValue(t, credentialClaims, "Course: Planned Reliability Training")
	if planned.Status != "Planned" || !strings.Contains(planned.Statement, "Status: Planned") {
		t.Fatalf("Planned credential status not preserved: %+v", planned)
	}

	resumeRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     planning.PolicyStrictPublic,
		MaxEducation: 10,
	})
	if resumeRes.Plan == nil {
		t.Fatalf("Expected resume plan, got diagnostics: %+v", resumeRes.Diagnostics)
	}
	if len(claimsByKind(selectedClaims(resumeRes.Plan), claims.KindCredential)) == 0 {
		t.Fatalf("Expected resume planning to receive eligible credentials, got %+v", resumeRes.Plan.SelectedClaims)
	}

	cvRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeCV,
		PolicyID:     planning.PolicyInternalRecord,
	})
	if cvRes.Plan == nil {
		t.Fatalf("Expected CV plan, got diagnostics: %+v", cvRes.Diagnostics)
	}
	if got := len(claimsByKind(selectedClaims(cvRes.Plan), claims.KindCredential)); got != 4 {
		t.Fatalf("Expected CV planning to receive all policy-authorized credentials, got %d", got)
	}
	firstRender := rendering.Render(context.Background(), rendering.RenderRequest{Plan: cvRes.Plan})
	secondRender := rendering.Render(context.Background(), rendering.RenderRequest{Plan: cvRes.Plan})
	if firstRender.Artifact == nil || secondRender.Artifact == nil {
		t.Fatalf("Expected rendered CV artifacts, got diagnostics: %+v / %+v", firstRender.Diagnostics, secondRender.Diagnostics)
	}
	if countRenderedText(firstRender.Artifact, "Fictional Cloud Architect Professional") != 1 {
		t.Fatalf("Expected active credential rendered once, got sections %+v", firstRender.Artifact.Sections)
	}
	if countRenderedText(firstRender.Artifact, "Professional Certifications") != 0 {
		t.Fatalf("Container credential heading should not render as a credential entry")
	}
	var firstJSON, secondJSON bytes.Buffer
	if err := export.ExportArtifactJSON(firstRender.Artifact, &firstJSON); err != nil {
		t.Fatalf("first artifact export failed: %v", err)
	}
	if err := export.ExportArtifactJSON(secondRender.Artifact, &secondJSON); err != nil {
		t.Fatalf("second artifact export failed: %v", err)
	}
	if !bytes.Equal(firstJSON.Bytes(), secondJSON.Bytes()) {
		t.Fatal("Expected repeated credential rendering/export to be deterministic")
	}
	if !strings.Contains(firstJSON.String(), "ev:cloud-cert") || !strings.Contains(firstJSON.String(), "cred:matrix") {
		t.Fatalf("Expected credential render provenance and evidence mapping in artifact JSON, got %s", firstJSON.String())
	}
}

func TestUnresolvedEvidenceIDsAreDroppedFromPublicOutputs(t *testing.T) {
	const unresolvedCanary = "ev:restricted-unresolved-canary"
	kb := model.NewKnowledgeBase()
	kb.Objects["skill:evidence-auth"] = skillMatrixObject("skill:evidence-auth", []model.Section{
		{
			Heading: "### Evidence Authorization Skills",
			Body: `| Skill Name | Proficiency | Confidence | Years | Last Used | Related Experience | Related Projects | Supporting Evidence |
| :--- | :--- | :---: | :---: | :---: | :--- | :--- | :--- |
| **Go** | Expert | 0.99 | 7 | Present | None | None | [ev:public-ok](./evidence.md), [ev:internal-no](./evidence.md), [ev:confidential-no](./evidence.md), [ev:wrong-type](./evidence.md), [ev:bad-state](./evidence.md), [ev:malformed_unresolved](./evidence.md), [ev:restricted-unresolved-canary](./evidence.md) |`,
		},
	})
	kb.Objects["ev:public-ok"] = evidenceObject("ev:public-ok", model.VisibilityPublic)
	kb.Objects["ev:internal-no"] = evidenceObject("ev:internal-no", model.VisibilityInternal)
	kb.Objects["ev:confidential-no"] = evidenceObject("ev:confidential-no", model.VisibilityConfidential)
	kb.Objects["ev:wrong-type"] = &model.Object{
		ID:         "ev:wrong-type",
		Type:       model.TypeSkill,
		SourceFile: "wrong-type.md",
		Metadata: model.Metadata{
			ID:           "ev:wrong-type",
			Type:         model.TypeSkill,
			Status:       model.StatusActive,
			Verification: model.VerificationIndependentlyVerified,
			Confidence:   1.0,
			Visibility:   model.VisibilityPublic,
		},
	}
	kb.Objects["ev:bad-state"] = evidenceObject("ev:bad-state", model.VisibilityPublic)
	kb.Objects["ev:bad-state"].Metadata.Verification = model.VerificationSuperseded

	publicPlan := buildEvidenceAuthPlan(t, kb, planning.PolicyStrictPublic, planning.TypeResume)
	publicClaim := claimWithValue(t, selectedClaims(publicPlan), "Go")
	if strings.Join(publicClaim.EvidenceObjectIDs, ",") != "ev:public-ok" {
		t.Fatalf("StrictPublic should retain only resolved public evidence, got %+v", publicClaim.EvidenceObjectIDs)
	}
	if publicClaim.Confidence != 0.99 || publicClaim.Verification != model.VerificationIndependentlyVerified {
		t.Fatalf("Evidence filtering should not upgrade confidence or verification, got confidence %.2f verification %s", publicClaim.Confidence, publicClaim.Verification)
	}
	if !hasDiagnosticCode(publicPlan.Diagnostics, model.CodeEvidenceUnresolved) {
		t.Fatalf("Expected unresolved evidence diagnostic, got %+v", publicPlan.Diagnostics)
	}

	renderRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: publicPlan})
	if renderRes.Artifact == nil {
		t.Fatalf("Expected public artifact, got diagnostics: %+v", renderRes.Diagnostics)
	}

	var planJSON, artifactJSON, sidecarJSON, markdownOut, textOut bytes.Buffer
	if err := export.ExportPlanJSON(publicPlan, &planJSON); err != nil {
		t.Fatalf("ExportPlanJSON failed: %v", err)
	}
	if err := export.ExportArtifactJSON(renderRes.Artifact, &artifactJSON); err != nil {
		t.Fatalf("ExportArtifactJSON failed: %v", err)
	}
	if err := export.ExportProvenanceSidecar(renderRes.Artifact, &sidecarJSON); err != nil {
		t.Fatalf("ExportProvenanceSidecar failed: %v", err)
	}
	if err := export.ExportMarkdown(renderRes.Artifact, &markdownOut, export.MarkdownOptions{IncludeHeadings: true, DebugProvenance: true}); err != nil {
		t.Fatalf("ExportMarkdown failed: %v", err)
	}
	if err := export.ExportText(renderRes.Artifact, &textOut, export.TextOptions{IncludeHeadings: true, DebugProvenance: true}); err != nil {
		t.Fatalf("ExportText failed: %v", err)
	}

	for label, data := range map[string][]byte{
		"plan JSON":          planJSON.Bytes(),
		"artifact JSON":      artifactJSON.Bytes(),
		"provenance sidecar": sidecarJSON.Bytes(),
		"markdown":           markdownOut.Bytes(),
		"text":               textOut.Bytes(),
	} {
		if bytes.Contains(data, []byte(unresolvedCanary)) {
			t.Fatalf("%s leaked unresolved evidence canary:\n%s", label, string(data))
		}
	}
	for _, refs := range [][]string{renderRes.Artifact.Manifest.EvidenceReferences, publicClaim.EvidenceObjectIDs} {
		if strings.Join(refs, ",") != "ev:public-ok" {
			t.Fatalf("Expected only allowed public evidence in public references, got %+v", refs)
		}
	}

	internalPlan := buildEvidenceAuthPlan(t, kb, planning.PolicyInternalRecord, planning.TypeSkillsSummary)
	internalClaim := claimWithValue(t, selectedClaims(internalPlan), "Go")
	if strings.Join(internalClaim.EvidenceObjectIDs, ",") != "ev:confidential-no,ev:internal-no,ev:public-ok" {
		t.Fatalf("InternalRecord should retain resolved policy-permitted evidence but drop unresolved/invalid IDs, got %+v", internalClaim.EvidenceObjectIDs)
	}
	if !hasDiagnosticCode(internalPlan.Diagnostics, model.CodeEvidenceUnresolved) {
		t.Fatalf("Expected unresolved evidence diagnostic for internal policy, got %+v", internalPlan.Diagnostics)
	}

	requireEvidencePolicy := claims.Policy{
		ID:                  "RequireEvidenceUnit",
		MinVerification:     model.VerificationSelfAttested,
		MinConfidence:       0.80,
		AllowedVisibilities: []model.Visibility{model.VisibilityPublic},
		RequireEvidence:     true,
	}
	noEvidenceClaim := publicClaim
	noEvidenceClaim.EvidenceObjectIDs = nil
	status, code, _ := claims.EvaluateEligibility(noEvidenceClaim, requireEvidencePolicy)
	if status != claims.StatusIneligible || code != claims.CodeClaimMissingEvidence {
		t.Fatalf("RequireEvidence policy should reject claims with no authorized evidence, got %s/%s", status, code)
	}

	repeatPlan := buildEvidenceAuthPlan(t, kb, planning.PolicyStrictPublic, planning.TypeResume)
	if strings.Join(claimIDsToStrings(planClaimIDs(publicPlan)), ",") != strings.Join(claimIDsToStrings(planClaimIDs(repeatPlan)), ",") {
		t.Fatal("Expected repeated unresolved-evidence planning to be deterministic")
	}
}

func TestTitanLikePercentagesDoNotBlockCVRendering(t *testing.T) {
	kb := model.NewKnowledgeBase()
	kb.Objects["exp:titan-like"] = &model.Object{
		ID:         "exp:titan-like",
		Type:       model.TypeExperience,
		SourceFile: "exp:titan-like.md",
		Metadata: model.Metadata{
			ID:           "exp:titan-like",
			Type:         model.TypeExperience,
			Status:       model.StatusActive,
			Verification: model.VerificationIndependentlyVerified,
			Confidence:   0.95,
			Visibility:   model.VisibilityPublic,
			Source:       "Example Platform Group",
			Lifecycle:    model.LifecycleActive,
		},
		Sections: []model.Section{
			{
				Heading: "## 1. Role Context",
				Body:    "Organization: Example Platform Group\nRole: Principal Platform Engineer\nDuration: 2024-01 - 2025-01\n",
			},
			{
				Heading: "## 4. Outcomes & Metrics",
				Body: strings.Join([]string{
					"- Increased utilization from 15% to 62% through scheduling changes.",
					"- Improved configuration remediation to 100% after drift checks.",
					"- Reduced defect escapes by 30% during rollout.",
					"- Improved availability to 99.9% during migration.",
				}, "\n"),
			},
		},
	}

	planRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeCV,
		PolicyID:     planning.PolicyComprehensiveCV,
	})
	if planRes.Plan == nil {
		t.Fatalf("Expected titan-like CV plan, got diagnostics: %+v", planRes.Diagnostics)
	}
	for _, conf := range planRes.Plan.Conflicts {
		if conf.Type == "MetricMismatch" && conf.Blocked {
			t.Fatalf("Unrelated percentage outcomes should not create blocking metric conflicts: %+v", conf)
		}
	}

	renderRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: planRes.Plan})
	if renderRes.Artifact == nil {
		t.Fatalf("Expected titan-like CV artifact, got diagnostics: %+v", renderRes.Diagnostics)
	}
	if countRenderedText(renderRes.Artifact, "100%") == 0 {
		t.Fatalf("Expected rendered CV to include titan-like remediation outcome, got %+v", renderRes.Artifact.Sections)
	}

	repeatRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: planning.TypeCV,
		PolicyID:     planning.PolicyComprehensiveCV,
	})
	if repeatRes.Plan == nil || strings.Join(claimIDsToStrings(planClaimIDs(planRes.Plan)), ",") != strings.Join(claimIDsToStrings(planClaimIDs(repeatRes.Plan)), ",") {
		t.Fatal("Expected repeated titan-like planning to be deterministic")
	}
}

func TestEvidenceCatalogVerificationStateControlsAuthorizedProvenance(t *testing.T) {
	dir := t.TempDir()
	evidencePath := filepath.Join(dir, "evidence.md")
	source := `| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | ev:catalog |
| **Type** | Evidence |
| **Status** | Active |
| **Verification Level** | Independently-Verified |
| **Confidence** | 1.00 |
| **Visibility** | Public |
| **Source** | Synthetic Evidence Catalog |
| **Last Updated** | 2026-07-10 |
| **Lifecycle State** | Active |

---

## 1. Relational Evidence Catalog

| Evidence ID | Evidence Type | Verification Level | Visibility | Storage Location (Fictionalized) | Description |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **ev:independent-row** | Repository | Independently-Verified | Public | local://synthetic/independent | Independent row. |
| **ev:artifact-row** | Repository | Artifact-Supported | Public | local://synthetic/artifact | Artifact row. |
| **ev:self-row** | Repository | Self-Attested | Public | local://synthetic/self | Self row. |
| **ev:disputed-row** | Repository | Disputed | Public | local://synthetic/disputed | Disputed row. |
| **ev:superseded-row** | Repository | Superseded | Public | local://synthetic/superseded | Superseded row. |
| **ev:invalid-row** | Repository | Verified | Public | local://synthetic/invalid | Invalid row. |
`
	if err := os.WriteFile(evidencePath, []byte(source), 0o600); err != nil {
		t.Fatalf("failed to write evidence fixture: %v", err)
	}

	omittedPath := filepath.Join(dir, "evidence_omitted.md")
	omittedSource := strings.Replace(source, "ev:catalog", "ev:catalog-omitted", 1)
	omittedSource = strings.Replace(omittedSource, "| Evidence ID | Evidence Type | Verification Level | Visibility | Storage Location (Fictionalized) | Description |\n| :--- | :--- | :--- | :--- | :--- | :--- |\n| **ev:independent-row** | Repository | Independently-Verified | Public | local://synthetic/independent | Independent row. |\n| **ev:artifact-row** | Repository | Artifact-Supported | Public | local://synthetic/artifact | Artifact row. |\n| **ev:self-row** | Repository | Self-Attested | Public | local://synthetic/self | Self row. |\n| **ev:disputed-row** | Repository | Disputed | Public | local://synthetic/disputed | Disputed row. |\n| **ev:superseded-row** | Repository | Superseded | Public | local://synthetic/superseded | Superseded row. |\n| **ev:invalid-row** | Repository | Verified | Public | local://synthetic/invalid | Invalid row. |\n",
		"| Evidence ID | Evidence Type | Visibility | Storage Location (Fictionalized) | Description |\n| :--- | :--- | :--- | :--- | :--- |\n| **ev:omitted-verification-row** | Repository | Public | local://synthetic/omitted | Omitted verification row. |\n", 1)
	if err := os.WriteFile(omittedPath, []byte(omittedSource), 0o600); err != nil {
		t.Fatalf("failed to write omitted evidence fixture: %v", err)
	}

	parseRes := parser.ParseFiles(context.Background(), []string{evidencePath, omittedPath}, parser.ParseOptions{Limits: parser.DefaultLimits()})
	if !hasDiagnosticCode(parseRes.Diagnostics, model.CodeMetadataInvalidEnum) {
		t.Fatalf("Expected invalid catalog verification diagnostic, got %+v", parseRes.Diagnostics)
	}
	expectedVerification := map[string]model.VerificationLevel{
		"ev:independent-row":          model.VerificationIndependentlyVerified,
		"ev:artifact-row":             model.VerificationArtifactSupported,
		"ev:self-row":                 model.VerificationSelfAttested,
		"ev:disputed-row":             model.VerificationDisputed,
		"ev:superseded-row":           model.VerificationSuperseded,
		"ev:omitted-verification-row": model.VerificationUnverified,
	}
	for id, want := range expectedVerification {
		obj := parseRes.KnowledgeBase.Objects[id]
		if obj == nil {
			t.Fatalf("Expected catalog evidence %s to be registered", id)
		}
		if obj.Metadata.Verification != want {
			t.Fatalf("Expected %s verification %s, got %s", id, want, obj.Metadata.Verification)
		}
	}
	if parseRes.KnowledgeBase.Objects["ev:invalid-row"] != nil {
		t.Fatal("Invalid catalog verification row must not be registered as evidence")
	}

	parseRes.KnowledgeBase.Objects["skill:catalog-auth"] = skillMatrixObject("skill:catalog-auth", []model.Section{
		{
			Heading: "### Catalog Authorization Skill",
			Body: `| Skill Name | Proficiency | Confidence | Years | Last Used | Related Experience | Related Projects | Supporting Evidence |
| :--- | :--- | :---: | :---: | :---: | :--- | :--- | :--- |
| **Go** | Expert | 0.99 | 7 | Present | None | None | [ev:independent-row](./evidence.md), [ev:artifact-row](./evidence.md), [ev:self-row](./evidence.md), [ev:disputed-row](./evidence.md), [ev:superseded-row](./evidence.md), [ev:omitted-verification-row](./evidence.md), [ev:invalid-row](./evidence.md) |`,
		},
	})

	publicPlan := buildEvidenceAuthPlan(t, parseRes.KnowledgeBase, planning.PolicyStrictPublic, planning.TypeResume)
	publicClaim := claimWithValue(t, selectedClaims(publicPlan), "Go")
	if strings.Join(publicClaim.EvidenceObjectIDs, ",") != "ev:artifact-row,ev:independent-row,ev:self-row" {
		t.Fatalf("StrictPublic should retain only policy-authorized catalog evidence, got %+v", publicClaim.EvidenceObjectIDs)
	}
	var planJSON bytes.Buffer
	if err := export.ExportPlanJSON(publicPlan, &planJSON); err != nil {
		t.Fatalf("ExportPlanJSON failed: %v", err)
	}
	for _, forbidden := range []string{"ev:disputed-row", "ev:superseded-row", "ev:omitted-verification-row", "ev:invalid-row"} {
		if strings.Contains(planJSON.String(), forbidden) {
			t.Fatalf("Public plan JSON leaked ineligible evidence %s:\n%s", forbidden, planJSON.String())
		}
	}

	renderRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: publicPlan})
	if renderRes.Artifact == nil {
		t.Fatalf("Expected public artifact from catalog verification fixture, got diagnostics: %+v", renderRes.Diagnostics)
	}
	var sidecar bytes.Buffer
	if err := export.ExportProvenanceSidecar(renderRes.Artifact, &sidecar); err != nil {
		t.Fatalf("ExportProvenanceSidecar failed: %v", err)
	}
	for _, forbidden := range []string{"ev:disputed-row", "ev:superseded-row", "ev:omitted-verification-row", "ev:invalid-row"} {
		if strings.Contains(sidecar.String(), forbidden) {
			t.Fatalf("Public provenance sidecar leaked ineligible evidence %s:\n%s", forbidden, sidecar.String())
		}
	}

	internalPlan := buildEvidenceAuthPlan(t, parseRes.KnowledgeBase, planning.PolicyInternalRecord, planning.TypeSkillsSummary)
	internalClaim := claimWithValue(t, selectedClaims(internalPlan), "Go")
	if strings.Join(internalClaim.EvidenceObjectIDs, ",") != "ev:artifact-row,ev:independent-row,ev:omitted-verification-row,ev:self-row" {
		t.Fatalf("InternalRecord should retain resolved catalog evidence states according to policy, got %+v", internalClaim.EvidenceObjectIDs)
	}

	repeatParse := parser.ParseFiles(context.Background(), []string{evidencePath, omittedPath}, parser.ParseOptions{Limits: parser.DefaultLimits()})
	if repeatParse.KnowledgeBase.Objects["ev:artifact-row"].Metadata.Verification != parseRes.KnowledgeBase.Objects["ev:artifact-row"].Metadata.Verification {
		t.Fatal("Expected repeated catalog verification parsing to be deterministic")
	}
}

func accomplishmentObject(id string, sections []model.Section) *model.Object {
	return &model.Object{
		ID:         id,
		Type:       model.TypeAccomplishment,
		SourceFile: id + ".md",
		Metadata: model.Metadata{
			ID:           id,
			Type:         model.TypeAccomplishment,
			Status:       model.StatusActive,
			Verification: model.VerificationSelfAttested,
			Confidence:   0.9,
			Visibility:   model.VisibilityPublic,
		},
		Sections: sections,
	}
}

func skillMatrixObject(id string, sections []model.Section) *model.Object {
	return &model.Object{
		ID:         id,
		Type:       model.TypeSkill,
		SourceFile: id + ".md",
		Metadata: model.Metadata{
			ID:           id,
			Type:         model.TypeSkill,
			Status:       model.StatusActive,
			Verification: model.VerificationIndependentlyVerified,
			Confidence:   0.95,
			Visibility:   model.VisibilityPublic,
			RelatedEvs:   []string{"ev:object-default"},
		},
		Sections: sections,
	}
}

func credentialObject(id string, sections []model.Section) *model.Object {
	return &model.Object{
		ID:         id,
		Type:       model.TypeCredential,
		SourceFile: id + ".md",
		Metadata: model.Metadata{
			ID:           id,
			Type:         model.TypeCredential,
			Status:       model.StatusActive,
			Verification: model.VerificationIndependentlyVerified,
			Confidence:   1.0,
			Visibility:   model.VisibilityPublic,
			RelatedEvs:   []string{"ev:credential-default"},
		},
		Sections: sections,
	}
}

func evidenceObject(id string, visibility model.Visibility) *model.Object {
	return &model.Object{
		ID:         id,
		Type:       model.TypeEvidence,
		SourceFile: id + ".md",
		Metadata: model.Metadata{
			ID:           id,
			Type:         model.TypeEvidence,
			Status:       model.StatusActive,
			Verification: model.VerificationIndependentlyVerified,
			Confidence:   1.0,
			Visibility:   visibility,
		},
	}
}

func claimsByKind(all []claims.Claim, kind claims.ClaimKind) []claims.Claim {
	var filtered []claims.Claim
	for _, c := range all {
		if c.Kind == kind {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func claimWithValue(t *testing.T, all []claims.Claim, value string) claims.Claim {
	t.Helper()
	for _, c := range all {
		if string(c.Value) == value {
			return c
		}
	}
	t.Fatalf("Expected claim value %q, got %+v", value, all)
	return claims.Claim{}
}

func selectedClaims(plan *planning.ArtifactPlan) []claims.Claim {
	selected := make([]claims.Claim, 0, len(plan.SelectedClaims))
	for _, pc := range plan.SelectedClaims {
		selected = append(selected, pc.Claim)
	}
	return selected
}

func claimsByValue(all []claims.Claim, value string) []claims.Claim {
	var matches []claims.Claim
	for _, c := range all {
		if string(c.Value) == value {
			matches = append(matches, c)
		}
	}
	return matches
}

func planClaimIDs(plan *planning.ArtifactPlan) []claims.ClaimID {
	ids := make([]claims.ClaimID, 0, len(plan.SelectedClaims))
	for _, pc := range plan.SelectedClaims {
		ids = append(ids, pc.Claim.ID)
	}
	return ids
}

func hasTargetGap(gaps []planning.Gap, context string) bool {
	for _, gap := range gaps {
		if gap.Code == "CKB-PLAN-TARGET-GAP" && gap.Context == context {
			return true
		}
	}
	return false
}

func countRenderedText(artifact *rendering.Artifact, needle string) int {
	count := 0
	for _, section := range artifact.Sections {
		for _, entry := range section.Entries {
			if strings.Contains(entry.Text, needle) {
				count++
			}
		}
	}
	return count
}

func buildEvidenceAuthPlan(t *testing.T, kb *model.KnowledgeBase, policyID string, artifactType planning.ArtifactType) *planning.ArtifactPlan {
	t.Helper()
	planRes := planning.BuildPlan(context.Background(), kb, planning.PlanRequest{
		ArtifactType: artifactType,
		PolicyID:     policyID,
		Target:       &planning.TargetProfile{DesiredSkills: []string{"Go"}},
		MaxSkills:    10,
	})
	if planRes.Plan == nil {
		t.Fatalf("Expected evidence authorization plan, got diagnostics: %+v", planRes.Diagnostics)
	}
	return planRes.Plan
}

func hasDiagnosticCode(diagnostics []model.Diagnostic, code model.DiagnosticCode) bool {
	for _, diag := range diagnostics {
		if diag.Code == code {
			return true
		}
	}
	return false
}

func TestEligibilityPolicies(t *testing.T) {
	c := claims.Claim{
		ID:           "claim:test:role:1",
		Kind:         claims.KindRole,
		Statement:    "Test Role",
		Verification: model.VerificationUnverified,
		Confidence:   0.9,
		Visibility:   model.VisibilityInternal,
		Status:       "Active",
	}

	strictPublic, _ := planning.GetPolicyByID(planning.PolicyStrictPublic)
	compCV, _ := planning.GetPolicyByID(planning.PolicyComprehensiveCV)

	// Strict Public should reject due to visibility and verification level
	status, code, _ := claims.EvaluateEligibility(c, strictPublic)
	if status != claims.StatusIneligible || code != claims.CodeClaimPrivate {
		t.Errorf("Strict Public should reject internal visibility with CodeClaimPrivate, got %s / %s", status, code)
	}

	// Comprehensive CV should accept it
	status, _, _ = claims.EvaluateEligibility(c, compCV)
	if status != claims.StatusEligible {
		t.Errorf("Comprehensive CV should accept internal claim, got %s", status)
	}
}

func TestConflictsDetection(t *testing.T) {
	c1 := claims.Claim{
		ID:              "claim:exp:acme:role:1",
		Kind:            claims.KindRole,
		Statement:       "Title A",
		Value:           "Staff Engineer",
		SourceObjectIDs: []string{"exp:acme"},
		TimeRange: &claims.TimeRange{
			Start: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	c2 := claims.Claim{
		ID:              "claim:exp:acme:role:2",
		Kind:            claims.KindRole,
		Statement:       "Title B",
		Value:           "Lead Engineer",
		SourceObjectIDs: []string{"exp:acme"},
		TimeRange: &claims.TimeRange{
			Start: time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	conflicts := claims.DetectConflicts([]claims.Claim{c1, c2})
	if len(conflicts) != 1 {
		t.Fatalf("Expected exactly 1 title mismatch conflict, got %d", len(conflicts))
	}

	if conflicts[0].Type != "TitleMismatch" || !conflicts[0].Blocked {
		t.Errorf("Expected blocked TitleMismatch conflict, got Type=%s, Blocked=%t", conflicts[0].Type, conflicts[0].Blocked)
	}
}

func TestDeduplication(t *testing.T) {
	c1 := claims.Claim{
		ID:              "claim1",
		Kind:            claims.KindTechnicalSkill,
		Statement:       "Skilled in Go Programming",
		SourceObjectIDs: []string{"exp:acme"},
	}
	c2 := claims.Claim{
		ID:              "claim2",
		Kind:            claims.KindTechnicalSkill,
		Statement:       "Skilled in Go Programming  ", // Trailing spaces
		SourceObjectIDs: []string{"proj:phoenix"},
	}

	deduped := claims.DeduplicateClaims([]claims.Claim{c1, c2})
	if len(deduped) != 1 {
		t.Fatalf("Expected duplicate claims to merge into 1, got %d", len(deduped))
	}

	m := deduped[0]
	if len(m.SourceObjectIDs) != 2 || m.SourceObjectIDs[0] != "exp:acme" || m.SourceObjectIDs[1] != "proj:phoenix" {
		t.Errorf("Expected merged claim to aggregate source IDs, got %v", m.SourceObjectIDs)
	}
}

func TestPolicySafeDeduplicationPreservesPublicDuplicate(t *testing.T) {
	kb := model.NewKnowledgeBase()
	kb.Objects["skill:public-duplicate"] = skillMatrixObject("skill:public-duplicate", []model.Section{{
		Heading: "## 1. Skill Matrix by Domain",
		Body: `| Skill Name | Proficiency | Confidence | Years | Last Used | Related Experience | Related Projects | Supporting Evidence |
| :--- | :--- | :---: | :---: | :---: | :--- | :--- | :--- |
| **Go** | Advanced | 0.95 | 5 | Present | None | None | [ev:public-duplicate](./evidence.md) |`,
	}})
	kb.Objects["skill:internal-duplicate"] = skillMatrixObject("skill:internal-duplicate", []model.Section{{
		Heading: "## 1. Skill Matrix by Domain",
		Body: `| Skill Name | Proficiency | Confidence | Years | Last Used | Related Experience | Related Projects | Supporting Evidence |
| :--- | :--- | :---: | :---: | :---: | :--- | :--- | :--- |
| **Go** | Advanced | 0.95 | 5 | Present | None | None | [ev:restricted-dedupe-canary](./evidence.md) |`,
	}})
	kb.Objects["skill:confidential-duplicate"] = skillMatrixObject("skill:confidential-duplicate", []model.Section{{
		Heading: "## 1. Skill Matrix by Domain",
		Body: `| Skill Name | Proficiency | Confidence | Years | Last Used | Related Experience | Related Projects | Supporting Evidence |
| :--- | :--- | :---: | :---: | :---: | :--- | :--- | :--- |
| **Go** | Advanced | 0.95 | 5 | Present | None | None | [ev:confidential-dedupe-canary](./evidence.md) |`,
	}})
	kb.Objects["skill:internal-duplicate"].Metadata.Visibility = model.VisibilityInternal
	kb.Objects["skill:confidential-duplicate"].Metadata.Visibility = model.VisibilityConfidential
	kb.Objects["ev:public-duplicate"] = evidenceObject("ev:public-duplicate", model.VisibilityPublic)
	kb.Objects["ev:restricted-dedupe-canary"] = evidenceObject("ev:restricted-dedupe-canary", model.VisibilityInternal)
	kb.Objects["ev:confidential-dedupe-canary"] = evidenceObject("ev:confidential-dedupe-canary", model.VisibilityConfidential)

	publicPlan := buildEvidenceAuthPlan(t, kb, planning.PolicyStrictPublic, planning.TypeSkillsSummary)
	publicGoClaims := claimsByValue(selectedClaims(publicPlan), "Go")
	if len(publicGoClaims) != 1 {
		t.Fatalf("Expected public duplicate to survive once under StrictPublic, got %+v", publicGoClaims)
	}
	if strings.Join(publicGoClaims[0].SourceObjectIDs, ",") != "skill:public-duplicate" {
		t.Fatalf("Restricted duplicate provenance leaked into public claim: %+v", publicGoClaims[0].SourceObjectIDs)
	}
	if strings.Join(publicGoClaims[0].EvidenceObjectIDs, ",") != "ev:public-duplicate" {
		t.Fatalf("Restricted duplicate evidence leaked into public claim: %+v", publicGoClaims[0].EvidenceObjectIDs)
	}

	renderRes := rendering.Render(context.Background(), rendering.RenderRequest{Plan: publicPlan})
	if renderRes.Artifact == nil {
		t.Fatalf("Expected public skills artifact, got diagnostics: %+v", renderRes.Diagnostics)
	}
	var planJSON, artifactJSON, sidecarJSON, markdownOut, textOut bytes.Buffer
	if err := export.ExportPlanJSON(publicPlan, &planJSON); err != nil {
		t.Fatalf("ExportPlanJSON failed: %v", err)
	}
	if err := export.ExportArtifactJSON(renderRes.Artifact, &artifactJSON); err != nil {
		t.Fatalf("ExportArtifactJSON failed: %v", err)
	}
	if err := export.ExportProvenanceSidecar(renderRes.Artifact, &sidecarJSON); err != nil {
		t.Fatalf("ExportProvenanceSidecar failed: %v", err)
	}
	if err := export.ExportMarkdown(renderRes.Artifact, &markdownOut, export.MarkdownOptions{IncludeHeadings: true, DebugProvenance: true}); err != nil {
		t.Fatalf("ExportMarkdown failed: %v", err)
	}
	if err := export.ExportText(renderRes.Artifact, &textOut, export.TextOptions{IncludeHeadings: true, DebugProvenance: true}); err != nil {
		t.Fatalf("ExportText failed: %v", err)
	}
	for label, data := range map[string][]byte{
		"plan JSON":          planJSON.Bytes(),
		"artifact JSON":      artifactJSON.Bytes(),
		"provenance sidecar": sidecarJSON.Bytes(),
		"markdown":           markdownOut.Bytes(),
		"text":               textOut.Bytes(),
	} {
		for _, restricted := range []string{"skill:internal-duplicate", "skill:confidential-duplicate", "ev:restricted-dedupe-canary", "ev:confidential-dedupe-canary"} {
			if bytes.Contains(data, []byte(restricted)) {
				t.Fatalf("%s leaked restricted duplicate canary %s:\n%s", label, restricted, string(data))
			}
		}
	}

	internalPlan := buildEvidenceAuthPlan(t, kb, planning.PolicyInternalRecord, planning.TypeSkillsSummary)
	internalGoClaims := claimsByValue(selectedClaims(internalPlan), "Go")
	if len(internalGoClaims) != 3 {
		t.Fatalf("Expected internal policy to retain all authorization variants, got %+v", internalGoClaims)
	}

	repeatPlan := buildEvidenceAuthPlan(t, kb, planning.PolicyStrictPublic, planning.TypeSkillsSummary)
	if strings.Join(claimIDsToStrings(planClaimIDs(publicPlan)), ",") != strings.Join(claimIDsToStrings(planClaimIDs(repeatPlan)), ",") {
		t.Fatal("Expected repeated policy-safe dedupe planning to be deterministic")
	}
}

func TestRelevanceScoring(t *testing.T) {
	c := claims.Claim{
		ID:        "claim:skill:go",
		Kind:      claims.KindTechnicalSkill,
		Statement: "Experienced Go Developer",
		Value:     "Go",
		Skills:    []string{"Go"},
	}

	target := &planning.TargetProfile{
		RoleTitle:     "Go Developer",
		DesiredSkills: []string{"Go", "Rust"},
	}

	score, components := planning.ScoreClaim(c, target, false)
	// Base (50) + Skill Match (15) + Role Match (15) = 80
	if score != 80 {
		t.Errorf("Expected score 80, got %d. Components: %v", score, components)
	}
}

func TestArtifactPlanning(t *testing.T) {
	// Parse a standard valid catalog directory
	rootFiles, err := parser.DiscoverFiles(filepath.Join(".."))
	if err != nil {
		t.Fatalf("failed to discover root files: %v", err)
	}

	// Add fixture file to complete links
	fixtures := []string{
		filepath.Join("..", "tests", "fixtures", "valid", "complete_experience.md"),
		filepath.Join("..", "tests", "fixtures", "valid", "complete_project.md"),
		filepath.Join("..", "tests", "fixtures", "valid", "evidence_graph.md"),
	}
	files := append(rootFiles, fixtures...)

	res := parser.ParseFiles(context.Background(), files, parser.ParseOptions{Limits: parser.DefaultLimits()})
	if len(res.Diagnostics) > 0 {
		// Log errors, but check if we can proceed
		for _, d := range res.Diagnostics {
			if d.Severity == model.SeverityFatal || d.Severity == model.SeverityError {
				t.Fatalf("Failed to parse: %v", d.Message)
			}
		}
	}

	// Build a plan request for Resume
	req := planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     planning.PolicyStrictPublic,
		Target: &planning.TargetProfile{
			RoleTitle:           "Lead Infrastructure Engineer",
			DesiredSkills:       []string{"Go", "Redis", "Docker", "Envoy"},
			DesiredTechnologies: []string{"GCP"},
		},
	}

	planRes := planning.BuildPlan(context.Background(), res.KnowledgeBase, req)
	if planRes.Plan == nil {
		t.Fatal("Expected non-nil ArtifactPlan result")
	}
	t.Logf("Selected Claims count: %d", len(planRes.Plan.SelectedClaims))
	for _, pc := range planRes.Plan.SelectedClaims {
		t.Logf("  Selected: ID=%s, Kind=%s, Statement=%q, Skills=%v", pc.Claim.ID, pc.Claim.Kind, pc.Claim.Statement, pc.Claim.Skills)
	}
	// t.Logf("Excluded Claims count: %d", len(planRes.Plan.ExcludedClaims))
	// for _, ec := range planRes.Plan.ExcludedClaims {
	// 	t.Logf("  Excluded: ID=%s, Reason=%s, Code=%s", ec.ClaimID, ec.ExclusionReason, ec.ExclusionReasonCode)
	// }
	t.Logf("Diagnostics count: %d", len(planRes.Diagnostics))
	for _, d := range planRes.Diagnostics {
		t.Logf("  Diag: Code=%s, Message=%s", d.Code, d.Message)
	}

	// Verify Resume planner constraints (Max 3 roles, budget filters applied)
	roleCount := 0
	for _, pc := range planRes.Plan.SelectedClaims {
		if pc.Claim.Kind == claims.KindRole {
			roleCount++
		}
	}
	if roleCount > 3 {
		t.Errorf("OnePageResume violated role budget limits (got %d)", roleCount)
	}

	// Verify gaps are reported
	hasGap := false
	for _, g := range planRes.Plan.Gaps {
		if g.Code == "CKB-PLAN-METRIC-OPPORTUNITY" || g.Code == "CKB-PLAN-CHRONOLOGY-GAP" {
			hasGap = true
		}
	}
	if !hasGap {
		t.Error("Expected planning gap report warnings, got 0")
	}
}

func TestResumeSelectionTreatsRoleAndEmploymentAsOneLogicalRole(t *testing.T) {
	policy, _ := planning.GetPolicyByID(planning.PolicyStrictPublic)
	tr := &claims.TimeRange{
		Start: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	role := selectionClaim(claims.KindRole, "claim:exp-one:role", "Served as Platform Engineer at Example Systems", "exp:one", "Example Systems", nil)
	employment := selectionClaim(claims.KindEmployment, "claim:exp-one:employment", "Employed at Example Systems", "exp:one", "Example Systems", tr)
	achievement := selectionClaim(claims.KindAccomplishment, "claim:exp-one:achievement", "Reduced incidents once.", "exp:one", "", nil)
	scores := map[string]int{
		string(employment.ID):  100,
		string(role.ID):        80,
		string(achievement.ID): 70,
	}

	selected, excluded := planning.SelectClaimsForArtifact([]claims.Claim{employment, role, achievement}, planning.PlanRequest{
		ArtifactType:           planning.TypeResume,
		MaxRoles:               1,
		MaxAchievementsPerRole: 2,
	}, scores, policy)

	var anchors []planning.PlannedClaim
	achievementCount := 0
	for _, pc := range selected {
		if pc.Claim.Kind == claims.KindRole || pc.Claim.Kind == claims.KindEmployment {
			anchors = append(anchors, pc)
		}
		if pc.Claim.Kind == claims.KindAccomplishment {
			achievementCount++
		}
	}
	if len(anchors) != 1 {
		t.Fatalf("Expected one logical role anchor, got %d selected claims: %+v", len(anchors), selected)
	}
	if anchors[0].Claim.Kind != claims.KindRole {
		t.Fatalf("Expected role claim to be preferred over higher-scored employment claim, got %s", anchors[0].Claim.Kind)
	}
	if anchors[0].Claim.TimeRange == nil || !anchors[0].Claim.TimeRange.Start.Equal(tr.Start) {
		t.Fatalf("Expected role anchor to retain chronology from employment fallback, got %+v", anchors[0].Claim.TimeRange)
	}
	if achievementCount != 1 {
		t.Fatalf("Expected associated achievement to be selected once, got %d selected claims: %+v", achievementCount, selected)
	}
	if !excludedClaimPresent(excluded, employment.ID) {
		t.Fatalf("Expected duplicate employment anchor to be excluded for auditability, got %+v", excluded)
	}

	artifact := renderSelectedClaims(t, planning.TypeResume, selected)
	if countExperienceEntriesByKind(artifact, "subheader") != 1 {
		t.Fatalf("Expected one resume role subheader, got entries %+v", artifact.Sections)
	}
	if countExperienceEntriesByKind(artifact, "bullet") != 1 {
		t.Fatalf("Expected achievement to render once, got entries %+v", artifact.Sections)
	}
}

func TestEmploymentOnlyExperienceIsResumeFallback(t *testing.T) {
	policy, _ := planning.GetPolicyByID(planning.PolicyStrictPublic)
	employment := selectionClaim(claims.KindEmployment, "claim:exp-only:employment", "Employed at Example Systems", "exp:employment-only", "Example Systems", &claims.TimeRange{
		Start: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	selected, _ := planning.SelectClaimsForArtifact([]claims.Claim{employment}, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		MaxRoles:     1,
	}, map[string]int{string(employment.ID): 50}, policy)

	if len(selected) != 1 || selected[0].Claim.Kind != claims.KindEmployment {
		t.Fatalf("Expected employment-only source to render through employment fallback, got %+v", selected)
	}
	artifact := renderSelectedClaims(t, planning.TypeResume, selected)
	if countExperienceEntriesByKind(artifact, "subheader") != 1 {
		t.Fatalf("Expected one employment fallback subheader, got entries %+v", artifact.Sections)
	}
}

func TestResumeSelectionKeepsDistinctSourcesAndPromotionGrouping(t *testing.T) {
	policy, _ := planning.GetPolicyByID(planning.PolicyStrictPublic)
	roleA := selectionClaim(claims.KindRole, "claim:exp-a:role", "Served as Senior Engineer at Example Systems", "exp:a", "Example Systems", &claims.TimeRange{
		Start: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	roleB := selectionClaim(claims.KindRole, "claim:exp-b:role", "Served as Staff Engineer at Example Systems", "exp:b", "Example Systems", &claims.TimeRange{
		Start: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	selected, _ := planning.SelectClaimsForArtifact([]claims.Claim{roleA, roleB}, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		MaxRoles:     2,
	}, map[string]int{string(roleA.ID): 80, string(roleB.ID): 90}, policy)

	anchorCount := 0
	for _, pc := range selected {
		if pc.Claim.Kind == claims.KindRole || pc.Claim.Kind == claims.KindEmployment {
			anchorCount++
		}
	}
	if anchorCount != 2 {
		t.Fatalf("Expected two distinct source IDs to produce two logical roles, got %+v", selected)
	}
	artifact := renderSelectedClaims(t, planning.TypeResume, selected)
	if countExperienceEntriesByKind(artifact, "header") != 1 || countExperienceEntriesByKind(artifact, "subheader") != 2 {
		t.Fatalf("Expected promotion grouping under one organization header with two roles, got entries %+v", artifact.Sections)
	}
}

func TestCVSelectionDeduplicatesRoleAndEmploymentAnchors(t *testing.T) {
	policy, _ := planning.GetPolicyByID(planning.PolicyComprehensiveCV)
	role := selectionClaim(claims.KindRole, "claim:cv:role", "Served as Platform Engineer at Example Systems", "exp:cv", "Example Systems", nil)
	employment := selectionClaim(claims.KindEmployment, "claim:cv:employment", "Employed at Example Systems", "exp:cv", "Example Systems", &claims.TimeRange{
		Start: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	selected, _ := planning.SelectClaimsForArtifact([]claims.Claim{employment, role}, planning.PlanRequest{
		ArtifactType: planning.TypeCV,
	}, map[string]int{string(employment.ID): 100, string(role.ID): 80}, policy)

	anchorCount := 0
	for _, pc := range selected {
		if pc.Claim.Kind == claims.KindRole || pc.Claim.Kind == claims.KindEmployment {
			anchorCount++
			if pc.Claim.Kind != claims.KindRole {
				t.Fatalf("Expected CV to use role as the single experience anchor when present, got %s", pc.Claim.Kind)
			}
		}
	}
	if anchorCount != 1 {
		t.Fatalf("Expected CV to deduplicate role/employment anchors, got %+v", selected)
	}
}

func TestPlanJSONExport(t *testing.T) {
	c := claims.Claim{
		ID:        "claim:test",
		Kind:      claims.KindTechnicalSkill,
		Statement: "Deterministic JSON testing",
		Status:    "Active",
	}
	p := planning.ArtifactPlan{
		ID:           "plan:test",
		ArtifactType: planning.TypeResume,
		SelectedClaims: []planning.PlannedClaim{
			{Claim: c, SelectionReason: "reason", RelevanceScore: 90, Section: "Skills", Rank: 1},
		},
		Gaps: []planning.Gap{
			{Code: "GAP01", Severity: "Warning", Description: "Gap description"},
		},
	}

	var w1, w2 bytes.Buffer
	err1 := export.ExportPlanJSON(&p, &w1)
	err2 := export.ExportPlanJSON(&p, &w2)

	if err1 != nil || err2 != nil {
		t.Fatalf("ExportPlanJSON failed: %v / %v", err1, err2)
	}

	if !bytes.Equal(w1.Bytes(), w2.Bytes()) {
		t.Error("ExportPlanJSON failed determinism check: successive runs produced different outputs")
	}
}

func selectionClaim(kind claims.ClaimKind, id claims.ClaimID, statement string, sourceID string, organization string, tr *claims.TimeRange) claims.Claim {
	c := claims.Claim{
		ID:              id,
		Kind:            kind,
		Statement:       statement,
		SourceObjectIDs: []string{sourceID},
		SourceLocations: []model.SourceLocation{{FilePath: sourceID + ".md", Line: 1}},
		Visibility:      model.VisibilityPublic,
		Verification:    model.VerificationSelfAttested,
		Confidence:      0.9,
		TimeRange:       tr,
		Status:          "Active",
	}
	if organization != "" {
		c.Organizations = []string{organization}
	}
	return c
}

func excludedClaimPresent(excluded []planning.ExcludedClaim, id claims.ClaimID) bool {
	for _, ex := range excluded {
		if ex.ClaimID == id {
			return true
		}
	}
	return false
}

func renderSelectedClaims(t *testing.T, artifactType planning.ArtifactType, selected []planning.PlannedClaim) *rendering.Artifact {
	t.Helper()
	res := rendering.Render(context.Background(), rendering.RenderRequest{
		Plan: &planning.ArtifactPlan{
			ID:            "plan:selection-render-test",
			ArtifactType:  artifactType,
			SchemaVersion: "1.0",
			Policy: claims.Policy{
				ID:                  planning.PolicyStrictPublic,
				AllowedVisibilities: []model.Visibility{model.VisibilityPublic},
				MinVerification:     model.VerificationSelfAttested,
				MinConfidence:       0.8,
			},
			SelectedClaims: selected,
		},
	})
	if res.Artifact == nil {
		t.Fatalf("Expected rendered artifact, got diagnostics: %+v", res.Diagnostics)
	}
	return res.Artifact
}

func countExperienceEntriesByKind(artifact *rendering.Artifact, kind string) int {
	count := 0
	for _, sec := range artifact.Sections {
		if sec.Kind != "Experience" {
			continue
		}
		for _, entry := range sec.Entries {
			if entry.Kind == kind {
				count++
			}
		}
	}
	return count
}
