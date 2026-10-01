package install

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// normalizedCustomPackage derives the package and adapter manifests from the
// resolved host package and its sidecar declaration, and validates both with
// the same contracts provider add enforces.
func normalizedCustomPackage(resolution customProviderResolution) (domain.PluginPackage, domain.AdapterContract, error) {
	evidence, declaration := resolution.Package.Package, resolution.Declaration
	pkg := domain.PluginPackage{
		SchemaVersion: "strategist-plugin-package/v1", ID: evidence.ID, Publisher: evidence.Publisher, Version: resolution.Version,
		Digest: evidence.Digest, ArtifactURI: evidence.ArtifactURI, ArtifactSize: evidence.ArtifactSize,
		License: firstNonEmpty(declaration.License, "NOASSERTION"), CreatedAt: time.Now().UTC().Format(time.RFC3339),
		ManifestSchema: evidence.ManifestSchema, UpstreamVersion: firstNonEmpty(declaration.UpstreamVersion, resolution.Version),
	}
	adapter := domain.AdapterContract{
		SchemaVersion: "strategist-plugin-adapter/v1", ID: evidence.ID, AdapterRevision: resolution.Version, PluginAPIRange: ">=1 <2",
		SupportedSlots: declaration.SupportedSlots, SupportedRoles: declaration.Roles,
		SupportedHandoffSchemas: declaration.SupportedHandoffSchemas, Capabilities: declaration.Capabilities,
		Entrypoints: []string{customHostEntrypoint}, PackageConstraint: evidence.ID + "@" + strings.SplitN(resolution.Version, ".", 2)[0],
		RequestedPermissions: []domain.PluginPermission{}, RiskScore: declaration.RiskScore, ScratchRoot: declaration.ScratchRoot,
	}
	if err := pkg.Validate(); err != nil {
		return domain.PluginPackage{}, domain.AdapterContract{}, fmt.Errorf("normalized package contract is invalid: %w", err)
	}
	if err := adapter.Validate(); err != nil {
		return domain.PluginPackage{}, domain.AdapterContract{}, fmt.Errorf("normalized adapter contract is invalid: %w", err)
	}
	return pkg, adapter, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// stageCustomPackage writes the normalized files into a private staging
// directory and renames it into providers/<instance>. The rename is the commit:
// nothing partial is ever visible, and an existing instance directory is
// replaced only after the new one is fully written.
func stageCustomPackage(strategistDir string, resolution customProviderResolution) (domain.CustomPackageFacts, string, error) {
	pkg, adapter, err := normalizedCustomPackage(resolution)
	if err != nil {
		return domain.CustomPackageFacts{}, "", err
	}
	packageRaw, err := yaml.Marshal(pkg)
	if err != nil {
		return domain.CustomPackageFacts{}, "", fmt.Errorf("encode package.yaml: %w", err)
	}
	adapterRaw, err := yaml.Marshal(adapter)
	if err != nil {
		return domain.CustomPackageFacts{}, "", fmt.Errorf("encode adapter.yaml: %w", err)
	}
	facts := domain.CustomPackageFacts{
		PackageID: pkg.ID, PackageVersion: pkg.Version, Role: slotRoleID(domain.SlotName(resolution.Slot)), Slot: resolution.Slot,
		PackageDigest: pkg.Digest, AdapterDigest: fmt.Sprintf("sha256:%x", sha256.Sum256(adapterRaw)),
		RuntimeKind: domain.RankedRuntimeHost, ConnectorID: customHostConnector, Entrypoint: customHostEntrypoint,
	}
	target := filepath.Join(strategistDir, "providers", facts.InstanceID())
	if err := commitStagedDirectory(target, map[string][]byte{"package.yaml": packageRaw, "adapter.yaml": adapterRaw}); err != nil {
		return domain.CustomPackageFacts{}, "", err
	}
	return facts, target, nil
}

// commitStagedDirectory writes files beside target and renames them into place.
func commitStagedDirectory(target string, files map[string][]byte) error {
	base := filepath.Dir(target)
	if err := os.MkdirAll(filepath.Join(base, ".staging"), 0o755); err != nil {
		return fmt.Errorf("create provider staging directory: %w", err)
	}
	stage, err := os.MkdirTemp(filepath.Join(base, ".staging"), filepath.Base(target)+"-")
	if err != nil {
		return fmt.Errorf("create provider staging directory: %w", err)
	}
	for name, raw := range files {
		if err := os.WriteFile(filepath.Join(stage, name), raw, 0o644); err != nil { //nolint:gosec // G306: provider manifests are not sensitive
			_ = os.RemoveAll(stage) //nolint:errcheck // best-effort cleanup of the failed stage.
			return fmt.Errorf("write staged %s: %w", name, err)
		}
	}
	if err := os.RemoveAll(target); err != nil {
		_ = os.RemoveAll(stage) //nolint:errcheck // best-effort cleanup of the failed stage.
		return fmt.Errorf("replace installed provider: %w", err)
	}
	if err := os.Rename(stage, target); err != nil {
		_ = os.RemoveAll(stage) //nolint:errcheck // best-effort cleanup of the failed stage.
		return fmt.Errorf("commit installed provider: %w", err)
	}
	return nil
}
