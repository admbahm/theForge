package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/admbahm/theForge/pkg/models"
)

type fakeIntelGenerator struct {
	intel string
	calls int
}

func (f *fakeIntelGenerator) GenerateIntel(_ context.Context, _ models.JobPost) (string, error) {
	f.calls++
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
	if generator.calls != 1 {
		t.Fatalf("generator calls = %d, want 1", generator.calls)
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
	if generator.calls != 0 {
		t.Fatalf("generator calls = %d, want 0", generator.calls)
	}
}
