package validation

import (
	"fmt"

	"github.com/admbahm/theForge/ckb/model"
)

// ValidateGraph checks global constraints, duplicate IDs, broken references, and orphaned evidence.
func ValidateGraph(kb *model.KnowledgeBase) []model.Diagnostic {
	var diags []model.Diagnostic

	// 1. Check duplicate IDs (normally caught during parsing, but check maps just in case)
	// 2. Validate relationships
	referencedIDs := make(map[string]bool)

	for _, obj := range kb.Objects {
		seenEdges := make(map[string]bool)
		for _, rel := range obj.Relationships {
			// Track target reference
			referencedIDs[rel.TargetID] = true

			// Check duplicate edge
			edgeKey := string(rel.Type) + ":" + rel.TargetID
			if seenEdges[edgeKey] {
				diags = append(diags, model.Diagnostic{
					Code:     model.CodeGraphDuplicateEdge,
					Severity: model.SeverityError,
					Message:  fmt.Sprintf("contains duplicate relationship reference to %q under %q", rel.TargetID, rel.Type),
					Source:   rel.Source,
					ObjectID: obj.ID,
					Field:    string(rel.Type),
				})
				continue
			}
			seenEdges[edgeKey] = true

			// Check self reference
			if rel.TargetID == obj.ID {
				diags = append(diags, model.Diagnostic{
					Code:     model.CodeGraphInvalidSelfRef,
					Severity: model.SeverityError,
					Message:  fmt.Sprintf("Self-reference detected: Object %q links to itself under %q", obj.ID, rel.Type),
					Source:   rel.Source,
					ObjectID: obj.ID,
					Field:    string(rel.Type),
				})
				continue
			}

			// Check broken reference (target ID does not exist in graph)
			targetNode, ok := kb.Objects[rel.TargetID]
			if !ok {
				diags = append(diags, model.Diagnostic{
					Code:     model.CodeGraphBrokenReference,
					Severity: model.SeverityError,
					Message:  fmt.Sprintf("Broken Reference: ID %q linked under %q does not exist in the vault", rel.TargetID, rel.Type),
					Source:   rel.Source,
					ObjectID: obj.ID,
					Field:    string(rel.Type),
				})
				continue
			}

			// Validate relationship compatibility against the resolved target object's declared type.
			switch rel.Type {
			case model.RelExperience:
				if targetNode.Type != model.TypeExperience {
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeRelationshipInvalidType,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("Invalid Link target: Related Experience %q must resolve to Type %q, got Type %q", rel.TargetID, model.TypeExperience, targetNode.Type),
						Source:   rel.Source,
						ObjectID: obj.ID,
						Field:    string(rel.Type),
					})
				}
			case model.RelProjects:
				if targetNode.Type != model.TypeProject {
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeRelationshipInvalidType,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("Invalid Link target: Related Projects %q must resolve to Type %q, got Type %q", rel.TargetID, model.TypeProject, targetNode.Type),
						Source:   rel.Source,
						ObjectID: obj.ID,
						Field:    string(rel.Type),
					})
				}
			case model.RelEvidence:
				if targetNode.Type != model.TypeEvidence {
					diags = append(diags, model.Diagnostic{
						Code:     model.CodeRelationshipInvalidType,
						Severity: model.SeverityError,
						Message:  fmt.Sprintf("Invalid Link target: Related Evidence %q must resolve to Type %q, got Type %q", rel.TargetID, model.TypeEvidence, targetNode.Type),
						Source:   rel.Source,
						ObjectID: obj.ID,
						Field:    string(rel.Type),
					})
				}
			}

			// Check if target is deleted/superseded
			if targetNode.Metadata.Verification == model.VerificationSuperseded {
				diags = append(diags, model.Diagnostic{
					Code:     model.CodeGraphBrokenReference,
					Severity: model.SeverityWarning,
					Message:  fmt.Sprintf("Reference to superseded object: ID %q is marked as Superseded", rel.TargetID),
					Source:   rel.Source,
					ObjectID: obj.ID,
					Field:    string(rel.Type),
				})
			}
		}
	}

	// 3. Orphaned evidence check
	// Every evidence node (prefix "ev:") must be referenced by at least one other node
	for id, obj := range kb.Objects {
		if obj.Type == model.TypeEvidence && id != "ev:main" && id != "ev:fictional-main" {
			if !referencedIDs[id] {
				diags = append(diags, model.Diagnostic{
					Code:     model.CodeEvidenceOrphaned,
					Severity: model.SeverityWarning,
					Message:  fmt.Sprintf("Evidence ID %q is never referenced by any experience, project, credential, or contribution node", id),
					Source:   model.SourceLocation{FilePath: obj.SourceFile, Line: 1},
					ObjectID: id,
				})
			}
		}
	}

	return diags
}
