package models

import (
	"gopkg.in/yaml.v3"
	"fmt"
	"strings"
)

// JobPost represents the core data structure for a job application tracking state.
// It maps to YAML frontmatter in Obsidian Markdown files.
type JobPost struct {
	JobID           string   `yaml:"job_id"`
	Company         string   `yaml:"company"`
	Title           string   `yaml:"title"`
	Location        string   `yaml:"location"`
	PostedAt        string   `yaml:"posted_at"`
	SalaryMin       int      `yaml:"salary_min"`
	SalaryMax       int      `yaml:"salary_max"`
	RoleType        string   `yaml:"role_type"`
	TechStack       []string `yaml:"tech_stack"`
	RegulatoryGates []string `yaml:"regulatory_gates"`
	ScrapedAt       string   `yaml:"scraped_at"`
	Favorite        bool     `yaml:"favorite"`
	State           string   `yaml:"state"` // e.g., "new", "favorite", "intel-ready", "apply", "completed"
	Content         string   `yaml:"-"`     // Markdown content body
}

// MarshalMarkdown encodes the JobPost to a Markdown file with YAML frontmatter.
func (j *JobPost) MarshalMarkdown() ([]byte, error) {
	frontmatter, err := yaml.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(frontmatter)
	sb.WriteString("---\n\n")
	sb.WriteString(j.Content)

	return []byte(sb.String()), nil
}

// UnmarshalMarkdown decodes a JobPost from a Markdown file with YAML frontmatter.
func UnmarshalMarkdown(data []byte, j *JobPost) error {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("missing frontmatter delimiter")
	}

	parts := strings.SplitN(content[4:], "---\n", 2)
	if len(parts) < 2 {
		// Try without the newline in case it's the end of file or different newline style
		parts = strings.SplitN(content[4:], "---", 2)
		if len(parts) < 2 {
			return fmt.Errorf("invalid frontmatter format")
		}
	}

	err := yaml.Unmarshal([]byte(parts[0]), j)
	if err != nil {
		return fmt.Errorf("failed to unmarshal frontmatter: %w", err)
	}

	j.Content = strings.TrimSpace(parts[1])
	return nil
}
