package rendering

import (
	"fmt"
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
)

// RenderBiography processes the plan to compile the biography narrative.
func RenderBiography(req RenderRequest, policy claims.Policy) ([]ArtifactSection, []string) {
	var sections []ArtifactSection
	var warnings []string

	selected := req.Plan.SelectedClaims

	// 1. Resolve pronouns
	subj := "They"

	candName := req.Options.Contact.Name
	if candName == "" {
		candName = "The candidate"
	}

	pStyle := req.Options.PronounStyle
	if pStyle == "" {
		pStyle = "neutral"
	}

	usePronouns := true
	if pStyle == "he/him" {
		subj = "He"
	} else if pStyle == "she/her" {
		subj = "She"
	} else if pStyle == "they/them" {
		subj = "They"
	} else {
		// Neutral pronoun style (no gendered or plural pronouns)
		usePronouns = false
	}

	// Helper to prepend subject or name
	renderSentence := func(claimStatement string) string {
		cleanStmt := strings.TrimSpace(claimStatement)
		cleanStmt = strings.TrimSuffix(cleanStmt, ".")
		words := strings.Fields(cleanStmt)
		if len(words) == 0 {
			return ""
		}

		// Adjust starting verb
		firstWord := words[0]
		cleanFirst := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				return r
			}
			return -1
		}, firstWord)

		cleanLower := strings.ToLower(cleanFirst)
		_, isPresent := verbToPast[cleanLower]
		_, isPast := verbToPresent[cleanLower]
		if isPresent || isPast {
			prefix := subj
			if !usePronouns {
				prefix = candName
			}
			// Lowercase the verb since it is now preceded by a subject
			words[0] = strings.ToLower(firstWord)
			return prefix + " " + strings.Join(words, " ") + "."
		}

		// Standard sentence: make sure first word is capitalized
		return strings.Title(firstWord) + strings.TrimPrefix(cleanStmt, firstWord) + "."
	}

	var bioParagraphs []string
	var bioParagraphClaimIDs [][]claims.ClaimID
	var bioParagraphSourceIDs [][]string
	var bioParagraphEvidenceIDs [][]string

	// Determine subset based on biography length (short, medium, long)
	length := req.Options.BiographyLength
	if length == "" {
		length = "medium"
	}

	// Gather elements
	type bioSentence struct {
		text      string
		claimID   claims.ClaimID
		sourceIDs []string
		evidence  []string
	}

	var summaries []bioSentence
	var roles []bioSentence
	var achievements []bioSentence
	var edus []bioSentence

	buildSentence := func(pc planningClaim) bioSentence {
		return bioSentence{
			text:      renderSentence(pc.Statement),
			claimID:   pc.ID,
			sourceIDs: pc.SourceObjectIDs,
			evidence:  pc.EvidenceObjectIDs,
		}
	}

	for _, pc := range selected {
		k := pc.Claim.Kind
		if pc.Section == "Summary" || k == claims.KindCareerObjective {
			summaries = append(summaries, buildSentence(planningClaimFromClaim(pc.Claim)))
		} else if pc.Section == "Experience" || k == claims.KindRole {
			roles = append(roles, buildSentence(planningClaimFromClaim(pc.Claim)))
		} else if pc.Section == "Achievements" || k == claims.KindAccomplishment || k == claims.KindMeasurableResult {
			achievements = append(achievements, buildSentence(planningClaimFromClaim(pc.Claim)))
		} else if pc.Section == "Education" || k == claims.KindEducation {
			edus = append(edus, buildSentence(planningClaimFromClaim(pc.Claim)))
		}
	}

	appendParagraph := func(sentences []bioSentence) {
		var texts []string
		var claimIDs []claims.ClaimID
		var sourceIDs []string
		var evidenceIDs []string
		for _, sentence := range sentences {
			if strings.TrimSpace(sentence.text) == "" {
				continue
			}
			texts = append(texts, sentence.text)
			claimIDs = append(claimIDs, sentence.claimID)
			sourceIDs = append(sourceIDs, sentence.sourceIDs...)
			evidenceIDs = append(evidenceIDs, sentence.evidence...)
		}
		if len(texts) == 0 {
			return
		}
		bioParagraphs = append(bioParagraphs, strings.Join(texts, " "))
		bioParagraphClaimIDs = append(bioParagraphClaimIDs, claimIDs)
		bioParagraphSourceIDs = append(bioParagraphSourceIDs, sourceIDs)
		bioParagraphEvidenceIDs = append(bioParagraphEvidenceIDs, evidenceIDs)
	}

	switch length {
	case "short": // 50-75 words
		// 1 summary sentence, 1 role sentence
		var sentenceList []bioSentence
		if len(summaries) > 0 {
			sentenceList = append(sentenceList, summaries[0])
		}
		if len(roles) > 0 {
			sentenceList = append(sentenceList, roles[0])
		}
		appendParagraph(sentenceList)

	case "long": // 200-300 words
		// Full summaries, multiple roles, multiple achievements, education
		var firstPara []bioSentence
		firstPara = append(firstPara, summaries...)
		firstPara = append(firstPara, roles...)
		appendParagraph(firstPara)

		var secondPara []bioSentence
		secondPara = append(secondPara, achievements...)
		secondPara = append(secondPara, edus...)
		if len(secondPara) > 0 {
			appendParagraph(secondPara)
		}

	default: // "medium" - 100-150 words
		// summaries, up to 2 roles, 2 achievements
		var sentenceList []bioSentence
		sentenceList = append(sentenceList, summaries...)
		for i := 0; i < len(roles) && i < 2; i++ {
			sentenceList = append(sentenceList, roles[i])
		}
		for i := 0; i < len(achievements) && i < 2; i++ {
			sentenceList = append(sentenceList, achievements[i])
		}
		appendParagraph(sentenceList)
	}

	var entries []ArtifactEntry
	for idx, para := range bioParagraphs {
		entries = append(entries, ArtifactEntry{
			ID:              fmt.Sprintf("entry:bio-paragraph-%d", idx),
			Kind:            "paragraph",
			Text:            para,
			ClaimIDs:        bioParagraphClaimIDs[idx],
			EvidenceIDs:     bioParagraphEvidenceIDs[idx],
			SourceObjectIDs: bioParagraphSourceIDs[idx],
		})
	}

	sections = append(sections, ArtifactSection{
		ID:      "section:biography",
		Kind:    "Biography",
		Heading: "Professional Biography",
		Entries: entries,
	})

	return sections, warnings
}

type planningClaim struct {
	ID                claims.ClaimID
	Statement         string
	SourceObjectIDs   []string
	EvidenceObjectIDs []string
}

func planningClaimFromClaim(c claims.Claim) planningClaim {
	return planningClaim{
		ID:                c.ID,
		Statement:         c.Statement,
		SourceObjectIDs:   c.SourceObjectIDs,
		EvidenceObjectIDs: c.EvidenceObjectIDs,
	}
}
