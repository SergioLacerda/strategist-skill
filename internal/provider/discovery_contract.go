package provider

import (
	"bytes"
	"fmt"
	"strings"
)

// InvocationRequestIDKey is the trusted frontmatter field that binds a pending
// artifact to the single mission invocation request that produced it.
const InvocationRequestIDKey = "invocation_request_id"

// DiscoveryProvenance is the ownership identity parsed from an artifact's
// frontmatter. It is read from structure, never from body text.
type DiscoveryProvenance struct {
	MissionID string
	Status    string
	RequestID string
}

// ParseDiscoveryProvenance parses the frontmatter of an existing discovery
// artifact. A missing or malformed frontmatter is an error, so an artifact
// whose ownership cannot be established is never treated as owned.
func ParseDiscoveryProvenance(raw []byte) (DiscoveryProvenance, error) {
	content := bytes.TrimSpace(raw)
	if !bytes.HasPrefix(content, []byte("---")) {
		return DiscoveryProvenance{}, fmt.Errorf("discovery artifact has no frontmatter")
	}
	frontmatter, _, err := splitDiscoveryFrontmatter(raw)
	if err != nil {
		return DiscoveryProvenance{}, err
	}
	text := func(key string) string {
		value, _ := frontmatter[key].(string) //nolint:errcheck // a non-string value is treated as absent.
		return strings.TrimSpace(value)
	}
	return DiscoveryProvenance{MissionID: text("mission_id"), Status: text("mission_status"), RequestID: text(InvocationRequestIDKey)}, nil
}

// DiscoveryOutputContract tells a Weapon executor the shape Ranger's
// normalization and the Archivist handoff require. It is the single source for
// both the host-bridge prompt and the manual `mission invoke` request.
const DiscoveryOutputContract = "Return Markdown that starts with YAML frontmatter containing a `sources_consulted` list " +
	"(each item: source_path, content_fingerprint, coverage_status; use an empty list when no source was opened) " +
	"and a `ranger_handoff_policy_facts` object with schema_version `strategist-ranger-handoff-policy-facts/v1`, " +
	"boolean fields require_recall, require_boundary, require_classification, require_verdict and informational_only; " +
	"and optionally `confidence_score`, `discovery_subtype` and `evaluation_verdict`. " +
	"The body must contain these exact second-level headings: `## mission_objective`, `## known_facts`, " +
	"`## uncertainties`, `## affected_scope`, `## side_quests`, `## confidence_summary`, " +
	"`## recommended_refinement_focus` and `## handoff`. Strategist overwrites identity fields " +
	"(mission_id, mission_status, schema_version, provider_id)."

// DiscoveryExecutionContract adapts an interactive external skill to Ranger's
// single-shot, read-only Weapon boundary. The host executes the compiled
// payload, but Strategist remains the owner of user interaction and approval.
const DiscoveryExecutionContract = "Execute the compiled Weapon payload once as a single-shot, read-only Ranger discovery call. " +
	"Use the payload's analysis methods, but do not ask the user questions, wait for an approval, invoke another skill, " +
	"write files, or perform implementation. Any interactive or implementation gate inside the payload belongs to its " +
	"standalone workflow and is not part of this adapter. Return only the discovery handoff required by output_contract."
