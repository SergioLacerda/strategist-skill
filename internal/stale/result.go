package stale

// Reason is a stable machine-readable stale-check reason.
type Reason string

const (
	// ReasonFresh means the artifact is usable. This is a "no drift detected"
	// result covering only the byte (manifest SHA256) and provenance (source
	// mtime/size lineage) drift classes — see docs/drift-detection-matrix.md
	// for what strategist check-stale does and does not check (it says
	// nothing about schema, contract, behavior, or semantic drift).
	ReasonFresh Reason = "fresh"
	// ReasonMissingArtifact means the target artifact does not exist.
	ReasonMissingArtifact Reason = "missing_artifact"
	// ReasonMissingManifest means the sibling .manifest.gz does not exist.
	ReasonMissingManifest Reason = "missing_manifest"
	// ReasonManifestEntryMissing means the manifest has no checksum for the artifact.
	ReasonManifestEntryMissing Reason = "manifest_entry_missing"
	// ReasonArtifactHashMismatch means the artifact checksum differs from the manifest.
	ReasonArtifactHashMismatch Reason = "artifact_hash_mismatch"
	// ReasonMissingSource means a source recorded in the artifact no longer exists.
	ReasonMissingSource Reason = "missing_source"
	// ReasonSourceNewer means a source mtime is newer than the compiled metadata.
	ReasonSourceNewer Reason = "source_newer"
	// ReasonSourceMetadataMismatch means strong source metadata differs from disk.
	ReasonSourceMetadataMismatch Reason = "source_metadata_mismatch"
)

// Result is the structured outcome of a stale check.
type Result struct {
	Stale        bool   `json:"stale"`
	Reason       Reason `json:"reason"`
	ArtifactPath string `json:"artifact_path"`
	SourcePath   string `json:"source_path,omitempty"`
	Detail       string `json:"detail,omitempty"`
}

// SourceMetadata records the source metadata written into compiled artifacts.
type SourceMetadata struct {
	MTime   int64  `json:"mtime"`
	MTimeNS int64  `json:"mtime_ns"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256,omitempty"`
}
