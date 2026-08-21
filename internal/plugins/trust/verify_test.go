package trust_test

import (
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/trust"
	"github.com/stretchr/testify/assert"
)

const trustDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestVerifyTrustPassesWhenPolicyExpectationsMatch(t *testing.T) {
	t.Parallel()

	result := trust.Verify(trust.Subject{
		Package: packageFixture(),
		Evidence: trust.Evidence{
			SourceURI:    "embedded://skills/brainstorming",
			Signatures:   []string{"sigstore/openai"},
			Attestations: []string{"slsa-provenance/v1"},
			Provenance: trust.Provenance{
				BuilderID:  "builder://strategist",
				SourceURI:  "embedded://skills/brainstorming",
				BuildType:  "local-embed",
				VerifiedAt: "2026-08-20T00:00:00Z",
			},
		},
	}, policyFixture(), mustTime(t, "2026-08-21T00:00:00Z"))

	assert.True(t, result.Trusted)
	assert.Empty(t, result.Reasons)
	assert.Equal(t, "policy-1", result.PolicyRevision)
	assert.Equal(t, trustDigest, result.PackageDigest)
}

func TestVerifyTrustBlocksPolicyMismatches(t *testing.T) {
	t.Parallel()

	policy := policyFixture()
	policy.RevokedDigests = []string{trustDigest}
	result := trust.Verify(trust.Subject{
		Package: packageFixture(),
		Evidence: trust.Evidence{
			SourceURI:    "file:///tmp/untrusted",
			Signatures:   []string{"sigstore/other"},
			Attestations: []string{"unknown-attestation"},
			Provenance: trust.Provenance{
				BuilderID: "builder://other",
				SourceURI: "file:///tmp/untrusted",
				BuildType: "manual",
			},
		},
	}, policy, mustTime(t, "2026-09-25T00:00:00Z"))

	assert.False(t, result.Trusted)
	assert.Contains(t, result.Reasons, trust.Reason{Code: "source_not_trusted", Detail: "file:///tmp/untrusted"})
	assert.Contains(t, result.Reasons, trust.Reason{Code: "missing_required_signature", Detail: "sigstore/openai"})
	assert.Contains(t, result.Reasons, trust.Reason{Code: "missing_required_attestation", Detail: "slsa-provenance/v1"})
	assert.Contains(t, result.Reasons, trust.Reason{Code: "provenance_builder_mismatch", Detail: "builder://other"})
	assert.Contains(t, result.Reasons, trust.Reason{Code: "digest_revoked", Detail: trustDigest})
	assert.Contains(t, result.Reasons, trust.Reason{Code: "freshness_expired", Detail: "2026-08-20T00:00:00Z"})
}

func TestVerifyTrustBlocksLicenseMismatch(t *testing.T) {
	t.Parallel()

	pkg := packageFixture()
	pkg.License = "GPL-3.0"
	result := trust.Verify(trust.Subject{Package: pkg}, policyFixture(), mustTime(t, "2026-08-21T00:00:00Z"))

	assert.False(t, result.Trusted)
	assert.Contains(t, result.Reasons, trust.Reason{Code: "license_not_allowed", Detail: "GPL-3.0"})
}

func TestVerifyTrustBlocksDeprecatedDigest(t *testing.T) {
	t.Parallel()

	policy := policyFixture()
	policy.Deprecations = []domain.TrustDeprecation{{
		PackageDigest:     trustDigest,
		ReasonCode:        "upstream_eol",
		ReplacementDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}}

	result := trust.Verify(trust.Subject{Package: packageFixture()}, policy, mustTime(t, "2026-08-21T00:00:00Z"))

	assert.False(t, result.Trusted)
	assert.Contains(t, result.Reasons, trust.Reason{Code: "digest_deprecated", Detail: "upstream_eol"})
}

func TestVerifyTrustAllowsScopedDevelopmentException(t *testing.T) {
	t.Parallel()

	policy := policyFixture()
	policy.DevelopmentExceptions = []domain.TrustDevelopmentException{{
		ID:            "dev-local",
		PackageDigest: trustDigest,
		Reason:        "local adapter development",
		ExpiresAt:     "2026-08-22T00:00:00Z",
	}}

	result := trust.Verify(trust.Subject{Package: packageFixture()}, policy, mustTime(t, "2026-08-21T00:00:00Z"))

	assert.True(t, result.Trusted)
	assert.True(t, result.DevelopmentException)
	assert.Equal(t, "dev-local", result.DevelopmentExceptionID)
}

func TestVerifyTrustBlocksExpiredDevelopmentException(t *testing.T) {
	t.Parallel()

	policy := policyFixture()
	policy.DevelopmentExceptions = []domain.TrustDevelopmentException{{
		ID:            "dev-local",
		PackageDigest: trustDigest,
		Reason:        "local adapter development",
		ExpiresAt:     "2026-08-20T00:00:00Z",
	}}

	result := trust.Verify(trust.Subject{Package: packageFixture()}, policy, mustTime(t, "2026-08-21T00:00:00Z"))

	assert.False(t, result.Trusted)
	assert.Contains(t, result.Reasons, trust.Reason{Code: "development_exception_expired", Detail: "dev-local"})
}

func packageFixture() domain.PluginPackage {
	return domain.PluginPackage{
		SchemaVersion:   "strategist-plugin-package/v1",
		ID:              "openai/brainstorming",
		Publisher:       "openai",
		Version:         "1.0.0",
		Digest:          trustDigest,
		ArtifactURI:     "embedded://skills/brainstorming",
		ArtifactSize:    1024,
		License:         "MIT",
		CreatedAt:       "2026-08-20T00:00:00Z",
		ManifestSchema:  "strategist-plugin-package/v1",
		UpstreamVersion: "1.0.0",
	}
}

func policyFixture() domain.TrustPolicy {
	return domain.TrustPolicy{
		SchemaVersion:        "strategist-plugin-trust-policy/v1",
		Revision:             "policy-1",
		TrustedPublishers:    []string{"openai"},
		TrustedSources:       []string{"embedded://"},
		AllowedLicenses:      []string{"MIT", "Apache-2.0"},
		RequiredSignatures:   []string{"sigstore/openai"},
		RequiredAttestations: []string{"slsa-provenance/v1"},
		FreshnessDays:        30,
		MinimumConformance:   "C1",
	}
}

func mustTime(t *testing.T, raw string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
