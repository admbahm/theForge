package llm

import (
	"strings"
	"testing"
)

func TestTruncateContextNoopIfWithinLimit(t *testing.T) {
	input := "Short job description."
	output := TruncateContext(input, 100)
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
	output := TruncateContext(input, 250)

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
