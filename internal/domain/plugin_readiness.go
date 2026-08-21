package domain

// ReadinessStatus is one explicit state in plugin readiness diagnostics.
type ReadinessStatus string

const (
	// ReadinessReady means the dimension is positively usable.
	ReadinessReady ReadinessStatus = "ready"
	// ReadinessBlocked means the dimension is known to block activation.
	ReadinessBlocked ReadinessStatus = "blocked"
	// ReadinessUnsupported means the runtime cannot support this dimension.
	ReadinessUnsupported ReadinessStatus = "unsupported"
	// ReadinessUnknown means the dimension has not been evaluated yet.
	ReadinessUnknown ReadinessStatus = "unknown"
)

// ReadinessCheck reports one dimension without collapsing unsupported into ready.
type ReadinessCheck struct {
	Status     ReadinessStatus `yaml:"status"`
	ReasonCode string          `yaml:"reason_code,omitempty"`
	Detail     string          `yaml:"detail,omitempty"`
}

// Ready reports whether this dimension is positively ready.
func (c ReadinessCheck) Ready() bool {
	return c.Status == ReadinessReady
}

// PluginReadinessVector is the truthful diagnostic state for a plugin instance.
type PluginReadinessVector struct {
	Descriptor          ReadinessCheck `yaml:"descriptor"`
	Source              ReadinessCheck `yaml:"source"`
	Trust               ReadinessCheck `yaml:"trust"`
	Dependencies        ReadinessCheck `yaml:"dependencies"`
	HostAPI             ReadinessCheck `yaml:"host_api"`
	Connector           ReadinessCheck `yaml:"connector"`
	Entrypoint          ReadinessCheck `yaml:"entrypoint"`
	PermissionGrant     ReadinessCheck `yaml:"permission_grant"`
	EnforcementCoverage ReadinessCheck `yaml:"enforcement_coverage"`
	ActiveBinding       ReadinessCheck `yaml:"active_binding"`
}

// Ready returns true only when every readiness dimension is ready.
func (v PluginReadinessVector) Ready() bool {
	for _, check := range v.checks() {
		if !check.Ready() {
			return false
		}
	}
	return true
}

// ReasonCodes returns non-empty reason codes for non-ready dimensions.
func (v PluginReadinessVector) ReasonCodes() []string {
	var reasons []string
	for _, check := range v.checks() {
		if check.Ready() || check.ReasonCode == "" {
			continue
		}
		reasons = append(reasons, check.ReasonCode)
	}
	return reasons
}

func (v PluginReadinessVector) checks() []ReadinessCheck {
	return []ReadinessCheck{
		v.Descriptor,
		v.Source,
		v.Trust,
		v.Dependencies,
		v.HostAPI,
		v.Connector,
		v.Entrypoint,
		v.PermissionGrant,
		v.EnforcementCoverage,
		v.ActiveBinding,
	}
}
