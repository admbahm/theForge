package rendering

import (
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
)

// CanCombineSkills checks if a set of skill claims can be safely combined.
func CanCombineSkills(c1, c2 claims.Claim) bool {
	if c1.Visibility != c2.Visibility {
		return false
	}
	if c1.Kind != c2.Kind {
		return false
	}
	return true
}

// CombineSkillStatements merges multiple technical skill statements into a single line.
func CombineSkillStatements(skillClaims []claims.Claim, abbreviationStyle string) string {
	var techNames []string
	for _, c := range skillClaims {
		// Extract technology name from statement
		tName := strings.TrimPrefix(c.Statement, "Utilized technology: ")
		tName = strings.TrimPrefix(tName, "Practiced methodology: ")
		tName = strings.TrimSuffix(tName, ".")
		tName = strings.TrimSpace(tName)

		if tName != "" {
			tName = ExpandAbbreviations(tName, abbreviationStyle)
			techNames = append(techNames, tName)
		}
	}

	if len(techNames) == 0 {
		return ""
	}

	return "Utilized technologies: " + JoinList(techNames) + "."
}
