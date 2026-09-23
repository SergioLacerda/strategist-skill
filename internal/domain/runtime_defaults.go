package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// InstallManifestRelPath is the runtime-local manifest written under .strategist/.
const InstallManifestRelPath = ".install-manifest.json"

// RuntimeFileOwnership identifies who owns an installed runtime file.
type RuntimeFileOwnership string

// Runtime file ownership classes used by the installer and check command.
const (
	RuntimeFileNormative RuntimeFileOwnership = "normative"
	RuntimeFileUserOwned RuntimeFileOwnership = "user_owned"
)

// RuntimeDefaultFile describes one embedded default file tracked by Strategist.
// Required files must be present in an installed runtime: `strategist check`
// reports an absent one as runtime_missing, and a present one that differs from
// the embedded default as runtime_stale.
type RuntimeDefaultFile struct {
	Path     string
	Owner    RuntimeFileOwnership
	Required bool
}

// NormativeRuntimeDefaultFiles returns Strategist-owned runtime files that must match embedded defaults.
func NormativeRuntimeDefaultFiles() []RuntimeDefaultFile {
	return []RuntimeDefaultFile{
		{Path: "SKILL.md", Owner: RuntimeFileNormative, Required: true},
		{Path: "skill.yaml", Owner: RuntimeFileNormative, Required: true},
		{Path: "protocol.md", Owner: RuntimeFileNormative, Required: true},
		{Path: "templates/agent-protocol.md", Owner: RuntimeFileNormative, Required: true},
		{Path: "contracts/machine/preflight.yaml", Owner: RuntimeFileNormative, Required: true},
		{Path: "contracts/narrative/05-approval-gate.md", Owner: RuntimeFileNormative, Required: true},
		{Path: "contracts/narrative/06-execution.md", Owner: RuntimeFileNormative, Required: true},
		{Path: "templates/domain/identity/drift-patterns.yaml", Owner: RuntimeFileNormative, Required: true},
	}
}

// NormativeRuntimeDefaultPaths returns paths for Strategist-owned normative runtime files.
func NormativeRuntimeDefaultPaths() []string {
	files := NormativeRuntimeDefaultFiles()
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

// InstallManifest records the embedded defaults installed into a .strategist/ runtime.
type InstallManifest struct {
	Schema                string                `json:"schema"`
	PackageID             string                `json:"package_id"`
	InstalledAt           string                `json:"installed_at"`
	Files                 []InstallManifestFile `json:"files"`
	LevelingPolicyVersion int                   `json:"leveling_policy_version,omitempty"`
	LevelingPolicyDigest  string                `json:"leveling_policy_digest,omitempty"`
}

// InstallManifestFile records one installed default file hash.
type InstallManifestFile struct {
	Path   string               `json:"path"`
	Owner  RuntimeFileOwnership `json:"owner"`
	SHA256 string               `json:"sha256"`
	// History lists the hashes this path was installed with before SHA256,
	// most recent first (bounded). It lets install and check tell a downgrade
	// (an older binary carrying a default this runtime already moved past)
	// from a genuine upgrade, which content alone cannot.
	History []string `json:"history,omitempty"`
}

// NewInstallManifest builds an install manifest from current embedded hashes.
func NewInstallManifest(packageID string, embeddedHashes map[string]string) InstallManifest {
	files := make([]InstallManifestFile, 0, len(NormativeRuntimeDefaultFiles()))
	for _, file := range NormativeRuntimeDefaultFiles() {
		files = append(files, InstallManifestFile{
			Path:   file.Path,
			Owner:  file.Owner,
			SHA256: embeddedHashes[file.Path],
		})
	}
	return InstallManifest{
		Schema:      "strategist.install-manifest.v1",
		PackageID:   packageID,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Files:       files,
	}
}

// FileByPath returns the manifest entry for path when present.
func (m InstallManifest) FileByPath(path string) (InstallManifestFile, bool) {
	for _, file := range m.Files {
		if file.Path == path {
			return file, true
		}
	}
	return InstallManifestFile{}, false
}

// SHA256Hex returns the lowercase hex SHA-256 digest for data.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// RuntimeDefaultDecision is the install/check decision for one normative file.
type RuntimeDefaultDecision string

// Runtime default decisions shared by installer and check diagnostics.
const (
	RuntimeDecisionWriteMissing    RuntimeDefaultDecision = "write_missing"
	RuntimeDecisionKeepCurrent     RuntimeDefaultDecision = "keep_current"
	RuntimeDecisionAutoUpgrade     RuntimeDefaultDecision = "auto_upgrade"
	RuntimeDecisionConflict        RuntimeDefaultDecision = "conflict"
	RuntimeDecisionUnknownManifest RuntimeDefaultDecision = "unknown_manifest"
	RuntimeDecisionForceOverwrite  RuntimeDefaultDecision = "force_overwrite"
	// RuntimeDecisionDowngrade: the runtime holds the last installed default,
	// and the running binary carries one this runtime was installed with
	// earlier — the binary is older than the runtime.
	RuntimeDecisionDowngrade RuntimeDefaultDecision = "downgrade"
)

// RuntimeDefaultDecisionInput contains hash state for one runtime default decision.
type RuntimeDefaultDecisionInput struct {
	Exists       bool
	CurrentHash  string
	EmbeddedHash string
	ManifestHash string
	// ManifestHistory is the manifest entry's History (earlier installed hashes).
	ManifestHistory []string
	HasManifest     bool
	Force           bool
	// AllowDowngrade accepts an older binary's default (explicit rollback).
	AllowDowngrade bool
}

// DecideRuntimeDefaultUpdate chooses how to handle one normative runtime default.
func DecideRuntimeDefaultUpdate(in RuntimeDefaultDecisionInput) RuntimeDefaultDecision {
	// Checked before Force: --force overrides customizations, not the binary's
	// age. Rolling back to an older default takes an explicit AllowDowngrade.
	if isDowngrade(in) {
		return RuntimeDecisionDowngrade
	}
	if in.Force {
		return RuntimeDecisionForceOverwrite
	}
	if !in.Exists {
		return RuntimeDecisionWriteMissing
	}
	if in.CurrentHash == in.EmbeddedHash {
		return RuntimeDecisionKeepCurrent
	}
	if in.HasManifest && in.CurrentHash == in.ManifestHash {
		return RuntimeDecisionAutoUpgrade
	}
	if !in.HasManifest {
		return RuntimeDecisionUnknownManifest
	}
	return RuntimeDecisionConflict
}

// FormatRuntimeStaleDiagnostic formats the user-facing check/install diagnostic for a stale file.
func FormatRuntimeStaleDiagnostic(path string, decision RuntimeDefaultDecision) string {
	switch decision {
	case RuntimeDecisionAutoUpgrade:
		return fmt.Sprintf("runtime_stale_auto_repairable: normative file %q differs from embedded default and matches a previously installed default — run strategist install", path)
	case RuntimeDecisionDowngrade:
		return fmt.Sprintf("runtime_newer_than_binary: normative file %q was installed by a newer strategist binary than the one running — update the binary (make install) instead of reinstalling, or pass --allow-downgrade to roll back deliberately", path)
	case RuntimeDecisionConflict:
		return fmt.Sprintf("runtime_stale_conflict: normative file %q differs from embedded default and previous installed default — inspect the file or run strategist install --force", path)
	case RuntimeDecisionUnknownManifest:
		return fmt.Sprintf("runtime_stale_unknown_manifest: normative file %q differs from embedded default and no install manifest entry exists — run strategist install --force once unless local edits must be preserved manually", path)
	case RuntimeDecisionWriteMissing, RuntimeDecisionKeepCurrent, RuntimeDecisionForceOverwrite:
		return fmt.Sprintf("runtime_stale: normative file %q differs from embedded default — run strategist install --force", path)
	default:
		return fmt.Sprintf("runtime_stale: normative file %q differs from embedded default — run strategist install --force", path)
	}
}
