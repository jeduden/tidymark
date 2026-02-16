package concisenessscoring

import _ "embed"

var (
	//go:embed models/cue-linear-v1.json
	embeddedModelArtifact []byte

	//go:embed models/manifest.json
	embeddedModelManifest []byte
)
