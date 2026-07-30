package model

import (
	"reflect"
	"testing"
)

func TestBlockingDiagnostics(t *testing.T) {
	diagnostics := []Diagnostic{
		{Code: "WARN", Severity: SeverityWarning},
		{Code: "ERROR-B", Severity: SeverityError},
		{Code: "ERROR-A", Severity: SeverityFatal},
		{Code: "ERROR-B", Severity: SeverityError},
	}
	if !HasBlockingDiagnostics(diagnostics) {
		t.Fatal("HasBlockingDiagnostics() = false, want true")
	}
	want := []DiagnosticCode{"ERROR-A", "ERROR-B"}
	if got := BlockingDiagnosticCodes(diagnostics); !reflect.DeepEqual(got, want) {
		t.Fatalf("BlockingDiagnosticCodes() = %v, want %v", got, want)
	}
	if HasBlockingDiagnostics([]Diagnostic{{Code: "WARN", Severity: SeverityWarning}}) {
		t.Fatal("warning-only diagnostics must not block publication")
	}
}
