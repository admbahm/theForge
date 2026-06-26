package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/admbahm/theForge/pkg/models"
)

type fakeIntelGenerator struct {
	intel   string
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
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
	for _, expected := range []string{"state: intel-ready", "custom_field: preserved", "## The Forge Intelligence", "Strong match."} {
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

	if err := orchestrator.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return orchestrator.pendingCount() == 0 })
	if generator.calls.Load() != 0 {
		t.Fatalf("generator calls = %d, want 0", generator.calls.Load())
	}
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

func (o *Orchestrator) pendingCount() int {
	o.pendingMu.Lock()
	defer o.pendingMu.Unlock()
	return len(o.pending)
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
