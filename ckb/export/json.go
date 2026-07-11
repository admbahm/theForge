package export

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/admbahm/theForge/ckb/model"
)

// ExportedModel represents the canonical sorted JSON output.
type ExportedModel struct {
	SchemaVersion string             `json:"schema_version"`
	Objects       []*model.Object    `json:"objects"`
	Diagnostics   []model.Diagnostic `json:"diagnostics,omitempty"`
}

// ExportJSON writes a deterministic JSON representation of the parsed KB.
func ExportJSON(kb *model.KnowledgeBase, diagnostics []model.Diagnostic, w io.Writer) error {
	// 1. Sort objects alphabetically by ID
	sortedObjects := make([]*model.Object, 0, len(kb.Objects))
	for _, obj := range kb.Objects {
		// Create a deep copy of the object to prevent mutation of in-memory graph
		copiedObj := &model.Object{
			ID:         obj.ID,
			Type:       obj.Type,
			SourceFile: obj.SourceFile,
			Metadata:   obj.Metadata,
			Sections:   make([]model.Section, len(obj.Sections)),
		}

		copy(copiedObj.Sections, obj.Sections)

		// Sort metadata lists
		copiedObj.Metadata.RelatedDocs = copyAndSort(obj.Metadata.RelatedDocs)
		copiedObj.Metadata.RelatedExps = copyAndSort(obj.Metadata.RelatedExps)
		copiedObj.Metadata.RelatedProjs = copyAndSort(obj.Metadata.RelatedProjs)
		copiedObj.Metadata.RelatedEvs = copyAndSort(obj.Metadata.RelatedEvs)
		copiedObj.Metadata.Tags = copyAndSort(obj.Metadata.Tags)

		// Sort relationships alphabetically by target ID
		copiedObj.Relationships = make([]model.Relationship, len(obj.Relationships))
		copy(copiedObj.Relationships, obj.Relationships)
		sort.Slice(copiedObj.Relationships, func(i, j int) bool {
			if copiedObj.Relationships[i].TargetID != copiedObj.Relationships[j].TargetID {
				return copiedObj.Relationships[i].TargetID < copiedObj.Relationships[j].TargetID
			}
			return copiedObj.Relationships[i].Type < copiedObj.Relationships[j].Type
		})

		sortedObjects = append(sortedObjects, copiedObj)
	}

	sort.Slice(sortedObjects, func(i, j int) bool {
		return sortedObjects[i].ID < sortedObjects[j].ID
	})

	// 2. Sort diagnostics deterministically
	sortedDiags := make([]model.Diagnostic, len(diagnostics))
	copy(sortedDiags, diagnostics)
	sort.Slice(sortedDiags, func(i, j int) bool {
		return model.DiagnosticsSorter(sortedDiags).Less(i, j)
	})

	modelExport := ExportedModel{
		SchemaVersion: "1.0",
		Objects:       sortedObjects,
		Diagnostics:   sortedDiags,
	}

	// 3. Encode with indentation
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(modelExport)
}

func copyAndSort(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	sort.Strings(dst)
	return dst
}
