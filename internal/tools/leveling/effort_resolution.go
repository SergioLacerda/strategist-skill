package leveling

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// EffortResolutionArtifact projects a resolved Level into the versioned
// cross-boundary artifact. The existing Level remains the compatibility shape
// used by labels and ledger records.
func (l Level) EffortResolutionArtifact() (domain.EffortResolutionArtifact, error) {
	artifact := domain.EffortResolutionArtifact{
		SchemaVersion:   domain.EffortResolutionArtifactSchemaVersion,
		TaxonomyVersion: domain.CanonicalTaxonomyVersion,
		Role:            l.Role,
		Provider:        l.Provider,
		Model:           l.Model,
		Effort:          l.Effort,
		Source:          l.Source,
		Capability:      l.Capability,
		FallbackUsed:    l.FallbackUsed,
		FallbackReason:  l.FallbackReason,
		PolicyVersion:   l.PolicyVersion,
		PolicyDigest:    l.PolicyDigest,
	}
	if err := artifact.Validate(); err != nil {
		return domain.EffortResolutionArtifact{}, fmt.Errorf("leveling effort resolution: %w", err)
	}
	return artifact, nil
}

// ResolveLevelArtifact resolves a Level and returns only its versioned
// artifact projection for consumers that do not need the legacy Level shape.
func ResolveLevelArtifact(policy Policy, provider, role string, signals Signals, host Host) (domain.EffortResolutionArtifact, error) {
	level, err := ResolveLevel(policy, provider, role, signals, host)
	if err != nil {
		return domain.EffortResolutionArtifact{}, err
	}
	return level.EffortResolutionArtifact()
}
