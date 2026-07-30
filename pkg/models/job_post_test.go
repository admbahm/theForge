package models

import (
	"strings"
	"testing"
)

func TestUnmarshalMarkdownAcceptsOpenHuntSalaryValues(t *testing.T) {
	tests := []struct {
		name    string
		minimum string
		maximum string
		wantMin SalaryAmount
		wantMax SalaryAmount
	}{
		{name: "numeric amounts", minimum: "180000", maximum: "220000", wantMin: 180000, wantMax: 220000},
		{name: "unspecified", minimum: "unspecified", maximum: "unspecified"},
		{name: "other missing markers", minimum: "unknown", maximum: "n/a"},
		{name: "empty quoted values", minimum: `""`, maximum: `""`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := []byte("---\ncompany: Apple\ntitle: Engineer\nstate: apply\nsalary_min: " + test.minimum + "\nsalary_max: " + test.maximum + "\n---\n\nBody\n")
			var job JobPost
			if err := UnmarshalMarkdown(input, &job); err != nil {
				t.Fatalf("UnmarshalMarkdown() error = %v", err)
			}
			if job.SalaryMin != test.wantMin || job.SalaryMax != test.wantMax || job.State != "apply" {
				t.Fatalf("unexpected parsed job: %+v", job)
			}
		})
	}
}

func TestUnmarshalMarkdownRejectsInvalidSalary(t *testing.T) {
	for _, value := range []string{"competitive", "100000.50", "true"} {
		t.Run(value, func(t *testing.T) {
			input := []byte("---\ncompany: Example\nsalary_min: " + value + "\n---\n")
			var job JobPost
			err := UnmarshalMarkdown(input, &job)
			if err == nil || !strings.Contains(err.Error(), "salary must be an integer or missing-value marker") {
				t.Fatalf("UnmarshalMarkdown() error = %v", err)
			}
		})
	}
}

func TestUpdateStateAndAppendIntelPreservesUnknownFrontmatter(t *testing.T) {
	input := []byte(`---
job_id: R123
company: Example
title: Engineer
favorite: true
custom_field: keep-me
---

# Engineer

Original body.
`)

	updated, err := UpdateStateAndAppendIntel(input, "intel-ready", "### Role Summary\nGood fit.", nil)
	if err != nil {
		t.Fatal(err)
	}

	text := string(updated)
	for _, expected := range []string{
		"custom_field: keep-me",
		"state: intel-ready",
		"Original body.",
		"## The Forge Intelligence",
		"### Role Summary",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("updated Markdown missing %q:\n%s", expected, text)
		}
	}
}

func TestUnmarshalMarkdownSupportsCRLF(t *testing.T) {
	input := []byte("---\r\ncompany: Example\r\ntitle: Engineer\r\n---\r\n\r\nBody\r\n")

	var job JobPost
	if err := UnmarshalMarkdown(input, &job); err != nil {
		t.Fatal(err)
	}
	if job.Company != "Example" || job.Content != "Body" {
		t.Fatalf("job = %#v", job)
	}
}

func TestUpdateStateAndAppendIntelOverwritesExisting(t *testing.T) {
	input := []byte(`---
job_id: R123
company: Example
title: Engineer
state: processed
---

Original body.

## The Forge Intelligence

### Company Profile
Old summary.
`)

	updated, err := UpdateStateAndAppendIntel(input, "intel-ready", "### Role Summary\nNew summary.", nil)
	if err != nil {
		t.Fatal(err)
	}

	text := string(updated)
	if strings.Count(text, "## The Forge Intelligence") != 1 {
		t.Fatalf("expected exactly one '## The Forge Intelligence' header, got: %d\nFull content:\n%s", strings.Count(text, "## The Forge Intelligence"), text)
	}
	if strings.Contains(text, "Old summary.") {
		t.Fatalf("expected old summary to be overwritten, but found it in:\n%s", text)
	}
	if !strings.Contains(text, "New summary.") {
		t.Fatalf("expected new summary to be present in:\n%s", text)
	}
}

func TestUpdateStateAndAppendIntelIncludesConfidence(t *testing.T) {
	input := []byte(`---
job_id: R123
company: Stark Industries
title: Arc Engineer
state: favorite
---

Body text.
`)

	conf := &AnalysisConfidence{
		Score: 0.85,
		Level: "High",
		Explanation: []string{
			"Full job description available",
			"Technologies explicitly listed",
		},
	}

	updated, err := UpdateStateAndAppendIntel(input, "intel-ready", "### Role Summary\nEnriched.", conf)
	if err != nil {
		t.Fatal(err)
	}

	text := string(updated)
	expectedStrings := []string{
		"analysis_confidence:",
		"score: 0.85",
		"level: High",
		"Full job description available",
		"Technologies explicitly listed",
		"🟢 High Confidence (Score: 0.85)",
		"Confidence Reasoning",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(text, expected) {
			t.Fatalf("updated note missing expected content %q:\n%s", expected, text)
		}
	}
}

func TestUpdateStateOnly(t *testing.T) {
	input := []byte(`---
job_id: R123
company: Stark Industries
title: Arc Engineer
state: apply
custom_field: keep-me
---

# Title

Body text.

## The Forge Intelligence
Old intelligence.
`)

	updated, err := UpdateStateOnly(input, "completed")
	if err != nil {
		t.Fatal(err)
	}

	text := string(updated)
	expectedStrings := []string{
		"state: completed",
		"custom_field: keep-me",
		"Body text.",
		"## The Forge Intelligence",
		"Old intelligence.",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(text, expected) {
			t.Fatalf("updated note missing expected content %q:\n%s", expected, text)
		}
	}
}
