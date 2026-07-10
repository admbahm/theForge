package models

import (
	"strings"
	"testing"
)

func TestComputeConfidence_FullWorkdayPosting(t *testing.T) {
	job := JobPost{
		JobID:     "WD-1001",
		Company:   "Stark Industries",
		Title:     "Senior Arc Reactor Engineer",
		Location:  "Los Angeles, CA",
		SalaryMin: 180000,
		SalaryMax: 220000,
		TechStack: []string{"Vibranium", "ArcTech", "Go"},
		Content:   "This is a complete job description outlining full responsibilities and qualifications. Stark Industries is looking for a leader to design arc reactor systems.",
	}

	conf := ComputeConfidence(job)

	if conf.Score != 1.0 {
		t.Errorf("Expected score 1.0, got %.2f", conf.Score)
	}
	if conf.Level != "High" {
		t.Errorf("Expected level High, got %q", conf.Level)
	}
	hasExplanation(t, conf.Explanation, "Full job description available")
	hasExplanation(t, conf.Explanation, "Technologies explicitly listed")
	hasExplanation(t, conf.Explanation, "Compensation details available")
	hasExplanation(t, conf.Explanation, "Job ID available")
	hasExplanation(t, conf.Explanation, "Job title available")
	hasExplanation(t, conf.Explanation, "Location details available")
}

func TestComputeConfidence_PartialPosting(t *testing.T) {
	// Missing Location and Job ID
	job := JobPost{
		Company:   "Acme Corp",
		Title:     "Software Engineer",
		SalaryMin: 100000,
		TechStack: []string{"Python"},
		Content:   "We need a backend developer to build standard web APIs.",
	}

	conf := ComputeConfidence(job)

	// Max score is 1.0. Deducts: JobID missing (-0.05), Location missing (-0.05).
	// Score should be 0.90
	if conf.Score != 0.90 {
		t.Errorf("Expected score 0.90, got %.2f", conf.Score)
	}
	if conf.Level != "High" {
		t.Errorf("Expected level High, got %q", conf.Level)
	}
	hasExplanation(t, conf.Explanation, "Job ID missing")
	hasExplanation(t, conf.Explanation, "Location details missing")
	hasExplanation(t, conf.Explanation, "Compensation details available")
}

func TestComputeConfidence_MissingDescription(t *testing.T) {
	job := JobPost{
		JobID:     "JR123",
		Company:   "Stark Industries",
		Title:     "Engineer",
		Location:  "Remote",
		SalaryMin: 120000,
		TechStack: []string{"Go"},
		Content:   "", // Missing
	}

	conf := ComputeConfidence(job)

	// Score capped at 0.35, level Low
	if conf.Score > 0.35 {
		t.Errorf("Expected score capped at <= 0.35, got %.2f", conf.Score)
	}
	if conf.Level != "Low" {
		t.Errorf("Expected level Low, got %q", conf.Level)
	}
	hasExplanation(t, conf.Explanation, "Job description missing")
}

func TestComputeConfidence_ConflictingMetadata(t *testing.T) {
	// Min salary > Max salary
	job := JobPost{
		JobID:     "JR123",
		Company:   "Stark Industries",
		Title:     "Engineer",
		Location:  "Remote",
		SalaryMin: 150000,
		SalaryMax: 120000, // Conflict!
		TechStack: []string{"Go"},
		Content:   "Build great things.",
	}

	conf := ComputeConfidence(job)

	// Score should exclude compensation (-0.15), so 1.0 - 0.15 = 0.85
	if conf.Score != 0.85 {
		t.Errorf("Expected score 0.85, got %.2f", conf.Score)
	}
	hasExplanation(t, conf.Explanation, "Conflicting salary metadata (minimum exceeds maximum)")
}

func TestComputeConfidence_EmptyTechnologies(t *testing.T) {
	job := JobPost{
		JobID:     "WD-1001",
		Company:   "Stark Industries",
		Title:     "Engineer",
		Location:  "Los Angeles, CA",
		SalaryMin: 180000,
		TechStack: []string{}, // Empty
		Content:   "This is a complete job description.",
	}

	conf := ComputeConfidence(job)

	// Score should deduct tech stack (-0.20), so 1.0 - 0.20 = 0.80
	if conf.Score != 0.80 {
		t.Errorf("Expected score 0.80, got %.2f", conf.Score)
	}
	hasExplanation(t, conf.Explanation, "Technologies not explicitly listed")
}

func TestComputeConfidence_EmptySalary(t *testing.T) {
	job := JobPost{
		JobID:     "WD-1001",
		Company:   "Stark Industries",
		Title:     "Engineer",
		Location:  "Los Angeles, CA",
		SalaryMin: 0,
		SalaryMax: 0, // Empty
		TechStack: []string{"Go"},
		Content:   "This is a complete job description.",
	}

	conf := ComputeConfidence(job)

	// Score should deduct salary (-0.15), so 1.0 - 0.15 = 0.85
	if conf.Score != 0.85 {
		t.Errorf("Expected score 0.85, got %.2f", conf.Score)
	}
	hasExplanation(t, conf.Explanation, "Compensation details missing")
}

func TestComputeConfidence_InvalidHTML(t *testing.T) {
	job := JobPost{
		JobID:     "WD-1001",
		Company:   "Stark Industries",
		Title:     "Engineer",
		Location:  "Los Angeles, CA",
		SalaryMin: 100000,
		TechStack: []string{"Go"},
		Content:   "This is a <b>job description with <div>unclosed tags.", // Invalid HTML
	}

	conf := ComputeConfidence(job)

	// Score: Description is invalid HTML (+0.35 instead of +0.50), so 1.0 - 0.15 = 0.85
	if conf.Score != 0.85 {
		t.Errorf("Expected score 0.85, got %.2f", conf.Score)
	}
	hasExplanation(t, conf.Explanation, "Job description contains invalid HTML")
}

func TestComputeConfidence_TruncatedPosting(t *testing.T) {
	job := JobPost{
		JobID:     "WD-1001",
		Company:   "Stark Industries",
		Title:     "Engineer",
		Location:  "Los Angeles, CA",
		SalaryMin: 100000,
		TechStack: []string{"Go"},
		Content:   "This is a job description that ends abruptly ... See More", // Truncated
	}

	conf1 := ComputeConfidence(job)
	if conf1.Score != 0.70 { // Description gets +0.20 instead of +0.50, so 1.0 - 0.30 = 0.70
		t.Errorf("Expected score 0.70, got %.2f", conf1.Score)
	}
	hasExplanation(t, conf1.Explanation, "Job description is truncated")

	job2 := JobPost{
		JobID:     "WD-1001",
		Company:   "Stark Industries",
		Title:     "Engineer",
		Location:  "Los Angeles, CA",
		SalaryMin: 100000,
		TechStack: []string{"Go"},
		Content:   "This description ends with three dots...", // Truncated suffix
	}

	conf2 := ComputeConfidence(job2)
	if conf2.Score != 0.70 {
		t.Errorf("Expected score 0.70 for suffix dots, got %.2f", conf2.Score)
	}
}

func TestRenderMarkdownBadge(t *testing.T) {
	conf := &AnalysisConfidence{
		Score:       0.87,
		Level:       "High",
		Explanation: []string{"Exp 1", "Exp 2"},
	}

	markdown := conf.RenderMarkdown()

	if !strings.Contains(markdown, "🟢 High Confidence") {
		t.Errorf("Markdown missing badge: %s", markdown)
	}
	if !strings.Contains(markdown, "Score: 0.87") {
		t.Errorf("Markdown missing score: %s", markdown)
	}
	if !strings.Contains(markdown, "<details>") {
		t.Errorf("Markdown missing details: %s", markdown)
	}
	if !strings.Contains(markdown, "- Exp 1") || !strings.Contains(markdown, "- Exp 2") {
		t.Errorf("Markdown missing explanations: %s", markdown)
	}
}

func hasExplanation(t *testing.T, explanations []string, target string) {
	t.Helper()
	for _, exp := range explanations {
		if exp == target {
			return
		}
	}
	t.Errorf("Expected explanation %q not found in: %v", target, explanations)
}
