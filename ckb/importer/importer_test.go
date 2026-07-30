package importer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/admbahm/theForge/ckb/claims"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/parser"
	"github.com/admbahm/theForge/ckb/planning"
)

const masterResumeFixture = `# Candidate Name
candidate@example.com | (555) 555-0199

# MASTER RESUME

## Senior Engineering Manager | Platform Engineering

## PROFESSIONAL SUMMARY
Engineering leader focused on reliable delivery systems.

# CORE COMPETENCIES

## Cloud & Infrastructure
Kubernetes, Terraform, Google Cloud Platform

## Programming Languages
Go, Python

# PROFESSIONAL EXPERIENCE

## Example Systems

### Senior Engineering Manager

June 2022 – Present
Remote

Led platform engineering and delivery reliability.

#### Leadership Impact
* Mentored engineering leaders and improved delivery planning.

#### Technical Impact
* Built CI/CD workflows using GitHub Actions.

## Earlier Systems

### Quality Engineering Manager

May 2020 – June 2022
Remote

#### Key Contributions
* Modernized Android automation using Kotlin and Appium.

# PLATFORM, CLOUD & AI PROJECTS

## Local AI Infrastructure Platform
Built a local model orchestration platform using containers and GPUs.

# EDUCATION
Bachelor of Science, Example University

# CERTIFICATIONS
Cloud Architecture Certificate
`

func TestImportCreatesReviewableValidatedCKBWithoutChangingSource(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "master.md")
	if err := os.WriteFile(sourcePath, []byte(masterResumeFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(root, "review-ckb")
	report, err := Import(Options{
		SourcePath:  sourcePath,
		OutputDir:   outputDir,
		LastUpdated: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.Experiences != 2 || report.Projects != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	sourceAfter, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(sourceAfter) != masterResumeFixture {
		t.Fatal("source resume was modified")
	}

	parsed := parser.ParseDirectory(context.Background(), outputDir, parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	})
	if model.HasBlockingDiagnostics(parsed.Diagnostics) {
		t.Fatalf("imported CKB failed validation: %+v", parsed.Diagnostics)
	}
	extracted, err := claims.ExtractClaims(parsed.KnowledgeBase)
	if err != nil {
		t.Fatal(err)
	}
	foundKubernetes := false
	foundAccomplishment := false
	for _, claim := range extracted {
		if strings.EqualFold(string(claim.Value), "Kubernetes") {
			foundKubernetes = true
		}
		if string(claim.Value) == "85" {
			t.Fatal("skill table columns collapsed into a confidence-value claim")
		}
		if claim.Kind == claims.KindAccomplishment && strings.Contains(claim.Statement, "Built CI/CD workflows") {
			foundAccomplishment = true
		}
	}
	if !foundKubernetes {
		t.Fatalf("imported Kubernetes skill was not extracted: %+v", extracted)
	}
	if !foundAccomplishment {
		t.Fatalf("imported achievement was not extracted: %+v", extracted)
	}
	planResult := planning.BuildPlan(context.Background(), parsed.KnowledgeBase, planning.PlanRequest{
		ArtifactType: planning.TypeResume,
		PolicyID:     planning.PolicyInternalRecord,
		MaxRoles:     2,
	})
	if planResult.Plan == nil {
		t.Fatalf("imported CKB did not produce a review plan: %+v", planResult.Diagnostics)
	}
	plannedAccomplishment := false
	for _, planned := range planResult.Plan.SelectedClaims {
		if planned.Claim.Kind == claims.KindAccomplishment {
			plannedAccomplishment = true
		}
	}
	if !plannedAccomplishment {
		t.Fatalf("resume plan omitted all imported accomplishments: %+v", planResult.Plan.SelectedClaims)
	}
	data, err := os.ReadFile(filepath.Join(outputDir, "experience", "example-systems-senior-engineering-manager.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "**Status** | Draft") || !strings.Contains(text, "Built CI/CD workflows using GitHub Actions") || !strings.Contains(text, "**Duration**: 2022-06 – Present") {
		t.Fatalf("unexpected experience output:\n%s", text)
	}
	if strings.Contains(text, "candidate@example.com") || strings.Contains(text, "555-0199") {
		t.Fatal("contact PII leaked into imported CKB")
	}
	if _, err := Import(Options{SourcePath: sourcePath, OutputDir: outputDir}); err == nil {
		t.Fatal("expected existing output directory to be rejected")
	}
}

func TestImportReadyProducesActiveSelfAttestedRecords(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "master.md")
	if err := os.WriteFile(sourcePath, []byte(masterResumeFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(root, "ready-ckb")
	report, err := Import(Options{
		SourcePath:  sourcePath,
		OutputDir:   outputDir,
		Ready:       true,
		LastUpdated: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready {
		t.Fatal("ready import did not report ready state")
	}
	data, err := os.ReadFile(filepath.Join(outputDir, "skills.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "**Status** | Active") || !strings.Contains(string(data), "**Verification Level** | Self-Attested") {
		t.Fatalf("ready metadata missing:\n%s", data)
	}
}
