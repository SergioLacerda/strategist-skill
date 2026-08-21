package domain

import (
	"fmt"
	"regexp"
)

var pluginDigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

var validPluginPermissions = stringSet(
	string(PluginPermissionReadWorkspace),
	string(PluginPermissionWriteAnalysis),
	string(PluginPermissionWriteDocs),
	string(PluginPermissionWriteSource),
	string(PluginPermissionNetworkAccess),
	string(PluginPermissionSubprocess),
	string(PluginPermissionExternalApp),
	string(PluginPermissionSecretAccess),
)

// Validate returns an error unless every version dimension is present.
func (v PluginVersionVector) Validate() error {
	var errs []string
	requireNonEmpty(&errs, "manifest_schema", v.ManifestSchema)
	requireNonEmpty(&errs, "plugin_api", v.PluginAPI)
	requireNonEmpty(&errs, "adapter_revision", v.AdapterRevision)
	requireNonEmpty(&errs, "upstream_version", v.UpstreamVersion)
	requireNonEmpty(&errs, "connector_api", v.ConnectorAPI)
	return joinPluginValidation("plugin version vector", errs)
}

// Validate checks required publisher fields and rejects local-state leakage.
func (p PluginPackage) Validate() error {
	var errs []string
	requireNonEmpty(&errs, "schema_version", p.SchemaVersion)
	requireNonEmpty(&errs, "id", p.ID)
	requireNonEmpty(&errs, "version", p.Version)
	requireDigest(&errs, "digest", p.Digest)
	requireNonEmpty(&errs, "artifact_uri", p.ArtifactURI)
	if p.ArtifactSize <= 0 {
		errs = append(errs, "artifact_size must be positive")
	}
	if p.ArtifactSize > MaxPluginManifestBytes {
		errs = append(errs, fmt.Sprintf("artifact_size exceeds %d bytes", MaxPluginManifestBytes))
	}
	requireNonEmpty(&errs, "license", p.License)
	requireNonEmpty(&errs, "created_at", p.CreatedAt)
	requireNonEmpty(&errs, "manifest_schema", p.ManifestSchema)
	requireNonEmpty(&errs, "upstream_version", p.UpstreamVersion)
	if p.BindingStatus != "" {
		errs = append(errs, "binding_status belongs to SlotBinding")
	}
	return joinPluginValidation("plugin package", errs)
}

// Validate checks the phase-1 adapter boundary without probing runtime state.
func (a AdapterContract) Validate() error {
	var errs []string
	requireNonEmpty(&errs, "schema_version", a.SchemaVersion)
	requireNonEmpty(&errs, "id", a.ID)
	requireNonEmpty(&errs, "adapter_revision", a.AdapterRevision)
	requireNonEmpty(&errs, "plugin_api_range", a.PluginAPIRange)
	if len(a.SupportedSlots) == 0 {
		errs = append(errs, "supported_slots must have at least one entry")
	}
	for _, slot := range a.SupportedSlots {
		if !IsValidSlot(slot) {
			errs = append(errs, fmt.Sprintf("supported_slots contains invalid slot %q", slot))
		}
	}
	if len(a.Entrypoints) == 0 {
		errs = append(errs, "entrypoints must have at least one entry")
	}
	requireNonEmpty(&errs, "package_constraint", a.PackageConstraint)
	for _, permission := range a.RequestedPermissions {
		if !hasString(validPluginPermissions, string(permission)) {
			errs = append(errs, fmt.Sprintf("requested_permissions contains invalid permission %q", permission))
		}
	}
	return joinPluginValidation("adapter contract", errs)
}

// CheckCompatibility evaluates the phase-1 host API dimension.
func (a AdapterContract) CheckCompatibility(v PluginVersionVector) CompatibilityResult {
	if err := v.Validate(); err != nil {
		return CompatibilityResult{Compatible: false, Reasons: []CompatibilityReason{{
			Dimension: "version_vector",
			Code:      "invalid_version_vector",
			Detail:    err.Error(),
		}}}
	}
	if !pluginAPIMatchesRange(v.PluginAPI, a.PluginAPIRange) {
		return CompatibilityResult{Compatible: false, Reasons: []CompatibilityReason{{
			Dimension: "plugin_api",
			Code:      "unsupported_host_api",
			Detail:    fmt.Sprintf("%s is outside %s", v.PluginAPI, a.PluginAPIRange),
		}}}
	}
	return CompatibilityResult{Compatible: true}
}
