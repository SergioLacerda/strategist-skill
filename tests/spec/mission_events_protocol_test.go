//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// The FSM only moves when an event is submitted, and the roles run as agents
// that never submit one, so a finished mission used to read BOOTSTRAP/INIT. The
// protocol therefore names, per pipeline step, the event the Strategist shell
// submits. This pins that section to the engine's own event vocabulary.
func TestAgentProtocolNamesTheMissionEventOfEveryPipelineStep(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	protocol := readFile(t, filepath.Join(root, "internal", "embed", "defaults", "templates", "agent-protocol.md"))
	_, section, found := strings.Cut(protocol, "## Mission State Events")
	if !found {
		t.Fatal("agent-protocol.md has no `## Mission State Events` section")
	}
	if next := strings.Index(section, "\n## "); next >= 0 {
		section = section[:next]
	}

	ordered := []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone,
		domain.MissionEventRefinementDone, domain.MissionEventNoTasks,
		domain.MissionEventGateApproved, domain.MissionEventGateApprovedAnalysisOnly,
		domain.MissionEventGateRevision, domain.MissionEventGateDenied,
		domain.MissionEventHandoffPassed, domain.MissionEventSniperDone,
	}
	last := -1
	for _, event := range ordered {
		at := strings.Index(section, "`"+string(event)+"`")
		if at < 0 {
			t.Fatalf("Mission State Events section does not name `%s`", event)
		}
		if strings.HasPrefix(string(event), "gate_") || event == domain.MissionEventNoTasks {
			continue // alternatives of one step: presence is enough, order is by step
		}
		if at < last {
			t.Fatalf("event `%s` is listed out of pipeline order", event)
		}
		last = at
	}
	for _, needle := range []string{"Strategist shell", "never replaces", "handoff_challenge_passed"} {
		if !strings.Contains(section, needle) {
			t.Fatalf("Mission State Events section is missing %q", needle)
		}
	}
}
