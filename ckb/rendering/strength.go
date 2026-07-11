package rendering

import (
	"fmt"
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
)

// getMaxRanks returns the highest keyword ranks across 4 independent dimensions.
func getMaxRanks(text string) (ownership, skill, certainty, scope int) {
	textLower := " " + strings.ToLower(text) + " "

	// Clean trailing punctuation
	textLower = strings.ReplaceAll(textLower, ".", " ")
	textLower = strings.ReplaceAll(textLower, ",", " ")
	textLower = strings.ReplaceAll(textLower, "*", " ")

	// 1. Ownership Strength
	ownershipRanks := map[string]int{
		"observed":    1,
		"assisted":    2,
		"contributed": 3,
		"supported":   4,
		"implemented": 5,
		"delivered":   6,
		"led":         7,
		"owned":       8,
		"directed":    9,
	}
	for word, r := range ownershipRanks {
		if strings.Contains(textLower, " "+word+" ") {
			if r > ownership {
				ownership = r
			}
		}
	}

	// 2. Skill Strength (phrases first, then words)
	skillPhrases := []struct {
		phrase string
		rank   int
	}{
		{"exposed to", 1},
		{"trained in", 2},
		{"familiar with", 3},
		{"working knowledge", 4},
	}
	for _, sp := range skillPhrases {
		if strings.Contains(textLower, " "+sp.phrase+" ") {
			if sp.rank > skill {
				skill = sp.rank
			}
		}
	}

	skillRanks := map[string]int{
		"proficient": 5,
		"advanced":   6,
		"expert":     7,
	}
	for word, r := range skillRanks {
		if strings.Contains(textLower, " "+word+" ") {
			if r > skill {
				skill = r
			}
		}
	}

	// 3. Outcome Certainty
	certaintyRanks := map[string]int{
		"considered": 1,
		"planned":    2,
		"targeted":   3,
		"estimated":  4,
		"forecast":   5,
		"attempted":  6,
		"observed":   7,
		"measured":   8,
		"achieved":   9,
		"verified":   10,
	}
	for word, r := range certaintyRanks {
		if strings.Contains(textLower, " "+word+" ") {
			if r > certainty {
				certainty = r
			}
		}
	}

	// 4. Scope Strength
	scopeRanks := map[string]int{
		"individual":   1,
		"component":    2,
		"project":      3,
		"team":         4,
		"department":   5,
		"organization": 6,
		"enterprise":   7,
	}
	for word, r := range scopeRanks {
		if strings.Contains(textLower, " "+word+" ") {
			if r > scope {
				scope = r
			}
		}
	}

	return
}

// CheckStrengthPreservation returns an error if the renderedText has a higher strength rank in any dimension than the source claim.
func CheckStrengthPreservation(sourceClaim claims.Claim, renderedText string) error {
	srcOwn, srcSkill, srcCert, srcScope := getMaxRanks(sourceClaim.Statement)
	destOwn, destSkill, destCert, destScope := getMaxRanks(renderedText)

	if destOwn > srcOwn && srcOwn > 0 {
		return fmt.Errorf("ownership strength upgrade: source %d -> dest %d", srcOwn, destOwn)
	}
	if destSkill > srcSkill && srcSkill > 0 {
		return fmt.Errorf("skill strength upgrade: source %d -> dest %d", srcSkill, destSkill)
	}
	if destCert > srcCert && srcCert > 0 {
		return fmt.Errorf("outcome certainty upgrade: source %d -> dest %d", srcCert, destCert)
	}
	if destScope > srcScope && srcScope > 0 {
		return fmt.Errorf("scope strength upgrade: source %d -> dest %d", srcScope, destScope)
	}
	return nil
}
