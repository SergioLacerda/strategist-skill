package domain

// PreflightResultSchemaVersion is the schema_version stamped on every
// PreflightResult value.
const PreflightResultSchemaVersion = "strategist-preflight-result/v1"

// PreflightResult aggregates strategist check's existing per-slot
// resolution, weapon-binding, and blocked-readiness diagnostics into one
// versioned envelope, per
// docs/adr/0041-cli-enforcement-sequencing-and-role-invocation-plan-naming.md
// D2. It wraps internal/cli/check/check_slots.go's and
// internal/roles/validation/role_weapon.go's already-computed outputs — it
// does not re-derive readiness independently, to avoid two implementations
// of the same check drifting apart (see D2's regression-test rationale).
//
// The interim note of D6 is resolved: P4's RuntimeConnector SPI and its
// readiness vector (domain.PluginReadinessVector) are wired into
// check_slots.go/check_readiness.go, and buildPreflightResult derives each
// Bindings[].Status from that vector (internal/cli/check/check_preflight_json.go).
// Warnings stays the full diagnostic aggregate plus the non-blocking
// advisories; the shape itself is unchanged.
type PreflightResult struct {
	SchemaVersion   string             `json:"schema_version"`
	TaxonomyVersion string             `json:"taxonomy_version,omitempty"`
	Status          string             `json:"status"` // "ready" | "blocked"
	Identity        PreflightIdentity  `json:"identity"`
	Bindings        []PreflightBinding `json:"bindings"`
	Language        *PreflightLanguage `json:"language,omitempty"`
	Warnings        []string           `json:"warnings"`
	Next            string             `json:"next,omitempty"`
}

// PreflightIdentity identifies the workspace a PreflightResult was computed for.
type PreflightIdentity struct {
	Root string `json:"root"`
	Mode string `json:"mode"`
}

// PreflightBinding is one slot's resolved provider and resolution status, as
// already computed by check_slots.go's resolveSlotProvider — never
// recomputed here.
type PreflightBinding struct {
	Slot     string `json:"slot"`
	Provider string `json:"provider"`
	Kind     string `json:"kind"`
	Status   string `json:"status"` // "ready" | "blocked"
}

// PreflightLanguage mirrors active.yaml's language block, so an agent's
// bootstrap step can read the configured chat/UI/docs/code language
// directly from `strategist check --json` instead of having to separately
// and unpromptedly re-read active.yaml (the gap tracked across
// .analysis/archived/01-governance-guardrails-docs' language-config-*
// packages and 20260923-wizard-language-hardening).
type PreflightLanguage struct {
	UI   string `json:"ui,omitempty"`
	Docs string `json:"docs,omitempty"`
	Chat string `json:"chat,omitempty"`
	Code string `json:"code,omitempty"`
}

// ParseLanguage reads active.yaml's loosely-typed `language` field (a
// map[string]any with string values by construction — see
// internal/install/active_yaml_language.go's validateStandaloneLanguage)
// into a PreflightLanguage. Returns nil when absent or malformed rather than
// erroring — an unset/legacy language block is a valid, pre-existing
// configuration state that `strategist check` must not fail on.
func ParseLanguage(raw any) *PreflightLanguage {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	get := func(key string) string {
		value, exists := m[key]
		if !exists {
			return ""
		}
		s, isString := value.(string)
		if !isString {
			return ""
		}
		return s
	}
	lang := &PreflightLanguage{UI: get("ui"), Docs: get("docs"), Chat: get("chat"), Code: get("code")}
	if lang.UI == "" && lang.Docs == "" && lang.Chat == "" && lang.Code == "" {
		return nil
	}
	return lang
}
