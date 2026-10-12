package refinement

// AdvisoryInput carries the canonical analysis and package identity to the
// composition root's confidence/advisory recorder.
type AdvisoryInput struct {
	MissionID     string
	PackageDigest string
	Analysis      []byte
}
