package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/rendering"
	"github.com/admbahm/theForge/pkg/models"
)

type fakeIntelGenerator struct {
	intel         string
	calls         atomic.Int32
	started       chan struct{}
	release       chan struct{}
	optimizeCalls atomic.Int32
}

func (f *fakeIntelGenerator) GenerateIntel(_ context.Context, _ models.JobPost) (string, error) {
	f.calls.Add(1)
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
	}
	if f.release != nil {
		<-f.release
	}
	return f.intel, nil
}

func (f *fakeIntelGenerator) OptimizeVRAM(ctx context.Context, targetModel string) error {
	f.optimizeCalls.Add(1)
	return nil
}

func TestInitialScanGeneratesIntelForFavoriteJob(t *testing.T) {
	vault := t.TempDir()
	path := filepath.Join(vault, "nested", "job.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	input := `---
job_id: R123
company: Example
title: Engineer
state: favorite
custom_field: preserved
---

# Engineer

Build systems.
`
	if err := os.WriteFile(path, []byte(input), 0o640); err != nil {
		t.Fatal(err)
	}

	generator := &fakeIntelGenerator{intel: "### Role Summary\nStrong match."}
	orchestrator, err := NewOrchestrator(vault, generator)
	if err != nil {
		t.Fatal(err)
	}
	defer orchestrator.Stop()

	if err := orchestrator.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		updated, err := os.ReadFile(path)
		return err == nil && strings.Contains(string(updated), "state: intel-ready")
	})

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(updated)
	for _, expected := range []string{"state: intel-ready", "custom_field: preserved", "## The Forge Intelligence", "Strong match.", "analysis_confidence:", "Confidence Reasoning"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("updated note missing %q:\n%s", expected, text)
		}
	}
	if generator.calls.Load() != 1 {
		t.Fatalf("generator calls = %d, want 1", generator.calls.Load())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("permissions = %o, want 640", info.Mode().Perm())
	}
}

func TestInitialScanSkipsUnselectedJob(t *testing.T) {
	vault := t.TempDir()
	path := filepath.Join(vault, "job.md")
	if err := os.WriteFile(path, []byte("---\ncompany: Example\nstate: new\n---\n\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	generator := &fakeIntelGenerator{intel: "unused"}
	orchestrator, err := NewOrchestrator(vault, generator)
	if err != nil {
		t.Fatal(err)
	}
	defer orchestrator.Stop()

	if err := orchestrator.SetTier("frontier"); err != nil {
		t.Fatal(err)
	}

	if err := orchestrator.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return orchestrator.pendingCount() == 0 })
	if generator.calls.Load() != 0 {
		t.Fatalf("generator calls = %d, want 0", generator.calls.Load())
	}
}

func TestOrchestratorTiers(t *testing.T) {
	// 1. Test local tier: new -> processed
	t.Run("local tier", func(t *testing.T) {
		vault := t.TempDir()
		path := filepath.Join(vault, "job.md")
		input := "---\ncompany: Example\nstate: new\n---\nBody"
		if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
			t.Fatal(err)
		}

		generator := &fakeIntelGenerator{intel: "Local Intel"}
		orchestrator, err := NewOrchestrator(vault, generator)
		if err != nil {
			t.Fatal(err)
		}
		defer orchestrator.Stop()

		if err := orchestrator.SetTier("local"); err != nil {
			t.Fatal(err)
		}
		if err := orchestrator.Start(); err != nil {
			t.Fatal(err)
		}

		waitFor(t, func() bool {
			data, err := os.ReadFile(path)
			return err == nil && strings.Contains(string(data), "state: processed")
		})

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "Local Intel") {
			t.Fatalf("expected Local Intel: %s", string(data))
		}
	})

	// 2. Test frontier tier: favorite -> intel-ready
	t.Run("frontier tier", func(t *testing.T) {
		vault := t.TempDir()
		path := filepath.Join(vault, "job.md")
		input := "---\ncompany: Example\nstate: favorite\n---\nBody"
		if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
			t.Fatal(err)
		}

		generator := &fakeIntelGenerator{intel: "Frontier Intel"}
		orchestrator, err := NewOrchestrator(vault, generator)
		if err != nil {
			t.Fatal(err)
		}
		defer orchestrator.Stop()

		if err := orchestrator.SetTier("frontier"); err != nil {
			t.Fatal(err)
		}
		if err := orchestrator.Start(); err != nil {
			t.Fatal(err)
		}

		waitFor(t, func() bool {
			data, err := os.ReadFile(path)
			return err == nil && strings.Contains(string(data), "state: intel-ready")
		})

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "Frontier Intel") {
			t.Fatalf("expected Frontier Intel: %s", string(data))
		}
	})
}

func TestDuplicateEventsDoNotGenerateIntelTwice(t *testing.T) {
	vault := t.TempDir()
	path := filepath.Join(vault, "job.md")
	if err := os.WriteFile(path, []byte("---\ncompany: Example\nstate: favorite\n---\n\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	generator := &fakeIntelGenerator{
		intel:   "### Role Summary\nStrong match.",
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	orchestrator, err := NewOrchestrator(vault, generator)
	if err != nil {
		t.Fatal(err)
	}
	defer orchestrator.Stop()

	if err := orchestrator.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-generator.started:
	case <-time.After(2 * time.Second):
		t.Fatal("generation did not start")
	}

	for range 10 {
		orchestrator.enqueue(path)
	}
	close(generator.release)
	waitFor(t, func() bool {
		data, err := os.ReadFile(path)
		return err == nil && strings.Contains(string(data), "state: intel-ready")
	})

	if generator.calls.Load() != 1 {
		t.Fatalf("generator calls = %d, want 1", generator.calls.Load())
	}
}

func TestConcurrentJobProcessing(t *testing.T) {
	vault := t.TempDir()

	// Create multiple job posts
	jobCount := 5
	paths := make([]string, jobCount)
	for i := range jobCount {
		paths[i] = filepath.Join(vault, fmt.Sprintf("job_%d.md", i))
		input := fmt.Sprintf(`---
company: Company%d
title: Role%d
state: favorite
---
`, i, i)
		if err := os.WriteFile(paths[i], []byte(input), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	startedChan := make(chan struct{}, jobCount)
	releaseChan := make(chan struct{})
	generator := &fakeIntelGenerator{
		intel:   "### Role Summary\nProcessed.",
		started: startedChan,
		release: releaseChan,
	}

	// Use 3 workers
	orchestrator, err := NewOrchestratorWithConcurrency(vault, generator, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer orchestrator.Stop()

	if err := orchestrator.Start(); err != nil {
		t.Fatal(err)
	}

	// Wait until at least 3 tasks have concurrently started processing (our worker limit limit)
	startedCount := 0
	deadline := time.Now().Add(2 * time.Second)
	for startedCount < 3 && time.Now().Before(deadline) {
		select {
		case <-startedChan:
			startedCount++
		case <-time.After(10 * time.Millisecond):
		}
	}

	if startedCount < 3 {
		t.Fatalf("Expected at least 3 concurrent workers to start processing, but got %d", startedCount)
	}

	// Now release them all
	close(releaseChan)

	// Wait for all to finish
	waitFor(t, func() bool {
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(data), "state: intel-ready") {
				return false
			}
		}
		return true
	})

	if generator.calls.Load() != int32(jobCount) {
		t.Fatalf("generator calls = %d, want %d", generator.calls.Load(), jobCount)
	}
}

func (o *Orchestrator) pendingCount() int {
	o.pendingMu.Lock()
	defer o.pendingMu.Unlock()
	return len(o.pending)
}

func TestOrchestratorOptimizeVRAM(t *testing.T) {
	vault := t.TempDir()
	path := filepath.Join(vault, "job.md")
	input := "---\ncompany: Example\nstate: new\n---\nBody"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	generator := &fakeIntelGenerator{intel: "Intel"}
	orchestrator, err := NewOrchestrator(vault, generator)
	if err != nil {
		t.Fatal(err)
	}
	defer orchestrator.Stop()

	if err := orchestrator.SetTier("local"); err != nil {
		t.Fatal(err)
	}
	if err := orchestrator.Start(); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool {
		data, err := os.ReadFile(path)
		return err == nil && strings.Contains(string(data), "state: processed")
	})

	if generator.optimizeCalls.Load() != 1 {
		t.Fatalf("expected 1 call to OptimizeVRAM, got %d", generator.optimizeCalls.Load())
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

func TestOrchestrator_ProcessApply(t *testing.T) {
	// Create mock CKB directory
	ckbDir := t.TempDir()
	expDir := filepath.Join(ckbDir, "experience")
	if err := os.MkdirAll(expDir, 0755); err != nil {
		t.Fatal(err)
	}

	mockExp := `| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | exp:stark-devops |
| **Type** | Experience |
| **Status** | Active |
| **Verification Level** | Independently-Verified |
| **Confidence** | 0.95 |
| **Visibility** | Public |
| **Source** | Stark Industries |
| **Last Updated** | 2026-07-16 |
| **Lifecycle State** | Completed |

---

## 1. Role Context
* **Role**: Principal DevOps Architect
* **Duration**: 2025-06 – 2026-06
* **Location**: Remote

## 2. Key Achievements
- Cost Reduction: Saved $1.2M in annual cloud spend.
`
	if err := os.WriteFile(filepath.Join(expDir, "stark-devops.md"), []byte(mockExp), 0644); err != nil {
		t.Fatal(err)
	}

	// Create temporary vault
	vault := t.TempDir()
	path := filepath.Join(vault, "job.md")
	input := `---
job_id: R123
company: Stark Industries
title: Principal DevOps Architect
state: apply
custom_field: preserved
---

# Principal DevOps Architect

Requirements:
- Kubernetes
- Cloud architecture

## The Forge Intelligence
Existing intelligence.
`
	if err := os.WriteFile(path, []byte(input), 0640); err != nil {
		t.Fatal(err)
	}

	generator := &fakeIntelGenerator{}
	orchestrator, err := NewOrchestrator(vault, generator)
	if err != nil {
		t.Fatal(err)
	}
	defer orchestrator.Stop()
	orchestrator.SetApplicationConfig(ApplicationConfig{
		CKBDir:   ckbDir,
		DemoMode: true,
		Contact: rendering.ContactInfo{
			Name:  "Tony Stark",
			Email: "tony@stark.com",
		},
	})

	if err := orchestrator.Start(); err != nil {
		t.Fatal(err)
	}

	// Wait for transition to completed state
	waitFor(t, func() bool {
		updated, err := os.ReadFile(path)
		return err == nil && strings.Contains(string(updated), "state: completed")
	})

	// Verify job note updates
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(updated)
	if !strings.Contains(text, "custom_field: preserved") {
		t.Error("Expected custom_field: preserved to be preserved")
	}
	if !strings.Contains(text, "Existing intelligence.") {
		t.Error("Expected existing intelligence section to be preserved")
	}

	// Verify generated artifacts in vault applications folder
	appDir := filepath.Join(vault, "applications", "stark_industries-principal_devops_architect")
	resumePath := filepath.Join(appDir, "resume.md")
	clPath := filepath.Join(appDir, "cover_letter.md")
	manifestPath := filepath.Join(appDir, "manifest.json")

	if _, err := os.Stat(resumePath); os.IsNotExist(err) {
		t.Fatal("Resume artifact was not generated")
	}
	if _, err := os.Stat(clPath); os.IsNotExist(err) {
		t.Fatal("Cover letter artifact was not generated")
	}
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Packet manifest was not generated: %v", err)
	}
	var manifest packetManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("Packet manifest is invalid: %v", err)
	}
	if manifest.Job.JobID != "R123" || !manifest.DemoMode || len(manifest.Files) != 2 {
		t.Fatalf("Unexpected packet manifest: %+v", manifest)
	}
	for _, file := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(appDir, file.Name))
		if err != nil {
			t.Fatal(err)
		}
		if file.SHA256 != digestBytes(data) {
			t.Fatalf("Manifest digest mismatch for %s", file.Name)
		}
	}

	resumeData, err := os.ReadFile(resumePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(resumeData), "# Tony Stark") {
		t.Errorf("Resume missing name, got:\n%s", string(resumeData))
	}
	if !strings.Contains(string(resumeData), "THE FORGE DEMO OUTPUT") {
		t.Fatalf("demo resume missing warning banner:\n%s", resumeData)
	}

	clData, err := os.ReadFile(clPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(clData), "Dear Hiring Manager at Stark Industries,") {
		t.Errorf("Cover letter missing salutation, got:\n%s", string(clData))
	}
	if !strings.Contains(string(clData), "Saved $1.2M in annual cloud spend.") {
		t.Errorf("Cover letter missing accomplishment, got:\n%s", string(clData))
	}
	if !strings.Contains(string(clData), "THE FORGE DEMO OUTPUT") {
		t.Fatalf("demo cover letter missing warning banner:\n%s", clData)
	}
}

func TestApplyStateRemainsUnchangedWhenApplicationConfigurationIsMissing(t *testing.T) {
	vault := t.TempDir()
	path := filepath.Join(vault, "job.md")
	input := "---\ncompany: Example\ntitle: Engineer\nstate: apply\n---\n\nJob body.\n"
	if err := os.WriteFile(path, []byte(input), 0o640); err != nil {
		t.Fatal(err)
	}

	orchestrator, err := NewOrchestrator(vault, &fakeIntelGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	defer orchestrator.Stop()
	if err := orchestrator.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return orchestrator.pendingCount() == 0 })

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != input {
		t.Fatalf("job note changed after rejected application build:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(vault, "applications")); !os.IsNotExist(err) {
		t.Fatalf("applications directory exists after rejected build: %v", err)
	}
}

func TestProcessApplyFailsClosedOnMissingApplicationConfiguration(t *testing.T) {
	orchestrator, err := NewOrchestrator(t.TempDir(), &fakeIntelGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	defer orchestrator.Stop()

	job := models.JobPost{Company: "Example", Title: "Engineer", State: "apply"}
	if err := orchestrator.processApply("job.md", job); err == nil || !strings.Contains(err.Error(), "THEFORGE_CKB_DIR is required") {
		t.Fatalf("processApply() error = %v, want missing CKB error", err)
	}

	orchestrator.SetApplicationConfig(ApplicationConfig{CKBDir: t.TempDir()})
	if err := orchestrator.processApply("job.md", job); err == nil || !strings.Contains(err.Error(), "THEFORGE_CONTACT_NAME is required") {
		t.Fatalf("processApply() error = %v, want missing name error", err)
	}

	orchestrator.SetApplicationConfig(ApplicationConfig{
		CKBDir: t.TempDir(),
		Contact: rendering.ContactInfo{
			Name: "Candidate Name",
		},
	})
	if err := orchestrator.processApply("job.md", job); err == nil || !strings.Contains(err.Error(), "THEFORGE_CONTACT_EMAIL is required") {
		t.Fatalf("processApply() error = %v, want missing email error", err)
	}
}

func TestBlockingDiagnosticErrorReportsCodesWithoutPrivateMessages(t *testing.T) {
	err := blockingDiagnosticError("resume planning", []model.Diagnostic{
		{Code: "CKB-PRIVATE-CANARY", Severity: model.SeverityFatal, Message: "private evidence body must not appear"},
		{Code: "CKB-SECOND", Severity: model.SeverityError, Message: "another private value"},
	})
	message := err.Error()
	if !strings.Contains(message, "CKB-PRIVATE-CANARY") || !strings.Contains(message, "CKB-SECOND") {
		t.Fatalf("error missing diagnostic codes: %s", message)
	}
	if strings.Contains(message, "private evidence body") || strings.Contains(message, "another private value") {
		t.Fatalf("error leaked diagnostic messages: %s", message)
	}
}
