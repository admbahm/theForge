package models

import (
	"strings"
	"testing"
)

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

	updated, err := UpdateStateAndAppendIntel(input, "intel-ready", "### Role Summary\nGood fit.")
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

	updated, err := UpdateStateAndAppendIntel(input, "intel-ready", "### Role Summary\nNew summary.")
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
