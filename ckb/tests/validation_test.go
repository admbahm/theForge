package tests

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/admbahm/theForge/ckb/export"
	"github.com/admbahm/theForge/ckb/model"
	"github.com/admbahm/theForge/ckb/parser"
)

func TestValidFixtures(t *testing.T) {
	rootPath := filepath.Join("..", "tests", "fixtures", "valid")

	rootFiles, err := parser.DiscoverFiles(filepath.Join(".."))
	if err != nil {
		t.Fatalf("failed to discover root CKB files: %v", err)
	}

	t.Logf("Found %d root files", len(rootFiles))
	for _, rf := range rootFiles {
		t.Logf("  Root file: %s", rf)
	}

	files := append(rootFiles,
		filepath.Join(rootPath, "minimal_valid.md"),
		filepath.Join(rootPath, "complete_experience.md"),
		filepath.Join(rootPath, "complete_project.md"),
		filepath.Join(rootPath, "evidence_graph.md"),
		filepath.Join(rootPath, "private_evidence.md"),
		filepath.Join(rootPath, "many_to_many.md"),
	)

	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	}

	res := parser.ParseFiles(context.Background(), files, opts)

	// Assert zero error/fatal diagnostics
	for _, diag := range res.Diagnostics {
		if diag.Severity == model.SeverityFatal || diag.Severity == model.SeverityError {
			t.Errorf("Unexpected error diagnostic in valid fixtures: %s (%s)", diag.Message, diag.Code)
		}
	}

	kb := res.KnowledgeBase

	// Assert expected object count (6 files = 7 IDs, since evidence_graph declares ev:git-fixture-repo and ev:eval-fixture-2025 sub-ids, wait, pre-scan was run in previous test, but here it's parsed directly)
	// Let's check objects map
	expectedIDs := []string{
		"profile:minimal-profile",
		"exp:fictional-staff",
		"proj:fictional-titan",
		"ev:fictional-main",
		"proj:private-spec",
		"acc:many-links",
	}

	for _, id := range expectedIDs {
		if _, ok := kb.Objects[id]; !ok {
			t.Errorf("Expected object ID %q was not found in parsed knowledge base", id)
		}
	}

	// Verify complete_experience object metadata and fields
	expObj := kb.Objects["exp:fictional-staff"]
	if expObj != nil {
		if expObj.Type != model.TypeExperience {
			t.Errorf("Expected exp:fictional-staff type to be Experience, got %s", expObj.Type)
		}
		if expObj.Metadata.Confidence != 0.95 {
			t.Errorf("Expected confidence 0.95, got %.2f", expObj.Metadata.Confidence)
		}
		if expObj.Metadata.Verification != model.VerificationIndependentlyVerified {
			t.Errorf("Expected verification Independently-Verified, got %s", expObj.Metadata.Verification)
		}
	}

	// Verify duplicate JSON export determinism (running twice produces byte-identical output)
	var w1, w2 bytes.Buffer
	err1 := export.ExportJSON(kb, res.Diagnostics, &w1)
	err2 := export.ExportJSON(kb, res.Diagnostics, &w2)

	if err1 != nil || err2 != nil {
		t.Fatalf("JSON export failed: %v / %v", err1, err2)
	}

	if !bytes.Equal(w1.Bytes(), w2.Bytes()) {
		t.Error("JSON export determinism failure: successive runs produced different bytes")
	}
}

func TestInvalidFixtures(t *testing.T) {
	rootPath := filepath.Join("..", "tests", "fixtures", "invalid")

	// Mapping invalid fixtures to expected Diagnostic Code and properties
	type expectDiag struct {
		Code     model.DiagnosticCode
		Severity model.DiagnosticSeverity
		Field    string
	}

	expectedDiagnostics := map[string]expectDiag{
		"missing_id.md":                 {model.CodeMetadataInvalidOrder, model.SeverityError, "Type"},
		"malformed_id.md":               {model.CodeIdentityInvalidID, model.SeverityFatal, "ID"},
		"duplicate_id.md":               {model.CodeIdentityDuplicateID, model.SeverityFatal, ""},
		"unknown_type.md":               {model.CodeMetadataInvalidObjType, model.SeverityFatal, "Type"},
		"missing_required_field.md":     {model.CodeMetadataInvalidOrder, model.SeverityError, "Confidence"},
		"invalid_field_order.md":        {model.CodeMetadataInvalidOrder, model.SeverityError, "Verification Level"},
		"duplicate_metadata_row.md":     {model.CodeMetadataDuplicateField, model.SeverityFatal, ""},
		"unexpected_metadata_field.md":  {model.CodeMetadataUnknownField, model.SeverityError, "Superfluous Key"},
		"invalid_enum.md":               {model.CodeMetadataInvalidEnum, model.SeverityError, "Status"},
		"malformed_date.md":             {model.CodeMetadataInvalidDate, model.SeverityError, "Last Updated"},
		"malformed_float.md":            {model.CodeMetadataInvalidConfidence, model.SeverityError, "Confidence"},
		"broken_relationship.md":        {model.CodeGraphBrokenReference, model.SeverityError, "Related Projects"},
		"invalid_relationship_type.md":  {model.CodeRelationshipInvalidType, model.SeverityError, "Related Projects"},
		"duplicate_edge.md":             {model.CodeGraphDuplicateEdge, model.SeverityError, "Related Projects"},
		"self_reference.md":             {model.CodeGraphInvalidSelfRef, model.SeverityError, "Related Experience"},
		"orphan_evidence.md":            {model.CodeEvidenceOrphaned, model.SeverityWarning, ""},
		"unsupported_schema_version.md": {model.CodeVersionUnsupported, model.SeverityError, "Schema Version"},
		"prohibited_pii.md":             {model.CodePrivacyProhibitedPII, model.SeverityFatal, ""},
	}

	rootFiles, err := parser.DiscoverFiles(filepath.Join(".."))
	if err != nil {
		t.Fatalf("failed to discover root CKB files: %v", err)
	}

	for file, expected := range expectedDiagnostics {
		filePath := filepath.Join(rootPath, file)

		// Parse individual invalid file alongside mock helper nodes to resolve relations
		files := append([]string{
			filePath,
			filepath.Join("..", "tests", "fixtures", "valid", "complete_project.md"), // proj:fictional-titan
			filepath.Join("..", "tests", "fixtures", "valid", "evidence_graph.md"),   // ev:fictional-main
		}, rootFiles...)

		opts := parser.ParseOptions{
			Strict:          true,
			ValidatePrivacy: true,
			Limits:          parser.DefaultLimits(),
		}

		res := parser.ParseFiles(context.Background(), files, opts)

		// Assert expected diagnostic code and severity
		found := false
		for _, diag := range res.Diagnostics {
			// Normalize filepath comparisons
			if diag.Code == expected.Code {
				found = true
				if diag.Severity != expected.Severity {
					t.Errorf("File %s: expected severity %s, got %s", file, expected.Severity, diag.Severity)
				}
				if expected.Field != "" && diag.Field != expected.Field {
					t.Errorf("File %s: expected field %q, got %q", file, expected.Field, diag.Field)
				}
				if !strings.Contains(filepath.ToSlash(diag.Source.FilePath), file) {
					t.Errorf("File %s: diagnostic source path %s does not contain filename", file, diag.Source.FilePath)
				}
				// Assert complete english wording does not break parser checks
				if len(diag.Message) == 0 {
					t.Errorf("File %s: expected descriptive message string, got empty message", file)
				}
				break
			}
		}

		if !found {
			t.Errorf("File %s: expected to fail with Diagnostic Code %s, but code was not generated.\nDiagnostics generated: %+v", file, expected.Code, res.Diagnostics)
		}
	}
}

func TestAdditionalParserConstraints(t *testing.T) {
	// 1. Context Cancellation Check
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	files := []string{filepath.Join("..", "tests", "fixtures", "valid", "minimal_valid.md")}
	opts := parser.ParseOptions{Strict: true, Limits: parser.DefaultLimits()}
	res := parser.ParseFiles(ctx, files, opts)

	cancelled := false
	for _, diag := range res.Diagnostics {
		if diag.Code == model.CodeStructureMalformed && strings.Contains(diag.Message, "cancelled") {
			cancelled = true
		}
	}
	if !cancelled {
		t.Error("Expected parsing cancellation diagnostic when context is cancelled")
	}

	// 2. Parser Sizing Limits Enforcements
	restrictLimits := parser.Limits{
		MaxFileSize:          100, // extremely small size limit
		MaxLineLength:        50,
		MaxMetadataRows:      5,
		MaxObjectCount:       10,
		MaxRelationshipsNode: 5,
		MaxSectionCount:      5,
		MaxHeadingDepth:      3,
	}

	resRestrict := parser.ParseFiles(context.Background(), files, parser.ParseOptions{Limits: restrictLimits})
	limitExceeded := false
	for _, diag := range resRestrict.Diagnostics {
		if diag.Code == model.CodeLimitsExceeded {
			limitExceeded = true
		}
	}
	if !limitExceeded {
		t.Error("Expected CodeLimitsExceeded when file size exceeds MaxFileSize restriction")
	}

	// 3. PII safe redaction check
	piiFile := filepath.Join("..", "tests", "fixtures", "invalid", "prohibited_pii.md")
	resPII := parser.ParseFiles(context.Background(), []string{piiFile}, opts)
	for _, diag := range resPII.Diagnostics {
		if diag.Code == model.CodePrivacyProhibitedPII {
			if strings.Contains(diag.Message, "adam@fictional.com") {
				t.Error("Privacy Leak: complete prohibited email value leaked in diagnostic message")
			}
			if !strings.Contains(diag.Message, "[REDACTED]") {
				t.Error("Expected redacted notation inside privacy diagnostics report")
			}
		}
	}
}

func TestParseDirectory(t *testing.T) {
	opts := parser.ParseOptions{
		Strict:          true,
		ValidatePrivacy: true,
		Limits:          parser.DefaultLimits(),
	}
	res := parser.ParseDirectory(context.Background(), "..", opts)
	if res.KnowledgeBase == nil {
		t.Fatal("Expected non-nil KnowledgeBase from ParseDirectory")
	}
	// Verify that it successfully discovered and parsed the root files
	if len(res.KnowledgeBase.Objects) == 0 {
		t.Error("Expected parsed objects in ParseDirectory, got 0")
	}
}
