// Package trust verifies plugin packages against operator trust policy.
package trust

import (
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Subject is the package plus externally collected verification evidence.
type Subject struct {
	Package  domain.PluginPackage
	Evidence Evidence
}

// Evidence records verifier observations. It never contains secrets.
type Evidence struct {
	SourceURI    string
	Signatures   []string
	Attestations []string
	Provenance   Provenance
}

// Provenance captures expected builder/source/build metadata.
type Provenance struct {
	BuilderID  string
	SourceURI  string
	BuildType  string
	VerifiedAt string
}

// Reason is a stable trust verification reason.
type Reason struct {
	Code   string
	Detail string
}

// Result is persisted as verification evidence on an installed instance.
type Result struct {
	Trusted                bool
	PolicyRevision         string
	PackageDigest          string
	Reasons                []Reason
	DevelopmentException   bool
	DevelopmentExceptionID string
	VerifiedAt             string
}

// Verify evaluates package evidence against consumer-owned trust policy.
func Verify(subject Subject, policy domain.TrustPolicy, now time.Time) Result {
	result := Result{
		PolicyRevision: policy.Revision,
		PackageDigest:  subject.Package.Digest,
		VerifiedAt:     now.UTC().Format(time.RFC3339),
	}
	if exception, ok, expired := matchingDevelopmentException(subject.Package.Digest, policy, now); ok {
		if expired {
			result.Reasons = append(result.Reasons, Reason{Code: "development_exception_expired", Detail: exception.ID})
			return result
		}
		result.Trusted = true
		result.DevelopmentException = true
		result.DevelopmentExceptionID = exception.ID
		return result
	}

	result.Reasons = append(result.Reasons, verifyPublisher(subject.Package, policy)...)
	result.Reasons = append(result.Reasons, verifySource(subject, policy)...)
	result.Reasons = append(result.Reasons, verifySignatures(subject.Evidence, policy)...)
	result.Reasons = append(result.Reasons, verifyAttestations(subject.Evidence, policy)...)
	result.Reasons = append(result.Reasons, verifyProvenance(subject, policy)...)
	result.Reasons = append(result.Reasons, verifyFreshness(subject.Package, policy, now)...)
	result.Reasons = append(result.Reasons, verifyRevocation(subject.Package, policy)...)
	result.Reasons = append(result.Reasons, verifyDeprecation(subject.Package, policy)...)
	result.Reasons = append(result.Reasons, verifyLicense(subject.Package, policy)...)
	result.Trusted = len(result.Reasons) == 0
	return result
}

func verifyPublisher(pkg domain.PluginPackage, policy domain.TrustPolicy) []Reason {
	publisher := pkg.Publisher
	if publisher == "" {
		publisher = strings.Split(pkg.ID, "/")[0]
	}
	if len(policy.TrustedPublishers) > 0 && !contains(policy.TrustedPublishers, publisher) {
		return []Reason{{Code: "publisher_not_trusted", Detail: publisher}}
	}
	return nil
}

func verifySource(subject Subject, policy domain.TrustPolicy) []Reason {
	source := subject.Evidence.SourceURI
	if source == "" {
		source = subject.Package.ArtifactURI
	}
	if len(policy.TrustedSources) == 0 {
		return nil
	}
	for _, trusted := range policy.TrustedSources {
		if strings.HasPrefix(source, trusted) {
			return nil
		}
	}
	return []Reason{{Code: "source_not_trusted", Detail: source}}
}

func verifySignatures(evidence Evidence, policy domain.TrustPolicy) []Reason {
	return missingReasons("missing_required_signature", policy.RequiredSignatures, evidence.Signatures)
}

func verifyAttestations(evidence Evidence, policy domain.TrustPolicy) []Reason {
	return missingReasons("missing_required_attestation", policy.RequiredAttestations, evidence.Attestations)
}

func verifyProvenance(subject Subject, policy domain.TrustPolicy) []Reason {
	var reasons []Reason
	if len(policy.RequiredAttestations) == 0 {
		return nil
	}
	if subject.Evidence.Provenance.BuilderID != "" && subject.Evidence.Provenance.BuilderID != "builder://strategist" {
		reasons = append(reasons, Reason{Code: "provenance_builder_mismatch", Detail: subject.Evidence.Provenance.BuilderID})
	}
	source := subject.Evidence.SourceURI
	if source == "" {
		source = subject.Package.ArtifactURI
	}
	if subject.Evidence.Provenance.SourceURI != "" && subject.Evidence.Provenance.SourceURI != source {
		reasons = append(reasons, Reason{Code: "provenance_source_mismatch", Detail: subject.Evidence.Provenance.SourceURI})
	}
	return reasons
}

func verifyFreshness(pkg domain.PluginPackage, policy domain.TrustPolicy, now time.Time) []Reason {
	if policy.FreshnessDays <= 0 || pkg.CreatedAt == "" {
		return nil
	}
	created, err := time.Parse(time.RFC3339, pkg.CreatedAt)
	if err != nil {
		return []Reason{{Code: "freshness_timestamp_invalid", Detail: pkg.CreatedAt}}
	}
	if now.Sub(created) > time.Duration(policy.FreshnessDays)*24*time.Hour {
		return []Reason{{Code: "freshness_expired", Detail: pkg.CreatedAt}}
	}
	return nil
}

func verifyRevocation(pkg domain.PluginPackage, policy domain.TrustPolicy) []Reason {
	if contains(policy.RevokedDigests, pkg.Digest) {
		return []Reason{{Code: "digest_revoked", Detail: pkg.Digest}}
	}
	return nil
}

func verifyDeprecation(pkg domain.PluginPackage, policy domain.TrustPolicy) []Reason {
	for _, deprecation := range policy.Deprecations {
		if deprecation.PackageDigest != pkg.Digest {
			continue
		}
		detail := deprecation.ReasonCode
		if detail == "" {
			detail = pkg.Digest
		}
		return []Reason{{Code: "digest_deprecated", Detail: detail}}
	}
	return nil
}

func verifyLicense(pkg domain.PluginPackage, policy domain.TrustPolicy) []Reason {
	if len(policy.AllowedLicenses) > 0 && !contains(policy.AllowedLicenses, pkg.License) {
		return []Reason{{Code: "license_not_allowed", Detail: pkg.License}}
	}
	return nil
}

func matchingDevelopmentException(digest string, policy domain.TrustPolicy, now time.Time) (domain.TrustDevelopmentException, bool, bool) {
	for _, exception := range policy.DevelopmentExceptions {
		if exception.PackageDigest != digest {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, exception.ExpiresAt)
		if err != nil || !now.Before(expiresAt) {
			return exception, true, true
		}
		return exception, true, false
	}
	return domain.TrustDevelopmentException{}, false, false
}
