package llm

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/admbahm/theForge/pkg/models"
)

// TruncateContext processes and trims a job posting's content if it exceeds maxChars.
// It prioritizes structural paragraphs containing requirements/responsibilities,
// strips legalese boilerplate, and falls back to a local LLM summarization pass
// if the content is still over the limit.
func TruncateContext(ctx context.Context, localClient Client, content string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}

	// 1. Remove boilerplate legalese paragraphs and run paragraph-level structural pruning
	pruned := structuralPrune(content)
	if len(pruned) <= maxChars {
		return pruned
	}

	// 2. If still over limits and localClient is available, use local Ollama to compress
	if localClient != nil {
		// Use a short 10-second timeout for the pre-summarization fallback to prevent hanging
		sumCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		localCtx := context.WithValue(sumCtx, "tier", "local")
		tempJob := models.JobPost{
			Content: pruned,
		}

		log.Printf("[Truncator] Job description size (%d chars) exceeds limit (%d chars). Triggering local Ollama pre-summarization pass...", len(pruned), maxChars)
		summary, err := localClient.GenerateIntel(localCtx, tempJob)
		if err == nil && len(summary) > 10 {
			result := "[Note: Original job description summarized locally to fit context bounds. Core requirements/responsibilities follow:]\n\n" + summary
			if len(result) <= maxChars {
				return result
			}
			// If summary itself is somehow too long, fall through to raw slice
			return result[:maxChars]
		}
		log.Printf("[Warning] Local pre-summarization failed: %v. Falling back to heuristic character truncation.", err)
	}

	// 3. Fallback: simple character slice
	return pruned[:maxChars]
}

func structuralPrune(content string) string {
	paragraphs := strings.Split(content, "\n\n")
	var keptParagraphs []string

	keywords := []string{
		"require", "qualification", "responsibility", "must have", "experience", "skill",
		"degree", "certificat", "familiarity", "technolog", "proficien", "expert",
	}

	for _, para := range paragraphs {
		trimmed := strings.TrimSpace(para)
		if trimmed == "" {
			continue
		}

		// Always keep section headers (e.g. Markdown headers)
		if strings.HasPrefix(trimmed, "#") {
			keptParagraphs = append(keptParagraphs, para)
			continue
		}

		if isBoilerplate(trimmed) {
			continue
		}

		matched := false
		lowerPara := strings.ToLower(trimmed)
		for _, kw := range keywords {
			if strings.Contains(lowerPara, kw) {
				matched = true
				break
			}
		}

		if matched {
			keptParagraphs = append(keptParagraphs, para)
		}
	}

	pruned := strings.Join(keptParagraphs, "\n\n")
	if len(pruned) == 0 {
		// Fallback to original content if structural pruning was too aggressive and returned nothing
		return content
	}
	return pruned
}

func isBoilerplate(text string) bool {
	lower := strings.ToLower(text)
	signatures := []string{
		"equal opportunity employer",
		"affirmative action",
		"without regard to race",
		"veteran status",
		"disability status",
		"reasonable accommodation",
		"gender identity",
		"sexual orientation",
		"national origin",
		"protected status",
		"employment decision",
		"diversity and inclusion",
	}

	matches := 0
	for _, sig := range signatures {
		if strings.Contains(lower, sig) {
			matches++
			if matches >= 2 {
				return true
			}
		}
	}

	strongSignatures := []string{
		"we are an equal opportunity employer",
		"employment decisions are made without regard",
	}
	for _, sig := range strongSignatures {
		if strings.Contains(lower, sig) {
			return true
		}
	}

	return false
}
