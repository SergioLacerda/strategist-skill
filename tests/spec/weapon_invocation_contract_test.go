//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// INV-01 verdict (option a+): a delegated sub-role reaches an embedded Weapon through the
// host skill loader. That channel is a named, defined degrade, recorded in
// weapon_invocation, and stated once across the contracts.

func defaultsFile(t *testing.T, parts ...string) string {
	t.Helper()
	return readFile(t, filepath.Join(append([]string{repoRoot(t), "internal", "embed", "defaults"}, parts...)...))
}

func requireAll(t *testing.T, label, content string, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(content, needle) {
			t.Fatalf("%s must contain %q", label, needle)
		}
	}
}

func TestWeaponProfileNamesTheAcceptedDegrade(t *testing.T) {
	t.Parallel()
	discovery := defaultsFile(t, "contracts", "narrative", "03-discovery.md")
	requireAll(t, "03-discovery.md", discovery,
		"host skill loader",
		"defined degrade",
		"`native_substitution: forbidden` is unchanged",
		"`weapon_invocation` is required for a delegated run",
		"does not certify the host copy",
		"`resolved_digest`",
	)
}

func TestWeaponInvocationSchemaRequiresItForDelegatedRuns(t *testing.T) {
	t.Parallel()
	schema := defaultsFile(t, "schemas", "handoff-ranger-to-archivist.schema.yaml")
	start := strings.Index(schema, "  weapon_invocation:")
	end := strings.Index(schema, "  evidence_cards:")
	if start < 0 || end < start {
		t.Fatal("weapon_invocation block not found in the Ranger handoff schema")
	}
	requireAll(t, "weapon_invocation block", schema[start:end],
		"required: false",
		"required_when: delegated_run",
		"item_fields: [invoked, resolved_from, steps_dropped, resolved_digest, receipt_status, pin_status, capability_isolation]",
		"sha256:<64 hex>",
		"raw bytes",
	)
}

func TestAgentProtocolStatesOneDiscoveryChannel(t *testing.T) {
	t.Parallel()
	protocol := defaultsFile(t, "templates", "agent-protocol.md")
	if strings.Contains(protocol, "Ranked uses the embedded connector") {
		t.Fatal("agent-protocol.md must not claim the embedded connector as the discovery channel: no production code wires it")
	}
	requireAll(t, "agent-protocol.md", protocol,
		"host skill loader",
		"03-discovery.md",
		"weapon_invocation",
	)
}
