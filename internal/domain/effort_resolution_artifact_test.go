package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffortResolutionArtifactValidatesVersionedProvenance(t *testing.T) {
	artifact := EffortResolutionArtifact{
		SchemaVersion: EffortResolutionArtifactSchemaVersion,
		Role:          "ranger", Provider: "CODEX", Model: "reasoning", Effort: "high",
		Source: "policy", Capability: "reasoning", PolicyVersion: 1, PolicyDigest: "sha256:policy",
	}
	require.NoError(t, artifact.Validate())
}

func TestEffortResolutionArtifactRejectsUnknownTaxonomyVersion(t *testing.T) {
	t.Parallel()

	artifact := EffortResolutionArtifact{
		SchemaVersion: EffortResolutionArtifactSchemaVersion, TaxonomyVersion: "strategist-taxonomy/v0", Role: "ranger",
	}
	assert.ErrorContains(t, artifact.Validate(), "unsupported taxonomy version")
}

func TestEffortResolutionArtifactAllowsUnresolvedObservation(t *testing.T) {
	artifact := EffortResolutionArtifact{SchemaVersion: EffortResolutionArtifactSchemaVersion, Role: "ranger"}
	require.NoError(t, artifact.Validate())
}

func TestEffortResolutionArtifactRejectsUnverifiableFallback(t *testing.T) {
	artifact := EffortResolutionArtifact{
		SchemaVersion: EffortResolutionArtifactSchemaVersion, Role: "ranger",
		FallbackUsed: true,
	}
	err := artifact.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fallback_reason")
}

func TestEffortResolutionArtifactRejectsPolicyWithoutDigest(t *testing.T) {
	artifact := EffortResolutionArtifact{
		SchemaVersion: EffortResolutionArtifactSchemaVersion, Role: "ranger", PolicyVersion: 1,
	}
	err := artifact.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "policy_digest")
}
