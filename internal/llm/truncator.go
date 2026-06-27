package llm

import (
	"strings"
)

// TruncateContext processes and trims a job posting's content if it exceeds maxChars.
// It prioritizes bullet points or lines containing keywords like "require", "qualification",
// "responsibility", "must have", "experience", "skill" to keep the core signals intact.
func TruncateContext(content string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}

	// Simple heuristic-based pruning to retain signal
	lines := strings.Split(content, "\n")
	var keptLines []string
	var otherLines []string

	keywords := []string{
		"require", "qualification", "responsibility", "must have", "experience", "skill",
		"degree", "certificat", "familiarity", "technolog", "proficien", "expert",
	}

	for _, line := range lines {
		trimmed := strings.ToLower(strings.TrimSpace(line))
		if trimmed == "" {
			continue
		}

		// Always keep section headers (e.g. Markdown headers)
		if strings.HasPrefix(line, "#") {
			keptLines = append(keptLines, line)
			continue
		}

		matched := false
		for _, kw := range keywords {
			if strings.Contains(trimmed, kw) {
				matched = true
				break
			}
		}

		if matched {
			keptLines = append(keptLines, line)
		} else {
			otherLines = append(otherLines, line)
		}
	}

	// Reconstruct content starting with critical keyword lines
	var builder strings.Builder
	builder.WriteString("[Note: Original job description truncated to fit context bounds. Core requirements/responsibilities extracted below:]\n\n")

	for _, line := range keptLines {
		if builder.Len()+len(line)+1 > maxChars {
			break
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	// If we still have room, backfill with other lines until we hit the budget
	if builder.Len() < maxChars {
		builder.WriteString("\n[Additional Context (Truncated)]:\n")
		for _, line := range otherLines {
			if builder.Len()+len(line)+1 > maxChars {
				break
			}
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}

	result := builder.String()
	if len(result) > maxChars {
		result = result[:maxChars]
	}
	return result
}
