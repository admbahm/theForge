package parser

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/validation"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Allowed Type to Prefix mapping
var allowedTypes = map[string]string{
	"Profile":        "profile",
	"Timeline":       "timeline",
	"Experience":     "exp",
	"Project":        "proj",
	"Skill":          "skill",
	"Accomplishment": "acc",
	"Credential":     "cred",
	"Contribution":   "contrib",
	"Reference":      "ref",
	"Evidence":       "ev",
	"Education":      "edu",
}

// Canonical required metadata table keys
var canonicalRequiredKeys = []string{
	"Schema Version",
	"ID",
	"Type",
	"Status",
	"Verification Level",
	"Confidence",
	"Visibility",
	"Source",
	"Last Updated",
	"Lifecycle State",
}

var canonicalOptionalKeys = map[string]bool{
	"Related Documents":  true,
	"Related Experience": true,
	"Related Projects":   true,
	"Related Evidence":   true,
	"Tags":               true,
}

// PII Regex Filters
var (
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phoneRegex = regexp.MustCompile(`\b(?:\+\d{1,2}\s)?\(?\d{3}\)?[\s.-]?\d{3}\s?[\s.-]?\d{4}\b`)
)

// ID standard regex
var idFormatRegex = regexp.MustCompile(`^[a-z0-9]+:[a-z0-9-]+$`)

// ParseOptions configures parsing pipelines.
type ParseOptions struct {
	Strict          bool
	ValidatePrivacy bool
	Limits          Limits
}

// Result holds parsed graph nodes and structural diagnostics.
type Result struct {
	KnowledgeBase *model.KnowledgeBase
	Diagnostics   []model.Diagnostic
}

// ParseDirectory walks a root path, parses CKB documents, and returns results.
func ParseDirectory(ctx context.Context, root string, options ParseOptions) Result {
	res := Result{
		KnowledgeBase: model.NewKnowledgeBase(),
		Diagnostics:   make([]model.Diagnostic, 0),
	}

	files, err := DiscoverFiles(root)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
			Code:     model.CodeLimitsExceeded,
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("Failed to scan directory: %v", err),
			Source:   model.SourceLocation{FilePath: root},
		})
		return res
	}

	return ParseFiles(ctx, files, options)
}

// ParseFiles parses list of target file paths.
func ParseFiles(ctx context.Context, files []string, options ParseOptions) Result {
	res := Result{
		KnowledgeBase: model.NewKnowledgeBase(),
		Diagnostics:   make([]model.Diagnostic, 0),
	}

	// Clean and copy the files slice to ensure stable, sorted execution
	sortedFiles := make([]string, len(files))
	for i, f := range files {
		abs, err := filepath.Abs(f)
		if err == nil {
			sortedFiles[i] = abs
		} else {
			sortedFiles[i] = filepath.Clean(f)
		}
	}
	sort.Slice(sortedFiles, func(i, j int) bool {
		return strings.ToLower(filepath.ToSlash(sortedFiles[i])) < strings.ToLower(filepath.ToSlash(sortedFiles[j]))
	})

	// 1. Enforce limits: maximum file objects
	limit := options.Limits
	if limit.MaxObjectCount == 0 {
		limit = DefaultLimits()
	}

	if len(sortedFiles) > limit.MaxObjectCount {
		res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
			Code:     model.CodeLimitsExceeded,
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("Object count limit exceeded: got %d files, max allowed is %d", len(sortedFiles), limit.MaxObjectCount),
			Source:   model.SourceLocation{FilePath: "ckb"},
		})
		return res
	}

	var parsedObjects []*model.Object

	for _, file := range sortedFiles {
		// Context check
		select {
		case <-ctx.Done():
			res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
				Code:     model.CodeStructureMalformed,
				Severity: model.SeverityFatal,
				Message:  "Parsing was cancelled by context timeout.",
				Source:   model.SourceLocation{FilePath: file},
			})
			return res
		default:
		}

		obj, diagnostics := parseSingleFile(file, limit, options)
		res.Diagnostics = append(res.Diagnostics, diagnostics...)

		if obj != nil {
			// Check if ID is already registered to catch duplicates early
			if existing, exists := res.KnowledgeBase.Objects[obj.ID]; exists {
				res.Diagnostics = append(res.Diagnostics, model.Diagnostic{
					Code:     model.CodeIdentityDuplicateID,
					Severity: model.SeverityFatal,
					Message:  fmt.Sprintf("duplicate ID: ID %q is already declared in %s", obj.ID, existing.SourceFile),
					Source:   model.SourceLocation{FilePath: file, Line: 1},
					ObjectID: obj.ID,
				})
			} else {
				parsedObjects = append(parsedObjects, obj)
				res.KnowledgeBase.Objects[obj.ID] = obj
				res.KnowledgeBase.Relationships = append(res.KnowledgeBase.Relationships, obj.Relationships...)

				if obj.Type == model.TypeEvidence {
					source, err := os.ReadFile(file)
					if err == nil {
						registerEvidenceCatalogIDs(res.KnowledgeBase, obj, source)
					}
				}
			}
		}
	}

	// 2. Perform global relationship graph validations
	graphDiags := validation.ValidateGraph(res.KnowledgeBase)
	res.Diagnostics = append(res.Diagnostics, graphDiags...)

	// Deterministic sorting of final diagnostics
	sortDiagnostics(res.Diagnostics)

	return res
}

func parseSingleFile(path string, limits Limits, options ParseOptions) (*model.Object, []model.Diagnostic) {
	var diags []model.Diagnostic

	info, err := os.Stat(path)
	if err != nil {
		diags = append(diags, model.Diagnostic{
			Code:     model.CodeStructureMalformed,
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("File stat failed: %v", err),
			Source:   model.SourceLocation{FilePath: path},
		})
		return nil, diags
	}

	// Limit Check: File size
	if info.Size() > limits.MaxFileSize {
		diags = append(diags, model.Diagnostic{
			Code:     model.CodeLimitsExceeded,
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("File size %d exceeds limit of %d bytes", info.Size(), limits.MaxFileSize),
			Source:   model.SourceLocation{FilePath: path},
		})
		return nil, diags
	}

	data, err := os.ReadFile(path)
	if err != nil {
		diags = append(diags, model.Diagnostic{
			Code:     model.CodeStructureMalformed,
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("Failed to read file: %v", err),
			Source:   model.SourceLocation{FilePath: path},
		})
		return nil, diags
	}

	// Limit Check: Line lengths & malformed UTF-8
	lines := strings.Split(string(data), "\n")
	for lineNum, line := range lines {
		if len(line) > limits.MaxLineLength {
			diags = append(diags, model.Diagnostic{
				Code:     model.CodeLimitsExceeded,
				Severity: model.SeverityFatal,
				Message:  fmt.Sprintf("Line %d length %d exceeds limit of %d bytes", lineNum+1, len(line), limits.MaxLineLength),
				Source:   model.SourceLocation{FilePath: path, Line: lineNum + 1},
			})
			return nil, diags
		}
	}

	// Privacy validate prior to parsing AST (saves compute)
	if options.ValidatePrivacy {
		if piiDiags := scanPII(path, string(data)); len(piiDiags) > 0 {
			diags = append(diags, piiDiags...)
		}
	}

	// Build Goldmark Parser with Tables Extension
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
	)

	reader := text.NewReader(data)
	doc := md.Parser().Parse(reader, gparser.WithContext(gparser.NewContext()))

	// Parse Metadata Table
	metaNode, metaErrDiag := extractMetadataTableNode(doc, path)
	if metaErrDiag != nil {
		diags = append(diags, *metaErrDiag)
		return nil, diags
	}

	meta, metaDiags := parseMetadataTableFields(metaNode, data, path, limits)
	diags = append(diags, metaDiags...)

	// Fatal check: if metadata ID or Type was missing/failed, we cannot proceed safely
	if meta.ID == "" || meta.Type == "" {
		return nil, diags
	}

	// Parse body sections
	sections, sectionDiags := parseMarkdownBodySections(doc, data, path, limits)
	diags = append(diags, sectionDiags...)

	// Construct relationships from metadata
	relationships := constructRelations(meta, path)

	obj := &model.Object{
		ID:            meta.ID,
		Type:          meta.Type,
		SourceFile:    path,
		Metadata:      *meta,
		Sections:      sections,
		Relationships: relationships,
	}

	return obj, diags
}

func extractMetadataTableNode(doc ast.Node, path string) (*extast.Table, *model.Diagnostic) {
	// Find the first non-empty block
	var firstBlock ast.Node
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		firstBlock = child
		break
	}

	if firstBlock == nil {
		return nil, &model.Diagnostic{
			Code:        model.CodeMetadataMissingID,
			Severity:    model.SeverityFatal,
			Message:     "Document is empty.",
			Source:      model.SourceLocation{FilePath: path},
			Remediation: "Add standard metadata table at the top of the file.",
		}
	}

	table, ok := firstBlock.(*extast.Table)
	if !ok {
		return nil, &model.Diagnostic{
			Code:        model.CodeMetadataMissingID,
			Severity:    model.SeverityFatal,
			Message:     "File does not start with a Metadata Table block.",
			Source:      model.SourceLocation{FilePath: path, Line: 1},
			Remediation: "Ensure the Metadata Table is the first structural element in the file.",
		}
	}

	return table, nil
}

func parseMetadataTableFields(table *extast.Table, source []byte, path string, limits Limits) (*model.Metadata, []model.Diagnostic) {
	var diags []model.Diagnostic
	meta := &model.Metadata{
		RelatedDocs:  make([]string, 0),
		RelatedExps:  make([]string, 0),
		RelatedProjs: make([]string, 0),
		RelatedEvs:   make([]string, 0),
		Tags:         make([]string, 0),
	}

	// Count rows
	rowCount := 0
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		rowCount++
	}

	// Limit check: metadata rows
	if rowCount > limits.MaxMetadataRows {
		diags = append(diags, model.Diagnostic{
			Code:     model.CodeLimitsExceeded,
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("Metadata table row count %d exceeds limit of %d", rowCount, limits.MaxMetadataRows),
			Source:   model.SourceLocation{FilePath: path, Line: 1},
		})
		return meta, diags
	}

	var headerCells []*extast.TableCell
	var bodyRows []*extast.TableRow

	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		if th, ok := child.(*extast.TableHeader); ok {
			for cell := th.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if c, ok := cell.(*extast.TableCell); ok {
					headerCells = append(headerCells, c)
				}
			}
		} else if row, ok := child.(*extast.TableRow); ok {
			bodyRows = append(bodyRows, row)
		}
	}

	if len(headerCells) < 2 {
		diags = append(diags, model.Diagnostic{
			Code:     model.CodeMetadataMissingField,
			Severity: model.SeverityFatal,
			Message:  "Metadata table headers are malformed.",
			Source:   model.SourceLocation{FilePath: path, Line: 1},
		})
		return meta, diags
	}

	headerKey := strings.TrimSpace(string(headerCells[0].Text(source)))
	headerVal := strings.TrimSpace(string(headerCells[1].Text(source)))

	if headerKey != "Metadata" || headerVal != "Value" {
		diags = append(diags, model.Diagnostic{
			Code:     model.CodeMetadataMissingField,
			Severity: model.SeverityFatal,
			Message:  fmt.Sprintf("Invalid table headers: expected '| Metadata | Value |', got '| %s | %s |'", headerKey, headerVal),
			Source:   model.SourceLocation{FilePath: path, Line: 1},
		})
		return meta, diags
	}

	requiredIndex := 0
	seenKeys := make(map[string]bool)

	// Traverse data rows
	rowIdx := 1
	for _, row := range bodyRows {
		rowIdx++
		var rowCells []*extast.TableCell
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if c, ok := cell.(*extast.TableCell); ok {
				rowCells = append(rowCells, c)
			}
		}

		if len(rowCells) < 2 {
			continue
		}

		rawKey := string(rowCells[0].Text(source))
		valRaw := string(rowCells[1].Text(source))

		// Check duplicate rows
		if seenKeys[rawKey] {
			diags = append(diags, model.Diagnostic{
				Code:     model.CodeMetadataDuplicateField,
				Severity: model.SeverityFatal,
				Message:  fmt.Sprintf("Duplicate metadata row key %q declared", rawKey),
				Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
			})
			continue
		}
		seenKeys[rawKey] = true

		// Check bolding
		isBold := false
		for c := rowCells[0].FirstChild(); c != nil; c = c.NextSibling() {
			if emp, ok := c.(*ast.Emphasis); ok && emp.Level == 2 {
				isBold = true
				break
			}
		}

		if !isBold {
			diags = append(diags, model.Diagnostic{
				Code:     model.CodeMetadataMissingField,
				Severity: model.SeverityFatal,
				Message:  fmt.Sprintf("Metadata key %q must be bolded", rawKey),
				Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
			})
			continue
		}

		key := rawKey
		val := strings.TrimSpace(valRaw)

		// Check canonical required fields sequence
		if requiredIndex < len(canonicalRequiredKeys) {
			expectedKey := canonicalRequiredKeys[requiredIndex]
			if key != expectedKey {
				diags = append(diags, model.Diagnostic{
					Code:     model.CodeMetadataInvalidOrder,
					Severity: model.SeverityError,
					Message:  fmt.Sprintf("Strict Order Violation: expected key %q at row %d, got %q", expectedKey, requiredIndex+3, key),
					Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
					Field:    key,
				})
			}
			requiredIndex++

			switch key {
			case "Schema Version":
				meta.SchemaVersion = val
				if val != "1.0" {
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeVersionUnsupported,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("Unsupported CKB Schema Version %q", val),
						Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
						Field:    key,
					})
				}
			case "ID":
				meta.ID = val
				if !idFormatRegex.MatchString(val) {
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeIdentityInvalidID,
						Severity: model.SeverityFatal,
						Message:  fmt.Sprintf("Invalid ID format %q. Must match lowercase 'prefix:slug'", val),
						Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
						Field:    key,
					})
				}
			case "Type":
				meta.Type = model.ObjectType(val)
				if _, ok := allowedTypes[val]; !ok {
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeMetadataInvalidObjType,
						Severity: model.SeverityFatal,
						Message:  fmt.Sprintf("Invalid Object Type %q", val),
						Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
						Field:    key,
					})
				}
			case "Status":
				meta.Status = model.Status(val)
				switch val {
				case "Draft", "Active", "Deprecated":
				default:
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeMetadataInvalidEnum,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("Invalid Status enum value %q", val),
						Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
						Field:    key,
					})
				}
			case "Verification Level":
				meta.Verification = model.VerificationLevel(val)
				switch val {
				case "Unverified", "Self-Attested", "Artifact-Supported", "Independently-Verified", "Disputed", "Superseded":
				default:
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeMetadataInvalidEnum,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("Invalid Verification Level value %q", val),
						Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
						Field:    key,
					})
				}
			case "Confidence":
				c, err := strconv.ParseFloat(val, 64)
				if err != nil {
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeMetadataInvalidConfidence,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("Confidence %q is not a valid float", val),
						Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
						Field:    key,
					})
				} else {
					meta.Confidence = c
					if c < 0.0 || c > 1.0 {
						diags = append(diags, model.Diagnostic{
							Code:     model.CodeMetadataInvalidConfidence,
							Severity: model.SeverityError,
							Message:  fmt.Sprintf("confidence %.2f must be between 0.00 and 1.00", c),
							Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
							Field:    key,
						})
					}
				}
			case "Visibility":
				meta.Visibility = model.Visibility(val)
				switch val {
				case "Public", "Confidential", "Internal":
				default:
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeMetadataInvalidEnum,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("Invalid Visibility value %q", val),
						Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
						Field:    key,
					})
				}
			case "Source":
				meta.Source = val
			case "Last Updated":
				meta.LastUpdated = parseDateQuiet(val)
				_, err := time.Parse("2006-01-02", val)
				if err != nil {
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeMetadataInvalidDate,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("invalid Last Updated date format %q", val),
						Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
						Field:    key,
					})
				}
			case "Lifecycle State":
				meta.Lifecycle = model.LifecycleState(val)
				switch val {
				case "Planned", "Active", "Completed", "Archived":
				default:
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeMetadataInvalidEnum,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("Invalid Lifecycle State value %q", val),
						Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
						Field:    key,
					})
				}
			}
		} else {
			// Optional fields
			if !canonicalOptionalKeys[key] {
				diags = append(diags, model.Diagnostic{
					Code:     model.CodeMetadataUnknownField,
					Severity: model.SeverityError,
					Message:  fmt.Sprintf("unexpected metadata key %q", key),
					Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
					Field:    key,
				})
				continue
			}

			// Parse lists
			var items []string
			if val != "None" && val != "" {
				parts := strings.Split(val, ",")
				for _, part := range parts {
					partClean := strings.TrimSpace(part)
					if partClean != "" {
						items = append(items, partClean)
					}
				}
			}

			switch key {
			case "Related Documents":
				meta.RelatedDocs = items
			case "Related Experience":
				meta.RelatedExps = items
			case "Related Projects":
				meta.RelatedProjs = items
			case "Related Evidence":
				meta.RelatedEvs = items
			case "Tags":
				meta.Tags = items
			}
		}
	}

	if requiredIndex < len(canonicalRequiredKeys) {
		diags = append(diags, model.Diagnostic{
			Code:     model.CodeMetadataMissingField,
			Severity: model.SeverityFatal,
			Message:  "table is missing mandatory metadata keys",
			Source:   model.SourceLocation{FilePath: path, Line: rowIdx},
		})
	}

	return meta, diags
}

func parseDateQuiet(val string) time.Time {
	t, err := time.Parse("2006-01-02", val)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseMarkdownBodySections(doc ast.Node, source []byte, path string, limits Limits) ([]model.Section, []model.Diagnostic) {
	var diags []model.Diagnostic
	var sections []model.Section

	var currentSection *model.Section
	var sectionBytes bytes.Buffer

	// Walk children of document root
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		// Ignore first block if it's the metadata Table
		if child == doc.FirstChild() {
			if _, isTable := child.(*extast.Table); isTable {
				continue
			}
		}

		if child.Kind() == ast.KindHeading {
			h := child.(*ast.Heading)

			// Limit check: heading depth
			if h.Level > limits.MaxHeadingDepth {
				diags = append(diags, model.Diagnostic{
					Code:     model.CodeLimitsExceeded,
					Severity: model.SeverityError,
					Message:  fmt.Sprintf("Heading level %d exceeds depth limit of %d", h.Level, limits.MaxHeadingDepth),
					Source:   model.SourceLocation{FilePath: path, Line: 1}, // approximate line
				})
			}

			// If we have an active section, save its body text
			if currentSection != nil {
				currentSection.Body = sectionBytes.String()
				sections = append(sections, *currentSection)
				sectionBytes.Reset()
			}

			// Check limits: section count
			if len(sections) >= limits.MaxSectionCount {
				diags = append(diags, model.Diagnostic{
					Code:     model.CodeLimitsExceeded,
					Severity: model.SeverityFatal,
					Message:  fmt.Sprintf("Section count exceeds limit of %d", limits.MaxSectionCount),
					Source:   model.SourceLocation{FilePath: path},
				})
				return sections, diags
			}

			// Start new section node
			headingText := string(h.Text(source))
			// Standardize output to include visual heading prefix level hashes
			prefixHashes := strings.Repeat("#", h.Level)
			currentSection = &model.Section{
				Heading: fmt.Sprintf("%s %s", prefixHashes, headingText),
			}
		} else {
			// Write the raw bytes of body block to section buffer
			if currentSection != nil {
				// extract segments of text from source
				lines := child.Lines()
				for j := 0; j < lines.Len(); j++ {
					seg := lines.At(j)
					sectionBytes.Write(seg.Value(source))
				}
				sectionBytes.WriteByte('\n')
			}
		}
	}

	// Save final trailing section
	if currentSection != nil {
		currentSection.Body = sectionBytes.String()
		sections = append(sections, *currentSection)
	}

	return sections, diags
}

func constructRelations(meta *model.Metadata, path string) []model.Relationship {
	var rels []model.Relationship

	// Map RelatedDocs
	for _, target := range meta.RelatedDocs {
		rels = append(rels, model.Relationship{
			SourceID: meta.ID,
			TargetID: target,
			Type:     model.RelDocuments,
			Source:   model.SourceLocation{FilePath: path, Line: 1}, // table reference
		})
	}

	// Map RelatedExps
	for _, target := range meta.RelatedExps {
		rels = append(rels, model.Relationship{
			SourceID: meta.ID,
			TargetID: target,
			Type:     model.RelExperience,
			Source:   model.SourceLocation{FilePath: path, Line: 1},
		})
	}

	// Map RelatedProjs
	for _, target := range meta.RelatedProjs {
		rels = append(rels, model.Relationship{
			SourceID: meta.ID,
			TargetID: target,
			Type:     model.RelProjects,
			Source:   model.SourceLocation{FilePath: path, Line: 1},
		})
	}

	// Map RelatedEvs
	for _, target := range meta.RelatedEvs {
		rels = append(rels, model.Relationship{
			SourceID: meta.ID,
			TargetID: target,
			Type:     model.RelEvidence,
			Source:   model.SourceLocation{FilePath: path, Line: 1},
		})
	}

	return rels
}

func scanPII(path string, text string) []model.Diagnostic {
	var diags []model.Diagnostic

	// Check emails
	if matches := emailRegex.FindAllStringIndex(text, -1); len(matches) > 0 {
		for _, match := range matches {
			line := countLinesToOffset(text, match[0])
			diags = append(diags, model.Diagnostic{
				Code:        model.CodePrivacyProhibitedPII,
				Severity:    model.SeverityFatal,
				Message:     "contains email address [REDACTED]",
				Source:      model.SourceLocation{FilePath: path, Line: line},
				Remediation: "Remove any email addresses from CKB files committed to the repository.",
			})
		}
	}

	// Check phone numbers
	if matches := phoneRegex.FindAllStringIndex(text, -1); len(matches) > 0 {
		for _, match := range matches {
			line := countLinesToOffset(text, match[0])
			diags = append(diags, model.Diagnostic{
				Code:        model.CodePrivacyProhibitedPII,
				Severity:    model.SeverityFatal,
				Message:     "contains phone number [REDACTED]",
				Source:      model.SourceLocation{FilePath: path, Line: line},
				Remediation: "Remove any phone numbers from CKB files committed to the repository.",
			})
		}
	}

	return diags
}

func countLinesToOffset(text string, offset int) int {
	line := 1
	for idx, r := range text {
		if idx >= offset {
			break
		}
		if r == '\n' {
			line++
		}
	}
	return line
}

func sortDiagnostics(diags []model.Diagnostic) {
	sorter := model.DiagnosticsSorter(diags)
	// Sort using standard sort block
	sortStable(sorter)
}

func sortStable(d model.DiagnosticsSorter) {
	// Standard bubble sort variant for stability and simplicity
	n := d.Len()
	for i := 0; i < n; i++ {
		for j := 0; j < n-i-1; j++ {
			if d.Less(j+1, j) {
				d.Swap(j, j+1)
			}
		}
	}
}

func registerEvidenceCatalogIDs(kb *model.KnowledgeBase, obj *model.Object, source []byte) {
	re := regexp.MustCompile(`\*\*(ev:[a-z0-9-]+)\*\*`)
	matches := re.FindAllSubmatch(source, -1)
	for _, m := range matches {
		if len(m) > 1 {
			subID := string(m[1])
			if _, exists := kb.Objects[subID]; !exists {
				kb.Objects[subID] = &model.Object{
					ID:         subID,
					Type:       model.TypeEvidence,
					SourceFile: obj.SourceFile,
					Metadata: model.Metadata{
						SchemaVersion: "1.0",
						ID:            subID,
						Type:          model.TypeEvidence,
						Status:        model.StatusActive,
						Verification:  model.VerificationIndependentlyVerified,
						Confidence:    1.0,
						Visibility:    model.VisibilityPublic,
						LastUpdated:   time.Now(),
						Lifecycle:     model.LifecycleActive,
					},
				}
			}
		}
	}
}
