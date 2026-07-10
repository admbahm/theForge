package models

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// JobPost represents the core data structure for a job opportunity tracking state.
// It maps to YAML frontmatter in Obsidian Markdown files.
type JobPost struct {
	JobID              string              `yaml:"job_id"`
	Company            string              `yaml:"company"`
	Title              string              `yaml:"title"`
	Location           string              `yaml:"location"`
	PostedAt           string              `yaml:"posted_at"`
	SalaryMin          int                 `yaml:"salary_min"`
	SalaryMax          int                 `yaml:"salary_max"`
	RoleType           string              `yaml:"role_type"`
	TechStack          []string            `yaml:"tech_stack"`
	RegulatoryGates    []string            `yaml:"regulatory_gates"`
	ScrapedAt          string              `yaml:"scraped_at"`
	Favorite           bool                `yaml:"favorite"`
	State              string              `yaml:"state"` // e.g., "new", "favorite", "intel-ready", "apply", "completed"
	AnalysisConfidence *AnalysisConfidence `yaml:"analysis_confidence,omitempty"`
	Content            string              `yaml:"-"` // Markdown content body
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
	frontmatter, body, err := splitMarkdown(data)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(frontmatter, j); err != nil {
		return fmt.Errorf("failed to unmarshal frontmatter: %w", err)
	}

	j.Content = strings.TrimSpace(string(body))
	return nil
}

// UpdateStateAndAppendIntel preserves existing frontmatter fields while adding
// generated intelligence and updating the job state.
func UpdateStateAndAppendIntel(data []byte, state, intel string, confidence *AnalysisConfidence) ([]byte, error) {
	frontmatter, body, err := splitMarkdown(data)
	if err != nil {
		return nil, err
	}

	var document yaml.Node
	if err := yaml.Unmarshal(frontmatter, &document); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontmatter: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter must be a YAML mapping")
	}

	mapping := document.Content[0]
	stateUpdated := false
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == "state" {
			mapping.Content[index+1].Kind = yaml.ScalarNode
			mapping.Content[index+1].Tag = "!!str"
			mapping.Content[index+1].Value = state
			stateUpdated = true
			break
		}
	}
	if !stateUpdated {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "state"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: state},
		)
	}

	if confidence != nil {
		var confNode yaml.Node
		confData, err := yaml.Marshal(confidence)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal confidence: %w", err)
		}
		if err := yaml.Unmarshal(confData, &confNode); err != nil {
			return nil, fmt.Errorf("failed to unmarshal confidence node: %w", err)
		}
		if len(confNode.Content) > 0 {
			actualConfNode := confNode.Content[0]
			confUpdated := false
			for index := 0; index+1 < len(mapping.Content); index += 2 {
				if mapping.Content[index].Value == "analysis_confidence" {
					mapping.Content[index+1] = actualConfNode
					confUpdated = true
					break
				}
			}
			if !confUpdated {
				mapping.Content = append(mapping.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "analysis_confidence"},
					actualConfNode,
				)
			}
		}
	}

	updatedFrontmatter, err := yaml.Marshal(&document)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Strip existing intelligence section if present
	intelMarker := []byte("\n## The Forge Intelligence\n")
	if markerIdx := bytes.Index(body, intelMarker); markerIdx >= 0 {
		body = body[:markerIdx]
	} else {
		intelMarkerAlt := []byte("\n\n## The Forge Intelligence\n")
		if markerIdxAlt := bytes.Index(body, intelMarkerAlt); markerIdxAlt >= 0 {
			body = body[:markerIdxAlt]
		}
	}

	var result strings.Builder
	result.WriteString("---\n")
	result.Write(updatedFrontmatter)
	result.WriteString("---\n")
	result.Write(bytes.TrimRight(body, "\r\n"))
	result.WriteString("\n\n## The Forge Intelligence\n\n")
	if confidence != nil {
		result.WriteString(confidence.RenderMarkdown())
	}
	result.WriteString(strings.TrimSpace(intel))
	result.WriteByte('\n')
	return []byte(result.String()), nil
}

func splitMarkdown(data []byte) ([]byte, []byte, error) {
	normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(normalized, []byte("---\n")) {
		return nil, nil, fmt.Errorf("missing frontmatter delimiter")
	}

	delimiter := []byte("\n---")
	end := bytes.Index(normalized[4:], delimiter)
	if end < 0 {
		return nil, nil, fmt.Errorf("invalid frontmatter format")
	}
	end += 4
	bodyStart := end + len(delimiter)
	if bodyStart < len(normalized) && normalized[bodyStart] == '\n' {
		bodyStart++
	}

	return normalized[4:end], normalized[bodyStart:], nil
}
