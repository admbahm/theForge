package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/admbahm/theForge/pkg/models"
)

func TestTruncateContextNoopIfWithinLimit(t *testing.T) {
	input := "Short job description."
	output := TruncateContext(context.Background(), nil, input, 100)
	if output != input {
		t.Fatalf("TruncateContext() = %q, want %q", output, input)
	}
}

func TestTruncateContextPrunesExtraneousLinesAndPreservesKeywords(t *testing.T) {
	input := `
# Engineering Lead

Join our awesome team where we have free lunch and snacks daily.

Our office has a slide and a ping pong table.

Must have AWS production experience.

Requirements include 5 years of Go development.

We love coding in our modern office.
`
	output := TruncateContext(context.Background(), nil, input, 250)

	if !strings.Contains(output, "AWS production experience") {
		t.Fatalf("expected output to contain AWS requirement, got: %q", output)
	}
	if !strings.Contains(output, "5 years of Go development") {
		t.Fatalf("expected output to contain Go requirement, got: %q", output)
	}
	if strings.Contains(output, "free lunch") {
		t.Fatalf("expected output to discard free lunch line, got: %q", output)
	}
}

func TestTruncateContextStripsBoilerplate(t *testing.T) {
	input := `
# Senior Go Developer

Must have 5 years of Go production experience.

We are an Equal Opportunity Employer. Employment decisions are made without regard to race, color, religion, national origin, veteran status, or disability status. We encourage diversity and inclusion.
`
	output := TruncateContext(context.Background(), nil, input, 200)

	if !strings.Contains(output, "5 years of Go production experience") {
		t.Fatalf("expected output to contain requirement, got: %q", output)
	}
	if strings.Contains(output, "Equal Opportunity Employer") || strings.Contains(output, "without regard") {
		t.Fatalf("expected output to strip EEO boilerplate, got: %q", output)
	}
}

type mockSummaryClient struct {
	called bool
}

func (m *mockSummaryClient) GenerateIntel(ctx context.Context, job models.JobPost) (string, error) {
	m.called = true
	return "### Key Requirements & Tech Stack\n- Mocked Summary", nil
}

func TestTruncateContextInvokesLocalSummarizer(t *testing.T) {
	input := `
# Senior Software Engineer

Must have 8 years of software development experience.

Requirements include proficiency in Go, GCP, and Kubernetes.

Candidate should be familiar with database performance tuning.

We value self-starting engineers who love collaborating.

Additional requirements include knowledge of system architecture, design patterns, microservices, distributed systems, continuous integration, continuous delivery, automated testing, agile methodologies, and team leadership.
`
	client := &mockSummaryClient{}
	// Make budget 200, input pruned length will be >200, triggering summarization
	output := TruncateContext(context.Background(), client, input, 200)

	if !client.called {
		t.Fatal("expected local summarizer fallback to be called")
	}
	if !strings.Contains(output, "Mocked Summary") {
		t.Fatalf("expected output to contain mocked summary, got: %q", output)
	}
}
