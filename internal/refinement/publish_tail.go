package refinement

import (
	"fmt"
	"time"
)

// publishTailInput is what every Archivist publisher hands the shared
// publishing tail: the validated canonical contents plus the two recorders.
// RecordPublication may be nil for library-only callers.
type publishTailInput struct {
	MissionID, ProviderChangeID string
	Refined, Pending            string
	Contents                    map[string][]byte
	RecordConfidence            func() error
	RecordPublication           func(PackagePublication) error
}

// publishTail records the Archivist confidence claim before anything is
// written, publishes (or promotes an identical existing) package atomically
// and records the publication. It is the part of publication shared by the
// OpenSpec bridge and the embedded Archivist publisher (ADR-0067).
func publishTail(t publishTailInput) error {
	if err := t.RecordConfidence(); err != nil {
		return fmt.Errorf("openspec bridge: record Archivist confidence: %w", err)
	}
	published, err := publishOrPromote(t.Refined, t.Pending, t.Contents)
	if err != nil {
		return err
	}
	return recordPublication(t, published)
}

func recordPublication(t publishTailInput, published bool) error {
	if !published || t.RecordPublication == nil {
		return nil
	}
	publication := PackagePublication{
		MissionID: t.MissionID, ProviderChangeID: t.ProviderChangeID,
		SourceDigest: digest(t.Contents["analysis.md"]), PackageDigest: canonicalPackageDigest(t.Contents), PublishedAt: time.Now().UTC(),
	}
	if err := t.RecordPublication(publication); err != nil {
		return fmt.Errorf("openspec bridge: record package publication: %w", err)
	}
	return nil
}
