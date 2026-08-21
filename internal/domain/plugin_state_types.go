package domain

// InstalledInstance is workspace-local materialization state.
type InstalledInstance struct {
	ID                   string `yaml:"id"`
	PackageDigest        string `yaml:"package_digest"`
	AdapterDigest        string `yaml:"adapter_digest"`
	ConnectorID          string `yaml:"connector_id"`
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

// SlotBinding is the local operator's slot selection.
type SlotBinding struct {
	SchemaVersion       string `yaml:"schema_version"`
	Slot                string `yaml:"slot"`
	InstalledInstanceID string `yaml:"installed_instance_id"`
	GrantID             string `yaml:"grant_id,omitempty"`
	Generation          int64  `yaml:"generation"`
	Status              string `yaml:"status"`
	NativeFallback      string `yaml:"native_fallback,omitempty"`
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

// PluginTransaction journals lifecycle transitions.
type PluginTransaction struct {
	SchemaVersion  string `yaml:"schema_version"`
	ID             string `yaml:"id"`
	State          string `yaml:"state"`
	FromGeneration int64  `yaml:"from_generation"`
	ToGeneration   int64  `yaml:"to_generation"`
	RollbackTarget string `yaml:"rollback_target,omitempty"`
}
