package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
