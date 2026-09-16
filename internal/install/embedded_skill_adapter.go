package install

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// externalSkillAdapterFileName is the Strategist-owned project adapter
// sidecar living alongside each external-skills-source/<id>/SKILL.md — the
// two-layer contract ADR-0033 defines: SKILL.md stays a portable ORKA
// package (name/description/metadata only); strategist.yaml carries the
// project-specific fields (roles/canonical_role, risk_score, category) SKILL.md
// must never declare itself.
const externalSkillAdapterFileName = "strategist.yaml"

// externalSkillAdapter is the minimal project adapter this mission needs —
// exactly the fields internal/install/plugin_catalog.go#pluginCatalogProvider
// already requires, not a speculative superset. ADR-0033 defers the full
// adapter schema (T2); this is the smallest slice that lets a real ingestion
// pipeline exist today without inventing that schema prematurely.
type externalSkillAdapter struct {
	CanonicalRole  string   `yaml:"canonical_role"`
	Roles          []string `yaml:"roles,omitempty"`
	RiskScore      string   `yaml:"risk_score"`
	Category       string   `yaml:"category"`
	Default        bool     `yaml:"default,omitempty"`
	AuxiliaryTools []string `yaml:"auxiliary_tools_allowed,omitempty"`
	// ScratchRoot declares whether this weapon creates its own working/scratch
	// files and, if so, that they belong in the runtime domain. Legal values:
	// "runtime" or "none" (or absent, which behaves as "none" — every skill
	// that doesn't need one, e.g. brainstorming). "workspace" is deliberately
	// not a legal value: a weapon's own working format is implementation
	// detail, never a curated mission artifact (see
	// .analysis/refined/20260915-weapon-scratch-root-taxonomy/design.md D1).
	ScratchRoot string `yaml:"scratch_root,omitempty"`
	// SupportedHandoffSchemas — see domain.ProviderContract's field of the
	// same name and internal/install/role_handoff_schemas.go. Omitted by
	// every embedded weapon today — added here only so a future honest
	// declaration isn't silently dropped by ingestion.
	SupportedHandoffSchemas []string `yaml:"supported_handoff_schemas,omitempty"`
	// UpstreamRepo, UpstreamSkillPath, UpstreamVersion, UpstreamCommit,
	// UpstreamContentDigest, and License are ADR-0029 DEC-002's per-provider
	// upstream-identity fields — independent of adapter_revision (this
	// project's own version field), per ADR-0029's "independent upstream and
	// adapter versions" decision. All optional: a package whose upstream
	// provenance has not yet been researched simply omits them (see
	// docs/adr/0029-external-skill-provider-lifecycle.md's own Context for
	// which packages already have this evidence recorded).
	UpstreamRepo          string `yaml:"upstream_repo,omitempty"`
	UpstreamSkillPath     string `yaml:"upstream_skill_path,omitempty"`
	UpstreamVersion       string `yaml:"upstream_version,omitempty"`
	UpstreamCommit        string `yaml:"upstream_commit,omitempty"`
	UpstreamContentDigest string `yaml:"upstream_content_digest,omitempty"`
	License               string `yaml:"license,omitempty"`
}

func loadExternalSkillAdapter(dir, packageID string) (externalSkillAdapter, error) {
	adapterRaw, err := os.ReadFile(filepath.Join(dir, externalSkillAdapterFileName)) //nolint:gosec // G304: operator-declared ingestion source
	if err != nil {
		return externalSkillAdapter{}, fmt.Errorf("external skill %s: read %s: %w", packageID, externalSkillAdapterFileName, err)
	}
	var adapter externalSkillAdapter
	if err := yaml.Unmarshal(adapterRaw, &adapter); err != nil {
		return externalSkillAdapter{}, fmt.Errorf("external skill %s: parse %s: %w", packageID, externalSkillAdapterFileName, err)
	}
	if err := validateExternalSkillAdapter(packageID, adapter); err != nil {
		return externalSkillAdapter{}, err
	}
	adapter = normalizeExternalSkillAdapter(adapter)
	return adapter, nil
}

func validateExternalSkillAdapter(packageID string, adapter externalSkillAdapter) error {
	if (adapter.CanonicalRole == "" && len(adapter.Roles) == 0) || adapter.RiskScore == "" {
		return fmt.Errorf("external skill %s: %s must declare canonical_role and risk_score", packageID, externalSkillAdapterFileName)
	}
	if adapter.ScratchRoot != "" && adapter.ScratchRoot != "runtime" && adapter.ScratchRoot != "none" {
		return fmt.Errorf("external skill %s: %s scratch_root must be \"runtime\" or \"none\", got %q", packageID, externalSkillAdapterFileName, adapter.ScratchRoot)
	}
	return nil
}

func normalizeExternalSkillAdapter(adapter externalSkillAdapter) externalSkillAdapter {
	if len(adapter.Roles) == 0 {
		adapter.Roles = []string{adapter.CanonicalRole}
	}
	if adapter.CanonicalRole == "" && len(adapter.Roles) == 1 {
		adapter.CanonicalRole = adapter.Roles[0]
	}
	adapter.Roles = normalizeRoles(adapter.Roles)
	return adapter
}
