package domain

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// CatalogRankedStamp is the minimal certification data needed to resolve a
// mode: ranked binding, read directly from the materialized
// .strategist/plugins/catalog.yaml — see
// docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md. It
// intentionally omits every field only relevant to Custom-pipeline
// resolution or Wizard display (see ProviderContract for the fuller shape
// the install package's Wizard code uses).
type CatalogRankedStamp struct {
	ID                      string   `yaml:"id"`
	CanonicalRole           string   `yaml:"canonical_role"`
	Roles                   []string `yaml:"roles"`
	Ranked                  bool     `yaml:"ranked"`
	CertificationDigest     string   `yaml:"certification_digest"`
	RankedBindingGeneration int64    `yaml:"ranked_binding_generation"`
	RankedBindingStatus     string   `yaml:"ranked_binding_status"`
	// HostAPIDigest, ConnectorDigest, TestSuiteDigest, PolicyDigest, and ConformanceLevel
	// are ADR-0043 DEC-006's generic conformance evidence — see
	// internal/check/check_readiness.go's evaluateRankedConformance, the
	// only reader of these four fields.
	HostAPIDigest    string        `yaml:"host_api_digest"`
	ConnectorDigest  string        `yaml:"connector_digest"`
	TestSuiteDigest  string        `yaml:"test_suite_digest"`
	PolicyDigest     string        `yaml:"policy_digest"`
	ConformanceLevel string        `yaml:"conformance_level"`
	Runtime          WeaponRuntime `yaml:"runtime"`
}

type catalogRankedStampFile struct {
	SchemaVersion string               `yaml:"schema_version"`
	Providers     []CatalogRankedStamp `yaml:"providers"`
}

// CurrentPluginCatalogSchemaVersion is the only accepted plugins/catalog.yaml
// schema. v2 is the strict Weapon vocabulary cutover: a v1 catalog carries the
// legacy runtime kinds and must be regenerated, never translated.
const CurrentPluginCatalogSchemaVersion = "strategist-plugin-catalog/v2"

// FindCatalogRankedStamp parses raw (a materialized plugins/catalog.yaml's
// bytes) and returns the entry for providerID. rolevalidation and check
// both call this instead of each parsing their own copy of the catalog
// shape — see NewRoleInvocationPlanFromLock's own doc comment on why this
// project avoids a second, parallel implementation of logic another part of
// the codebase already owns.
func FindCatalogRankedStamp(raw []byte, providerID string) (CatalogRankedStamp, bool, error) {
	var file catalogRankedStampFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return CatalogRankedStamp{}, false, fmt.Errorf("parse catalog: %w", err)
	}
	if file.SchemaVersion != CurrentPluginCatalogSchemaVersion {
		return CatalogRankedStamp{}, false, fmt.Errorf("%w: catalog schema_version %q is not supported (want %q); regenerate or reinstall the workspace", ErrLegacyWeaponState, file.SchemaVersion, CurrentPluginCatalogSchemaVersion)
	}
	for _, p := range file.Providers {
		if p.ID != providerID {
			continue
		}
		if isLegacyRuntimeKind(p.Runtime.Kind) {
			return CatalogRankedStamp{}, false, fmt.Errorf("provider %s: %w", p.ID, unsupportedRuntimeKindError(p.Runtime.Kind))
		}
		return p, true, nil
	}
	return CatalogRankedStamp{}, false, nil
}

// HasRole reports whether the stamp declares role affinity for role, via
// either Roles or the legacy single-role CanonicalRole field.
func (s CatalogRankedStamp) HasRole(role string) bool {
	for _, r := range s.Roles {
		if r == role {
			return true
		}
	}
	return s.CanonicalRole == role
}

// Certified reports whether s is a build-time-certified Ranked candidate
// (ADR-0043): both Ranked and a non-empty CertificationDigest must hold — a
// partially-stamped entry (e.g. ranked: true with no digest yet) is not
// certified.
func (s CatalogRankedStamp) Certified() bool {
	return s.Ranked && s.CertificationDigest != ""
}
