package export

import (
	"encoding/json"
	"io"

	"github.com/admbahm/theForge/ckb/rendering"
)

// ExportArtifactJSON serializes the structured intermediate artifact model and manifest to deterministic JSON format.
func ExportArtifactJSON(artifact *rendering.Artifact, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(artifact)
}
