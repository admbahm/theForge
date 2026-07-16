package rendering

import (
	"fmt"
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
)

// RenderCoverLetter processes the plan to build a structured Cover Letter.
func RenderCoverLetter(req RenderRequest, policy claims.Policy) ([]ArtifactSection, []string) {
	var sections []ArtifactSection
	var warnings []string

	selected := req.Plan.SelectedClaims

	// 1. Recipient Details Section
	recipientName := "Hiring Manager"
	companyName := "the Target Company"
	roleTitle := "the Position"

	if req.Plan.Target != nil {
		if req.Plan.Target.RoleTitle != "" {
			roleTitle = req.Plan.Target.RoleTitle
		}
		if req.Plan.Target.Company != "" {
			companyName = req.Plan.Target.Company
		}
	}

	if companyName == "the Target Company" {
		// Try to find the company name in the selected claims' organizations to customize,
		// or fallback to a standard company indicator.
		for _, pc := range selected {
			if len(pc.Claim.Organizations) > 0 {
				companyName = pc.Claim.Organizations[0]
				break
			}
		}
	}

	// 2. Header and Contact Info
	senderName := "Candidate"
	if req.Options.Contact.Name != "" {
		senderName = req.Options.Contact.Name
	}

	var headerTexts []string
	if req.Options.Contact.Address != "" {
		headerTexts = append(headerTexts, req.Options.Contact.Address)
	}
	if req.Options.Contact.Phone != "" {
		headerTexts = append(headerTexts, req.Options.Contact.Phone)
	}
	if req.Options.Contact.Email != "" {
		headerTexts = append(headerTexts, req.Options.Contact.Email)
	}

	sections = append(sections, ArtifactSection{
		ID:      "section:cover-letter-header",
		Kind:    "Header",
		Heading: senderName,
		Entries: []ArtifactEntry{
			{
				ID:   "entry:header-contact",
				Kind: "paragraph",
				Text: strings.Join(headerTexts, " | "),
			},
		},
	})

	// 3. Salutation
	sections = append(sections, ArtifactSection{
		ID:      "section:salutation",
		Kind:    "Salutation",
		Heading: "",
		Entries: []ArtifactEntry{
			{
				ID:   "entry:salutation-text",
				Kind: "paragraph",
				Text: fmt.Sprintf("Dear %s at %s,", recipientName, companyName),
			},
		},
	})

	// 4. Introduction
	introText := fmt.Sprintf("I am writing to express my strong interest in the %s position. With a solid foundation in systems engineering and a track record of driving key infrastructure initiatives, I am confident in my ability to deliver high-quality contributions to your team.", roleTitle)
	sections = append(sections, ArtifactSection{
		ID:      "section:introduction",
		Kind:    "Introduction",
		Heading: "",
		Entries: []ArtifactEntry{
			{
				ID:   "entry:intro-text",
				Kind: "paragraph",
				Text: introText,
			},
		},
	})

	// 5. Body / Alignment Paragraph (Selected Accomplishments)
	var bodyEntries []ArtifactEntry
	achievementCount := 0
	for _, pc := range selected {
		if pc.Claim.Kind == claims.KindAccomplishment || pc.Claim.Kind == claims.KindMeasurableResult {
			if achievementCount < 3 && pc.Claim.Statement != "" {
				bodyEntries = append(bodyEntries, ArtifactEntry{
					ID:              fmt.Sprintf("entry:alignment-achievement-%d", achievementCount),
					Kind:            "bullet",
					Text:            pc.Claim.Statement,
					ClaimIDs:        []claims.ClaimID{pc.Claim.ID},
					SourceObjectIDs: pc.Claim.SourceObjectIDs,
				})
				achievementCount++
			}
		}
	}

	if len(bodyEntries) > 0 {
		sections = append(sections, ArtifactSection{
			ID:      "section:alignment",
			Kind:    "Body",
			Heading: "Selected Accomplishments",
			Entries: bodyEntries,
		})
	}

	// 6. Conclusion
	conclusionText := "Thank you for your time and consideration. I welcome the opportunity to discuss how my background and verified experience align with your team's operational needs."
	sections = append(sections, ArtifactSection{
		ID:      "section:conclusion",
		Kind:    "Conclusion",
		Heading: "",
		Entries: []ArtifactEntry{
			{
				ID:   "entry:conclusion-text",
				Kind: "paragraph",
				Text: conclusionText,
			},
		},
	})

	// 7. Sign-off
	sections = append(sections, ArtifactSection{
		ID:      "section:sign-off",
		Kind:    "SignOff",
		Heading: "",
		Entries: []ArtifactEntry{
			{
				ID:   "entry:sign-off-text",
				Kind: "paragraph",
				Text: fmt.Sprintf("Sincerely,\n\n%s", senderName),
			},
		},
	})

	return sections, warnings
}
