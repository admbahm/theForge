package export

import (
	"fmt"
	"io"
	"strings"

	"github.com/admbahm/theForge/ckb/rendering"
)

// MarkdownOptions controls details of the markdown format output.
type MarkdownOptions struct {
	IncludeHeadings bool
	DebugProvenance bool
}

// ExportMarkdown serializes the artifact into deterministic, portable Markdown.
func ExportMarkdown(artifact *rendering.Artifact, w io.Writer, options MarkdownOptions) error {
	var sb strings.Builder

	// Title and Subtitle
	sb.WriteString(fmt.Sprintf("# %s\n\n", strings.TrimSpace(artifact.Title)))
	if artifact.Subtitle != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", strings.TrimSpace(artifact.Subtitle)))
	}

	for _, sec := range artifact.Sections {
		if options.IncludeHeadings {
			sb.WriteString(fmt.Sprintf("## %s\n\n", strings.TrimSpace(sec.Heading)))
		}

		inBulletList := false
		for _, ent := range sec.Entries {
			text := strings.TrimSpace(ent.Text)
			if text == "" {
				continue
			}

			// Clean any trailing spaces inside text line by line
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
				sb.WriteString(fmt.Sprintf("### %s\n\n", text))
			case "subheader":
				if inBulletList {
					sb.WriteString("\n")
					inBulletList = false
				}
				sb.WriteString(fmt.Sprintf("#### %s\n\n", text))
			case "bullet":
				inBulletList = true
				sb.WriteString(fmt.Sprintf("- %s\n", text))
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

	// Clean trailing spaces and normalize whitespace across the entire document
	doc := sb.String()
	lines := strings.Split(doc, "\n")
	var cleanedLines []string
	for _, l := range lines {
		cleanedLines = append(cleanedLines, strings.TrimRight(l, " \t"))
	}
	docClean := strings.Join(cleanedLines, "\n")

	// Ensure exactly one trailing empty line
	docClean = strings.TrimRight(docClean, "\n") + "\n"

	_, err := io.WriteString(w, docClean)
	return err
}

func cleanTrailingSpaces(text string) string {
	lines := strings.Split(text, "\n")
	for idx, l := range lines {
		lines[idx] = strings.TrimRight(l, " \t")
	}
	return strings.Join(lines, "\n")
}
