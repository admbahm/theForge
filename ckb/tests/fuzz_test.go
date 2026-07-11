package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/admbahm/theForge/ckb/parser"
)

func FuzzObjectID(f *testing.F) {
	f.Add("exp:stark-devops")
	f.Add("invalid_id")
	f.Add("proj:titan-consolidation")
	f.Add("ev:git-repo")

	f.Fuzz(func(t *testing.T, id string) {
		// Mock a markdown document string with the fuzzed ID
		docTemplate := `| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | %s |
| **Type** | Experience |
| **Status** | Active |
| **Verification Level** | Self-Attested |
| **Confidence** | 1.00 |
| **Visibility** | Public |
| **Source** | Self |
| **Last Updated** | 2026-07-11 |
| **Lifecycle State** | Active |

---

## 1. Role Context
Context text.
`
		doc := fmt.Sprintf(docTemplate, id)

		// Create a temporary file
		tmpFile, err := osCreateTempFile(doc)
		if err != nil {
			return
		}
		defer tmpFile.Close()

		opts := parser.ParseOptions{
			Strict:          true,
			ValidatePrivacy: true,
			Limits:          parser.DefaultLimits(),
		}

		// Ensure parsing completes safely without panics
		_ = parser.ParseFiles(context.Background(), []string{tmpFile.Name()}, opts)
	})
}

func FuzzMetadataTable(f *testing.F) {
	f.Add("Schema Version")
	f.Add("Value")
	f.Add("| Metadata | Value |")

	f.Fuzz(func(t *testing.T, tableInput string) {
		doc := fmt.Sprintf(`| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | exp:fuzz-test |
| **Type** | Experience |
| **Status** | Active |
| **Verification Level** | Self-Attested |
| **Confidence** | 1.00 |
| **Visibility** | Public |
| **Source** | Self |
| **Last Updated** | 2026-07-11 |
| **Lifecycle State** | Active |
| **%s** | FuzzVal |
`, tableInput)

		tmpFile, err := osCreateTempFile(doc)
		if err != nil {
			return
		}
		defer tmpFile.Close()

		opts := parser.ParseOptions{
			Strict:          true,
			ValidatePrivacy: true,
			Limits:          parser.DefaultLimits(),
		}

		_ = parser.ParseFiles(context.Background(), []string{tmpFile.Name()}, opts)
	})
}

func FuzzRelationships(f *testing.F) {
	f.Add("proj:titan-consolidation")
	f.Add("invalid-id, ev:cert-aws")

	f.Fuzz(func(t *testing.T, relInput string) {
		doc := fmt.Sprintf(`| Metadata | Value |
| :--- | :--- |
| **Schema Version** | 1.0 |
| **ID** | exp:fuzz-rel |
| **Type** | Experience |
| **Status** | Active |
| **Verification Level** | Self-Attested |
| **Confidence** | 1.00 |
| **Visibility** | Public |
| **Source** | Self |
| **Last Updated** | 2026-07-11 |
| **Lifecycle State** | Active |
| **Related Projects** | %s |
`, relInput)

		tmpFile, err := osCreateTempFile(doc)
		if err != nil {
			return
		}
		defer tmpFile.Close()

		opts := parser.ParseOptions{
			Strict:          true,
			ValidatePrivacy: true,
			Limits:          parser.DefaultLimits(),
		}

		// Ensure parsing completes safely without panics
		res := parser.ParseFiles(context.Background(), []string{tmpFile.Name()}, opts)
		if res.KnowledgeBase != nil {
			// Traverse the loaded relationships mapping to assert no panics
			for _, rel := range res.KnowledgeBase.Relationships {
				_ = rel.TargetID
				_ = rel.Type
			}
		}
	})
}

type tempFile struct {
	*testing.T
	file *os.File
}

func (tf *tempFile) Close() {
	if tf.file != nil {
		tf.file.Close()
		os.Remove(tf.file.Name())
	}
}

func (tf *tempFile) Name() string {
	if tf.file != nil {
		return tf.file.Name()
	}
	return ""
}

func osCreateTempFile(content string) (*tempFile, error) {
	// Create temporary directory in standard path
	f, err := os.CreateTemp("", "ckb-fuzz-*.md")
	if err != nil {
		return nil, err
	}
	_, err = f.Write([]byte(content))
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	f.Close()

	// Reopen for reference
	fOpen, err := os.Open(f.Name())
	if err != nil {
		os.Remove(f.Name())
		return nil, err
	}

	return &tempFile{file: fOpen}, nil
}
