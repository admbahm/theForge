package models

import (
	"fmt"
	"strings"
)

// AnalysisConfidence represents the confidence metadata for a job analysis.
type AnalysisConfidence struct {
	Score       float64  `yaml:"score"`
	Level       string   `yaml:"level"`
	Explanation []string `yaml:"explanation"`
}

// ComputeConfidence calculates a deterministic confidence score and level based on evidence completeness.
func ComputeConfidence(job JobPost) *AnalysisConfidence {
	var explanations []string
	var score float64

	// 1. Job Description Content (Max 0.50)
	if job.Content == "" {
		explanations = append(explanations, "Job description missing")
	} else if isTruncated(job.Content) {
		score += 0.20
		explanations = append(explanations, "Job description is truncated")
	} else if hasInvalidHTML(job.Content) {
		score += 0.35
		explanations = append(explanations, "Job description contains invalid HTML")
	} else {
		score += 0.50
		explanations = append(explanations, "Full job description available")
	}

	// 2. Technologies / Tech Stack (Max 0.20)
	if len(job.TechStack) > 0 {
		score += 0.20
		explanations = append(explanations, "Technologies explicitly listed")
	} else {
		explanations = append(explanations, "Technologies not explicitly listed")
	}

	// 3. Compensation / Salary (Max 0.15)
	if job.SalaryMin > 0 && job.SalaryMax > 0 && job.SalaryMin > job.SalaryMax {
		explanations = append(explanations, "Conflicting salary metadata (minimum exceeds maximum)")
	} else if job.SalaryMin > 0 || job.SalaryMax > 0 {
		score += 0.15
		explanations = append(explanations, "Compensation details available")
	} else {
		explanations = append(explanations, "Compensation details missing")
	}

	// 4. Metadata Completeness (Max 0.15)
	if job.JobID != "" {
		score += 0.05
		explanations = append(explanations, "Job ID available")
	} else {
		explanations = append(explanations, "Job ID missing")
	}

	if job.Title != "" {
		score += 0.05
		explanations = append(explanations, "Job title available")
	} else {
		explanations = append(explanations, "Job title missing")
	}

	if job.Location != "" {
		score += 0.05
		explanations = append(explanations, "Location details available")
	} else {
		explanations = append(explanations, "Location details missing")
	}

	// Round score to two decimal places
	score = float64(int(score*100+0.5)) / 100.0

	// Determine confidence level
	var level string
	if job.Content == "" {
		// Cap score at 0.35 and force level to Low if job description is missing
		if score > 0.35 {
			score = 0.35
		}
		level = "Low"
	} else {
		if score >= 0.80 {
			level = "High"
		} else if score >= 0.50 {
			level = "Medium"
		} else {
			level = "Low"
		}
	}

	return &AnalysisConfidence{
		Score:       score,
		Level:       level,
		Explanation: explanations,
	}
}

func isTruncated(content string) bool {
	contentLower := strings.ToLower(content)
	indicators := []string{
		"... see more",
		"read more",
		"[truncated]",
		"see full description",
	}
	for _, ind := range indicators {
		if strings.Contains(contentLower, ind) {
			return true
		}
	}
	trimmed := strings.TrimSpace(content)
	if strings.HasSuffix(trimmed, "...") {
		return true
	}
	return false
}

func hasInvalidHTML(content string) bool {
	var tags []string
	i := 0
	n := len(content)

	for i < n {
		idx := strings.IndexByte(content[i:], '<')
		if idx == -1 {
			break
		}
		start := i + idx
		if start+1 >= n {
			return true
		}
		next := content[start+1]
		isTag := (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || next == '/' || next == '!' || next == '?'
		if !isTag {
			i = start + 1
			continue
		}

		endIdx := strings.IndexByte(content[start:], '>')
		if endIdx == -1 {
			return true
		}
		end := start + endIdx
		tagContent := content[start+1 : end]
		i = end + 1

		tagContent = strings.TrimSpace(tagContent)
		if tagContent == "" {
			return true
		}

		if strings.HasPrefix(tagContent, "/") {
			tagName := strings.ToLower(strings.TrimSpace(tagContent[1:]))
			if len(tags) == 0 {
				return true
			}
			lastTag := tags[len(tags)-1]
			if lastTag != tagName {
				return true
			}
			tags = tags[:len(tags)-1]
		} else if strings.HasPrefix(tagContent, "!") || strings.HasPrefix(tagContent, "?") {
			continue
		} else {
			if strings.HasSuffix(tagContent, "/") {
				continue
			}
			parts := strings.Fields(tagContent)
			if len(parts) == 0 {
				continue
			}
			tagName := strings.ToLower(parts[0])
			voidElements := map[string]bool{
				"br": true, "hr": true, "img": true, "input": true,
				"link": true, "meta": true, "col": true, "area": true,
				"base": true, "embed": true, "param": true, "source": true,
				"track": true, "wbr": true,
			}
			if !voidElements[tagName] {
				tags = append(tags, tagName)
			}
		}
	}

	return len(tags) > 0
}

// RenderMarkdown generates the confidence badge HTML/Markdown block.
func (c *AnalysisConfidence) RenderMarkdown() string {
	var badge string
	switch c.Level {
	case "High":
		badge = "🟢 High Confidence"
	case "Medium":
		badge = "🟡 Medium Confidence"
	default:
		badge = "🔴 Low Confidence"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<span title=\"Score: %.2f. Expand below to see details.\">%s (Score: %.2f)</span>\n\n", c.Score, badge, c.Score))
	sb.WriteString("<details>\n<summary>Confidence Reasoning</summary>\n\n")
	for _, exp := range c.Explanation {
		sb.WriteString(fmt.Sprintf("- %s\n", exp))
	}
	sb.WriteString("</details>\n\n")
	return sb.String()
}
