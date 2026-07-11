package export

import (
	"fmt"
	"io"
	"strings"

	"github.com/admbahm/theForge/ckb/rendering"
)

// TextOptions controls plain text formatting configurations.
type TextOptions struct {
	IncludeHeadings bool
	DebugProvenance bool
	ASCIIBorders    bool
}

// ExportText serializes the artifact into deterministic, clean plain text without Markdown artifacts.
func ExportText(artifact *rendering.Artifact, w io.Writer, options TextOptions) error {
	var sb strings.Builder

	// Title and Subtitle
	sb.WriteString(fmt.Sprintf("%s\n", strings.ToUpper(strings.TrimSpace(artifact.Title))))
	if artifact.Subtitle != "" {
		sb.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(artifact.Subtitle)))
	}
	sb.WriteString("\n")

	for _, sec := range artifact.Sections {
		if options.IncludeHeadings {
			if options.ASCIIBorders {
				sb.WriteString(fmt.Sprintf("=== %s ===\n\n", strings.ToUpper(strings.TrimSpace(sec.Heading))))
			} else {
				sb.WriteString(fmt.Sprintf("%s\n\n", strings.ToUpper(strings.TrimSpace(sec.Heading))))
			}
		}

		inBulletList := false
		for _, ent := range sec.Entries {
			text := strings.TrimSpace(ent.Text)
			if text == "" {
				continue
			}

			// Clean Markdown bold/italic formatting tags
			text = stripMarkdownArtifacts(text)
			text = cleanTrailingSpaces(text)

			switch ent.Kind {
			case "paragraph":
				if inBulletList {
					sb.WriteString("\n")
					inBulletList = false
				}
				sb.WriteString(fmt.Sprintf("%s\n\n", text))
			case "header":
				if inBulletList {
					sb.WriteString("\n")
					inBulletList = false
				}
				if options.ASCIIBorders {
					sb.WriteString(fmt.Sprintf("--- %s ---\n\n", text))
				} else {
					sb.WriteString(fmt.Sprintf("%s\n\n", text))
				}
			case "subheader":
				if inBulletList {
					sb.WriteString("\n")
					inBulletList = false
				}
				if options.ASCIIBorders {
					sb.WriteString(fmt.Sprintf("  %s  \n\n", text))
				} else {
					sb.WriteString(fmt.Sprintf("%s\n\n", text))
				}
			case "bullet":
				inBulletList = true
				sb.WriteString(fmt.Sprintf("* %s\n", text))
			default:
				if inBulletList {
					sb.WriteString("\n")
					inBulletList = false
				}
				sb.WriteString(fmt.Sprintf("%s\n\n", text))
			}
		}
		if inBulletList {
			sb.WriteString("\n")
		}
	}

	doc := sb.String()
	lines := strings.Split(doc, "\n")
	var cleanedLines []string
	for _, l := range lines {
		cleanedLines = append(cleanedLines, strings.TrimRight(l, " \t"))
	}
	docClean := strings.Join(cleanedLines, "\n")

	// Ensure exactly one trailing newline
	docClean = strings.TrimRight(docClean, "\n") + "\n"

	_, err := io.WriteString(w, docClean)
	return err
}

func stripMarkdownArtifacts(text string) string {
	res := text
	res = strings.ReplaceAll(res, "**", "")
	res = strings.ReplaceAll(res, "*", "")
	res = strings.ReplaceAll(res, "`", "")
	return res
}
