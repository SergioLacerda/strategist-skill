package domain

// InstalledInstance is workspace-local materialization state.
type InstalledInstance struct {
	ID                   string `yaml:"id"`
	PackageDigest        string `yaml:"package_digest"`
	AdapterDigest        string `yaml:"adapter_digest"`
	ConnectorID          string `yaml:"connector_id"`
	ProviderOrigin       string `yaml:"provider_origin,omitempty"`
	SeedPath             string `yaml:"seed_path,omitempty"`
	Entrypoint           string `yaml:"entrypoint,omitempty"`
	LockDigest           string `yaml:"lock_digest"`
	TrustPolicyRevision  string `yaml:"trust_policy_revision"`
	VerificationEvidence string `yaml:"verification_evidence"`
	State                string `yaml:"state"`
	LastKnownGood        bool   `yaml:"last_known_good,omitempty"`
}

// PluginInventory stores workspace-local installed instances.
type PluginInventory struct {
	SchemaVersion string              `yaml:"schema_version"`
	Instances     []InstalledInstance `yaml:"instances"`
}

// Slot binding pipeline discriminator values (docs/architecture/strategist-concepts.md
// §"Ranked Class" § Pipeline scope). Custom is a wizard-selected, runtime-verified
// weapon (today's only implemented pipeline); Ranked is a build-time-certified
// weapon that bypasses Custom's runtime trust/grant/readiness machinery entirely.
const (
	SlotBindingModeCustom = "custom"
	SlotBindingModeRanked = "ranked"
)

// SlotBinding is the local operator's slot selection.
type SlotBinding struct {
	SchemaVersion       string `yaml:"schema_version"`
	Slot                string `yaml:"slot"`
	InstalledInstanceID string `yaml:"installed_instance_id"`
	GrantID             string `yaml:"grant_id,omitempty"`
	Generation          int64  `yaml:"generation"`
	Status              string `yaml:"status"`

	// Mode discriminates the Custom vs. Ranked binding pipeline (see the
	// constants above). Empty is legacy: every binding persisted before this
	// field existed is a Custom binding — Ranked was never implemented, so
	// no existing plugins.lock needs migration. Read Mode via EffectiveMode(),
	// not directly, so this default is applied consistently.
	Mode string `yaml:"mode,omitempty"`
}

// EffectiveMode returns b.Mode, defaulting to SlotBindingModeCustom when
// empty (see the Mode field's own doc comment).
func (b SlotBinding) EffectiveMode() string {
	if b.Mode == "" {
		return SlotBindingModeCustom
	}
	return b.Mode
}

// ValidMode reports whether b.Mode is empty (legacy Custom) or one of the
// known discriminator values. A non-empty, unrecognized Mode is invalid —
// fail closed rather than silently defaulting an operator's typo or a future
// schema drift to Custom.
func (b SlotBinding) ValidMode() bool {
	switch b.Mode {
	case "", SlotBindingModeCustom, SlotBindingModeRanked:
		return true
	default:
		return false
	}
}

// TrustPolicy is consumer-owned verification policy.
type TrustPolicy struct {
	SchemaVersion         string                      `yaml:"schema_version"`
	Revision              string                      `yaml:"revision"`
	TrustedPublishers     []string                    `yaml:"trusted_publishers"`
	TrustedSources        []string                    `yaml:"trusted_sources"`
	AllowedLicenses       []string                    `yaml:"allowed_licenses,omitempty"`
	RequiredSignatures    []string                    `yaml:"required_signatures,omitempty"`
	RequiredAttestations  []string                    `yaml:"required_attestations,omitempty"`
	RevokedDigests        []string                    `yaml:"revoked_digests,omitempty"`
	Deprecations          []TrustDeprecation          `yaml:"deprecations,omitempty"`
	FreshnessDays         int                         `yaml:"freshness_days,omitempty"`
	MinimumConformance    string                      `yaml:"minimum_conformance"`
	DevelopmentExceptions []TrustDevelopmentException `yaml:"development_exceptions,omitempty"`
}

// TrustDeprecation records an operator-owned package deprecation decision.
type TrustDeprecation struct {
	PackageDigest     string `yaml:"package_digest"`
	ReasonCode        string `yaml:"reason_code"`
	ReplacementDigest string `yaml:"replacement_digest,omitempty"`
}

// TrustDevelopmentException is a scoped, expiring trust bypass for local development.
type TrustDevelopmentException struct {
	ID            string `yaml:"id"`
	PackageDigest string `yaml:"package_digest"`
	Reason        string `yaml:"reason"`
	ExpiresAt     string `yaml:"expires_at"`
}

// PermissionGrant binds local approval to immutable package and adapter digests.
type PermissionGrant struct {
	SchemaVersion      string             `yaml:"schema_version"`
	ID                 string             `yaml:"id"`
	PackageDigest      string             `yaml:"package_digest"`
	AdapterDigest      string             `yaml:"adapter_digest"`
	GrantedPermissions []PluginPermission `yaml:"granted_permissions"`
	ExpiresAt          string             `yaml:"expires_at,omitempty"`
}

// PermissionGrantFile is the workspace-owned envelope for operator grants.
// The envelope allows future schema evolution without changing the individual
// grant resource contract.
type PermissionGrantFile struct {
	SchemaVersion string            `yaml:"schema_version"`
	Grants        []PermissionGrant `yaml:"grants"`
}

// Plugin governance persistence schema versions.
const (
	TrustPolicySchemaVersion         = "strategist-trust-policy/v1"
	PermissionGrantFileSchemaVersion = "strategist-permission-grants/v1"
)

// PluginLock pins the exact resolved graph for offline replay.
type PluginLock struct {
	SchemaVersion string           `yaml:"schema_version"`
	ResolutionID  string           `yaml:"resolution_id"`
	GraphDigest   string           `yaml:"graph_digest"`
	Nodes         []PluginLockNode `yaml:"nodes"`
}

// PluginLockNode is one locked package/adapter/dependency node.
type PluginLockNode struct {
	ID     string `yaml:"id"`
	Kind   string `yaml:"kind"`
	Digest string `yaml:"digest"`
}

// PluginLockFileSchemaVersion is the schema_version stamped on plugins.lock,
// the on-disk envelope persisting a resolved role/provider binding across
// `strategist install` invocations (docs/adr/0037-wizard-role-binding-persistence.md).
const PluginLockFileSchemaVersion = "strategist-plugin-lock-file/v1"

// PluginLockFile is the on-disk envelope for a lifecycle.Store's durable
// state: which instances are installed and which slot currently binds to
// which instance. It intentionally excludes a Store's Transactions/Dependents
// maps — those are per-run activation journal state, not durable
// configuration to persist across invocations.
type PluginLockFile struct {
	SchemaVersion string          `yaml:"schema_version"`
	Lock          PluginLock      `yaml:"lock"`
	Inventory     PluginInventory `yaml:"inventory"`
	Bindings      []SlotBinding   `yaml:"bindings"`
}

// NodeDigest returns the digest of the single lock node matching id and kind
// (e.g. id="brainstorming", kind="adapter_contract"), or "" if none matches.
// Shared by every caller that needs one node's pinned digest, instead of each
// re-implementing the same lookup over f.Lock.Nodes.
func (f PluginLockFile) NodeDigest(id, kind string) string {
	for _, n := range f.Lock.Nodes {
		if n.ID == id && n.Kind == kind {
			return n.Digest
		}
	}
	return ""
}

// PluginTransaction journals lifecycle transitions.
type PluginTransaction struct {
	SchemaVersion  string `yaml:"schema_version"`
	ID             string `yaml:"id"`
	State          string `yaml:"state"`
	FromGeneration int64  `yaml:"from_generation"`
	ToGeneration   int64  `yaml:"to_generation"`
	RollbackTarget string `yaml:"rollback_target,omitempty"`
}
