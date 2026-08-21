package domain

const (
	// MaxPluginManifestBytes bounds untrusted plugin descriptors before decoding.
	MaxPluginManifestBytes = 256 * 1024

	// MaxPluginPathLength bounds workspace-local plugin resource paths.
	MaxPluginPathLength = 240
)

// PluginResourceKind is the canonical vocabulary for the phase-1 plugin domain.
type PluginResourceKind string

const (
	// PluginResourcePackage is publisher-owned immutable package metadata.
	PluginResourcePackage PluginResourceKind = "package"
	// PluginResourceAdapter is host-owned adapter compatibility metadata.
	PluginResourceAdapter PluginResourceKind = "adapter_contract"
	// PluginResourceInventory is workspace-owned installed instance state.
	PluginResourceInventory PluginResourceKind = "inventory"
	// PluginResourceBinding is workspace-owned slot selection state.
	PluginResourceBinding PluginResourceKind = "binding"
	// PluginResourceTrustPolicy is operator-owned trust policy.
	PluginResourceTrustPolicy PluginResourceKind = "trust_policy"
	// PluginResourceGrant is operator-owned permission approval state.
	PluginResourceGrant PluginResourceKind = "grant"
	// PluginResourceLock is resolver-owned graph pinning state.
	PluginResourceLock PluginResourceKind = "lock"
	// PluginResourceTransaction is installer-owned lifecycle journal state.
	PluginResourceTransaction PluginResourceKind = "transaction"
)

// PluginResourceAuthority documents which resource owns which facts.
type PluginResourceAuthority struct {
	Kind      PluginResourceKind
	Authority string
	Owns      []string
}

// SlotExtensionLabel is the user-facing term for a configured slot extension.
// Legacy files still use "provider" in field names; UI should prefer this label.
const SlotExtensionLabel = "slot plugin"

// SlotExtensionKindLabel returns the vocabulary label for a resolved slot target.
func SlotExtensionKindLabel(kind string) string {
	switch kind {
	case "skill_provider":
		return "external skill plugin"
	case "native_role":
		return "native role"
	default:
		return kind
	}
}

// PluginResourceAuthorities returns the non-overlapping ownership table.
func PluginResourceAuthorities() []PluginResourceAuthority {
	return []PluginResourceAuthority{
		{Kind: PluginResourcePackage, Authority: "publisher", Owns: []string{"publisher_identity", "artifact_source", "artifact_digest", "upstream_version", "license"}},
		{Kind: PluginResourceAdapter, Authority: "adapter_maintainer", Owns: []string{"host_api_compatibility", "entrypoints", "role_contracts", "requested_permissions", "adapter_revision"}},
		{Kind: PluginResourceInventory, Authority: "workspace", Owns: []string{"installed_instance", "verification_evidence", "probe_result", "last_known_good"}},
		{Kind: PluginResourceBinding, Authority: "workspace", Owns: []string{"slot_selection", "binding_generation", "fallback_policy", "binding_status"}},
		{Kind: PluginResourceTrustPolicy, Authority: "operator_policy", Owns: []string{"trusted_publishers", "trusted_sources", "freshness_policy", "minimum_conformance"}},
		{Kind: PluginResourceGrant, Authority: "operator_policy", Owns: []string{"granted_permissions", "grant_subject", "grant_expiry"}},
		{Kind: PluginResourceLock, Authority: "resolver", Owns: []string{"resolved_graph", "resolution_digest", "conflict_explanation"}},
		{Kind: PluginResourceTransaction, Authority: "installer", Owns: []string{"journaled_transition", "transaction_state", "rollback_target"}},
	}
}

// PluginVersionVector tracks independent compatibility dimensions.
type PluginVersionVector struct {
	ManifestSchema  string `yaml:"manifest_schema"`
	PluginAPI       string `yaml:"plugin_api"`
	AdapterRevision string `yaml:"adapter_revision"`
	UpstreamVersion string `yaml:"upstream_version"`
	ConnectorAPI    string `yaml:"connector_api"`
}

// PluginPackage is immutable publisher-owned package metadata.
type PluginPackage struct {
	SchemaVersion   string `yaml:"schema_version"`
	ID              string `yaml:"id"`
	Publisher       string `yaml:"publisher,omitempty"`
	Version         string `yaml:"version"`
	Digest          string `yaml:"digest"`
	ArtifactURI     string `yaml:"artifact_uri"`
	ArtifactSize    int64  `yaml:"artifact_size"`
	License         string `yaml:"license"`
	CreatedAt       string `yaml:"created_at"`
	ManifestSchema  string `yaml:"manifest_schema"`
	UpstreamVersion string `yaml:"upstream_version"`

	// BindingStatus is intentionally present only to catch mixed-authority input
	// during migration from provider manifests.
	BindingStatus string `yaml:"binding_status,omitempty"`
}

// PluginPermission is a declared/requested/granted permission vocabulary item.
type PluginPermission string

const (
	// PluginPermissionReadWorkspace allows reading workspace files.
	PluginPermissionReadWorkspace PluginPermission = "workspace.read"
	// PluginPermissionWriteAnalysis allows writing analysis artifacts.
	PluginPermissionWriteAnalysis PluginPermission = "analysis.write"
	// PluginPermissionWriteDocs allows writing documentation artifacts.
	PluginPermissionWriteDocs PluginPermission = "docs.write"
	// PluginPermissionWriteSource allows modifying source files.
	PluginPermissionWriteSource PluginPermission = "source.write"
	// PluginPermissionNetworkAccess allows network calls.
	PluginPermissionNetworkAccess PluginPermission = "network.access"
	// PluginPermissionSubprocess allows subprocess execution.
	PluginPermissionSubprocess PluginPermission = "subprocess.exec"
	// PluginPermissionExternalApp allows connector-backed external app access.
	PluginPermissionExternalApp PluginPermission = "external_app.access"
	// PluginPermissionSecretAccess allows reading approved secrets.
	PluginPermissionSecretAccess PluginPermission = "secret.access"
)

// AdapterContract is Strategist-owned compatibility metadata for one package family.
type AdapterContract struct {
	SchemaVersion        string             `yaml:"schema_version"`
	ID                   string             `yaml:"id"`
	AdapterRevision      string             `yaml:"adapter_revision"`
	PluginAPIRange       string             `yaml:"plugin_api_range"`
	SupportedSlots       []string           `yaml:"supported_slots"`
	Entrypoints          []string           `yaml:"entrypoints"`
	PackageConstraint    string             `yaml:"package_constraint"`
	RequestedPermissions []PluginPermission `yaml:"requested_permissions"`
}

// CompatibilityReason identifies one structured compatibility failure.
type CompatibilityReason struct {
	Dimension string `yaml:"dimension"`
	Code      string `yaml:"code"`
	Detail    string `yaml:"detail"`
}

// CompatibilityResult is a structured compatibility answer, not a Boolean alone.
type CompatibilityResult struct {
	Compatible bool                  `yaml:"compatible"`
	Reasons    []CompatibilityReason `yaml:"reasons,omitempty"`
}
