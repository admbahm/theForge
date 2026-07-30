package model

import (
	"path/filepath"
	"sort"
	"strings"
)

// DiagnosticCode represents a stable public error identifier.
type DiagnosticCode string

const (
	CodeMetadataMissingID          DiagnosticCode = "CKB-METADATA-MISSING-ID"
	CodeIdentityInvalidID          DiagnosticCode = "CKB-IDENTITY-INVALID-ID"
	CodeIdentityPrefixTypeMismatch DiagnosticCode = "CKB-IDENTITY-PREFIX-TYPE-MISMATCH"
	CodeIdentityDuplicateID        DiagnosticCode = "CKB-IDENTITY-DUPLICATE-ID"
	CodeMetadataInvalidObjType     DiagnosticCode = "CKB-METADATA-INVALID-OBJECT-TYPE"
	CodeMetadataMissingField       DiagnosticCode = "CKB-METADATA-MISSING-FIELD"
	CodeMetadataInvalidOrder       DiagnosticCode = "CKB-METADATA-INVALID-ORDER"
	CodeMetadataDuplicateField     DiagnosticCode = "CKB-METADATA-DUPLICATE-FIELD"
	CodeMetadataUnknownField       DiagnosticCode = "CKB-METADATA-UNKNOWN-FIELD"
	CodeMetadataInvalidEnum        DiagnosticCode = "CKB-METADATA-INVALID-ENUM"
	CodeMetadataInvalidDate        DiagnosticCode = "CKB-METADATA-INVALID-DATE"
	CodeMetadataInvalidConfidence  DiagnosticCode = "CKB-METADATA-INVALID-CONFIDENCE"
	CodeGraphBrokenReference       DiagnosticCode = "CKB-GRAPH-BROKEN-REFERENCE"
	CodeRelationshipInvalidType    DiagnosticCode = "CKB-RELATIONSHIP-INVALID-TYPE"
	CodeGraphDuplicateEdge         DiagnosticCode = "CKB-GRAPH-DUPLICATE-EDGE"
	CodeGraphInvalidSelfRef        DiagnosticCode = "CKB-GRAPH-INVALID-SELF-REFERENCE"
	CodeEvidenceOrphaned           DiagnosticCode = "CKB-EVIDENCE-ORPHANED"
	CodeEvidenceUnresolved         DiagnosticCode = "CKB-EVIDENCE-UNRESOLVED"
	CodeVersionUnsupported         DiagnosticCode = "CKB-VERSION-UNSUPPORTED"
	CodePrivacyProhibitedPII       DiagnosticCode = "CKB-PRIVACY-PROHIBITED-PII"
	CodeLimitsExceeded             DiagnosticCode = "CKB-LIMITS-EXCEEDED"
	CodeLimitRelationshipsExceeded DiagnosticCode = "CKB-LIMIT-RELATIONSHIPS-EXCEEDED"
	CodeStructureMalformed         DiagnosticCode = "CKB-STRUCTURE-MALFORMED"
	CodeMarkdownRawHTML            DiagnosticCode = "CKB-MARKDOWN-RAW-HTML"
	CodeMetadataInvalidColumnCount DiagnosticCode = "CKB-METADATA-INVALID-COLUMN-COUNT"
	CodeMetadataInvalidHeader      DiagnosticCode = "CKB-METADATA-INVALID-HEADER"
	CodeMetadataMalformedRow       DiagnosticCode = "CKB-METADATA-MALFORMED-ROW"
	CodeSectionMissingRequired     DiagnosticCode = "CKB-SECTION-MISSING-REQUIRED"
)

// DiagnosticSeverity defines the severity levels of diagnostic output.
type DiagnosticSeverity string

const (
	SeverityFatal   DiagnosticSeverity = "Fatal"
	SeverityError   DiagnosticSeverity = "Error"
	SeverityWarning DiagnosticSeverity = "Warning"
	SeverityInfo    DiagnosticSeverity = "Info"
)

// SourceLocation defines a position within a CKB source file.
type SourceLocation struct {
	FilePath string `json:"file_path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

func (s SourceLocation) Less(other SourceLocation) bool {
	p1 := filepath.ToSlash(s.FilePath)
	p2 := filepath.ToSlash(other.FilePath)
	if p1 != p2 {
		return p1 < p2
	}
	if s.Line != other.Line {
		return s.Line < other.Line
	}
	return s.Column < other.Column
}

// Diagnostic represents a structured parser error or warning.
type Diagnostic struct {
	Code          DiagnosticCode     `json:"code"`
	Severity      DiagnosticSeverity `json:"severity"`
	Message       string             `json:"message"`
	Source        SourceLocation     `json:"source"`
	ObjectID      string             `json:"object_id,omitempty"`
	Field         string             `json:"field,omitempty"`
	Section       string             `json:"section,omitempty"`
	RelatedSource *SourceLocation    `json:"related_source,omitempty"`
	Remediation   string             `json:"remediation,omitempty"`
}

// HasBlockingDiagnostics reports whether a compilation stage emitted an error
// or fatal diagnostic. Candidate-facing artifacts must not be published when
// this returns true, even if the stage also returned a non-nil result.
func HasBlockingDiagnostics(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SeverityError || diagnostic.Severity == SeverityFatal {
			return true
		}
	}
	return false
}

// BlockingDiagnosticCodes returns stable diagnostic identifiers without
// exposing private diagnostic messages or evidence bodies in operational logs.
func BlockingDiagnosticCodes(diagnostics []Diagnostic) []DiagnosticCode {
	seen := make(map[DiagnosticCode]struct{})
	var codes []DiagnosticCode
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != SeverityError && diagnostic.Severity != SeverityFatal {
			continue
		}
		if _, exists := seen[diagnostic.Code]; exists {
			continue
		}
		seen[diagnostic.Code] = struct{}{}
		codes = append(codes, diagnostic.Code)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}

// DiagnosticsSorter implements sort.Interface for deterministic diagnostic sorting.
type DiagnosticsSorter []Diagnostic

func (d DiagnosticsSorter) Len() int      { return len(d) }
func (d DiagnosticsSorter) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d DiagnosticsSorter) Less(i, j int) bool {
	// 1. Sort by relative path & line & column
	s1 := d[i].Source
	s2 := d[j].Source
	p1 := strings.ToLower(filepath.ToSlash(s1.FilePath))
	p2 := strings.ToLower(filepath.ToSlash(s2.FilePath))
	if p1 != p2 {
		return p1 < p2
	}
	if s1.Line != s2.Line {
		return s1.Line < s2.Line
	}
	if s1.Column != s2.Column {
		return s1.Column < s2.Column
	}

	// 2. Sort by severity (Fatal > Error > Warning > Info)
	sevOrder := map[DiagnosticSeverity]int{
		SeverityFatal:   4,
		SeverityError:   3,
		SeverityWarning: 2,
		SeverityInfo:    1,
	}
	if sevOrder[d[i].Severity] != sevOrder[d[j].Severity] {
		return sevOrder[d[i].Severity] > sevOrder[d[j].Severity] // higher severity first
	}

	// 3. Sort by code
	if d[i].Code != d[j].Code {
		return d[i].Code < d[j].Code
	}

	// 4. Sort by ObjectID
	return d[i].ObjectID < d[j].ObjectID
}
