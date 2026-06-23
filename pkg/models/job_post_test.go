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
