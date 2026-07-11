package rendering

import (
	"fmt"
	"strings"

	"github.com/admbahm/theForge/ckb/claims"
)

var verbToPast = map[string]string{
	"lead":       "led",
	"serve":      "served",
	"utilize":    "utilized",
	"reduce":     "reduced",
	"improve":    "improved",
	"contribute": "contributed",
	"support":    "supported",
	"implement":  "implemented",
	"deliver":    "delivered",
	"own":        "owned",
	"direct":     "directed",
	"manage":     "managed",
	"design":     "designed",
	"build":      "built",
	"develop":    "developed",
}

var verbToPresent = map[string]string{
	"led":         "lead",
	"served":      "serve",
	"utilized":    "utilize",
	"reduced":     "reduce",
	"improved":    "improve",
	"contributed": "contribute",
	"supported":   "support",
	"implemented": "implement",
	"delivered":   "deliver",
	"owned":       "own",
	"directed":    "direct",
	"managed":     "manage",
	"designed":    "design",
	"built":       "build",
	"developed":   "develop",
}

// AdaptVerbTense changes the tense of the leading verb in a sentence if recognized.
func AdaptVerbTense(sentence string, targetTense string) string {
	if sentence == "" {
		return ""
	}
	words := strings.Fields(sentence)
	if len(words) == 0 {
		return sentence
	}

	firstWord := words[0]
	// Clean punctuation from first word to check match
	cleanFirst := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return r
		}
		return -1
	}, firstWord)

	cleanLower := strings.ToLower(cleanFirst)
	replaced := firstWord

	if targetTense == "past" {
		if past, exists := verbToPast[cleanLower]; exists {
			replaced = matchCase(firstWord, past)
		}
	} else if targetTense == "present" {
		if pres, exists := verbToPresent[cleanLower]; exists {
			replaced = matchCase(firstWord, pres)
		}
	}

	// Reconstruct sentence
	words[0] = replaced + strings.TrimPrefix(firstWord, cleanFirst)
	return strings.Join(words, " ")
}

func matchCase(original, replacement string) string {
	if len(original) == 0 {
		return replacement
	}
	// Uppercase first letter if original is capitalized
	if original[0] >= 'A' && original[0] <= 'Z' {
		return strings.Title(replacement)
	}
	return replacement
}

// JoinList joins string array with Oxford comma.
func JoinList(items []string) string {
	n := len(items)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return items[0]
	}
	if n == 2 {
		return items[0] + " and " + items[1]
	}
	return strings.Join(items[:n-1], ", ") + ", and " + items[n-1]
}

// FormatMetric converts a claims.Metric to a string depending on MetricStyle (Standard vs Compact).
func FormatMetric(m claims.Metric, style string) string {
	prefix := ""
	if style == "Compact" && (m.IsAmbiguous || m.Status == "Ambiguous" || m.Status == "Target" || m.Status == "Estimate") {
		prefix = "~"
	}

	valStr := fmt.Sprintf("%g", m.Value)
	unitStr := m.Unit

	if unitStr == "%" {
		return prefix + valStr + "%"
	}
	return prefix + valStr + " " + unitStr
}

// ExpandAbbreviations replaces GFM/tech abbreviations in standard styles.
func ExpandAbbreviations(text string, style string) string {
	if style != "Expanded" {
		return text
	}
	replacements := map[string]string{
		"GCP": "Google Cloud Platform",
		"AWS": "Amazon Web Services",
		"SRE": "Site Reliability Engineering",
	}
	res := text
	for abbrev, full := range replacements {
		res = strings.ReplaceAll(res, abbrev, full)
	}
	return res
}
