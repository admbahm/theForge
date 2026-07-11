# Supported API Contract — Career Knowledge Base (CKB)

This document describes the public boundaries, configuration parameters, concurrency guarantees, and compatibility guidelines for the Career Knowledge Base (CKB) package.

---

## 1. Supported Packages & Entrypoints

Downstream Forge modules (such as resume, CV, and STAR story generators) must interface exclusively with the following packages:

*   `github.com/admbahm/theForge/ckb/model`: Domain structures, enums, and diagnostic types.
*   `github.com/admbahm/theForge/ckb/parser`: Walk directories and parse Markdown documents.
*   `github.com/admbahm/theForge/ckb/planning`: Build plan entrypoints and policies.
*   `github.com/admbahm/theForge/ckb/rendering`: Compile plans into structured intermediate models.
*   `github.com/admbahm/theForge/ckb/export`: Exporters for JSON, Markdown, Plain Text, and Provenance Sidecar.

---

## 2. Public API Surface

### Types & Structs

#### `model.KnowledgeBase`
Holds the compiled graph nodes (Objects) and edges (Relationships).
```go
type KnowledgeBase struct {
	Objects       map[string]*Object
	Relationships []Relationship
}
```

#### `model.Object`
Represents a structured record parsed from a CKB Markdown document.
```go
type Object struct {
	ID            string
	Type          ObjectType
	Metadata      Metadata
	Sections      []Section
	Relationships []Relationship
	SourceFile    string
}
```

#### `parser.ParseOptions`
Controls strictness checks, privacy scans, and resource boundaries.
```go
type ParseOptions struct {
	Strict          bool
	ValidatePrivacy bool
	Limits          Limits
}
```

#### `parser.Limits`
Defines safety thresholds to prevent pathological inputs from causing crashes.
```go
type Limits struct {
	MaxFileSize     int64
	MaxLineLength   int
	MaxObjectCount  int
	MaxMetadataRows int
}
```

---

## 3. Entrypoint Functions

### Parsing Directory
Scans the relative vault path, filters Markdown files, and compiles the CKB.
```go
func ParseDirectory(ctx context.Context, root string, options ParseOptions) Result
```

### Parsing Batch Files
Processes a specific slice of file paths.
```go
func ParseFiles(ctx context.Context, files []string, options ParseOptions) Result
```

### Exporting CKB Graph
Exports the compiled KnowledgeBase and its diagnostics as a deterministic, sorted JSON payload.
```go
func ExportJSON(kb *model.KnowledgeBase, diagnostics []model.Diagnostic, w io.Writer) error
```

---

## 4. Default Limits Configuration
Callers can fetch safe default limits using `parser.DefaultLimits()`:
*   `MaxFileSize`: 1 MB (`1048576` bytes)
*   `MaxLineLength`: 4096 bytes
*   `MaxObjectCount`: 1000 files
*   `MaxMetadataRows`: 32 rows

---

## 5. Ownership, Mutability & Concurrency Rules

*   **Read-Only Guarantees**: All structures, maps, and slices returned by `parser.ParseDirectory` or `parser.ParseFiles` belong to the compiled graph. Callers must treat the returned `*model.KnowledgeBase` as **immutable**. Downstream generation engines must not edit, insert, or remove elements from the returned data structures.
*   **Thread Safety**: The parser and validator maintain zero global mutable state. It is fully safe to run concurrent parser instances across independent goroutines.
*   **Context Honor**: Both batch and directory parsing monitor `ctx.Done()`. If a parent context expires or is cancelled, processing terminates immediately, returning a fatal `CKB-STRUCTURE-MALFORMED` diagnostic.

---

## 6. Usage Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/admbahm/theForge/ckb/export"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/parser"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	}

	// Parse CKB vault directory
	res := parser.ParseDirectory(ctx, "./vault", opts)

	// Check for fatal/error diagnostics
	hasErrors := false
	for _, diag := range res.Diagnostics {
		if diag.Severity == model.SeverityFatal || diag.Severity == model.SeverityError {
			fmt.Printf("[%s] Error: %s in %s (Line: %d)\n", diag.Code, diag.Message, diag.Source.FilePath, diag.Source.Line)
			hasErrors = true
		}
	}

	if hasErrors {
		os.Exit(1)
	}

	// Deterministic serialization to JSON
	outFile, err := os.Create("ckb_export.json")
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	if err := export.ExportJSON(res.KnowledgeBase, res.Diagnostics, outFile); err != nil {
		panic(err)
	}
}
```
