package dojo

import (
	"fmt"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// pipelineEvent is one parsed line from an emit.log structured log line.
type pipelineEvent struct {
	Raw    string
	Key    string
	Phase  string
	Status string
}

// slotRolePrefix maps a pipeline slot name to the emit-log role/phase prefix it drives.
// See docs/strategist-concepts.md: discovery=Ranger, refinement=Archivist, execution=Sniper.
var slotRolePrefix = map[string]string{
	"discovery":  "ranger",
	"refinement": "archivist",
	"execution":  "sniper",
}

// CheckPipeline validates the pipeline section of criteria — which slots were invoked,
// which were not, where the pipeline stopped, and whether it auto-stopped at the approval
// gate — against emit log events. It is a no-op when criteria has no pipeline assertions,
// or when filesOnly is true, since pipeline evaluation depends on emit.log like emit_log
// and timing checks do.
func CheckPipeline(criteria domain.DojoCriteria, logPath string, filesOnly bool) []domain.DojoCheckItem {
	if filesOnly || !hasPipelineCriteria(criteria) {
		return nil
	}

	if !fileExists(logPath) {
		return []domain.DojoCheckItem{newItem("pipeline", false, "emit.log not found — run the LLM scenario first")}
	}
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return []domain.DojoCheckItem{newItem("pipeline", false, err.Error())}
	}

	events := parsePipelineEvents(string(raw))
	var items []domain.DojoCheckItem
	items = append(items, checkSlotsInvoked(criteria.Pipeline.SlotsInvoked, events)...)
	items = append(items, checkSlotsNotInvoked(criteria.Pipeline.SlotsNotInvoked, events)...)
	items = append(items, checkMustStopAt(criteria.Pipeline.MustStopAt, events)...)
	items = append(items, checkAutoStopAtGate(criteria.AutoStopAtGate, events)...)
	return items
}

func hasPipelineCriteria(c domain.DojoCriteria) bool {
	return len(c.Pipeline.SlotsInvoked) > 0 || len(c.Pipeline.SlotsNotInvoked) > 0 ||
		c.Pipeline.MustStopAt != "" || c.AutoStopAtGate
}

func parsePipelineEvents(log string) []pipelineEvent {
	var events []pipelineEvent
	for _, line := range strings.Split(log, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := parseLogFields(line)
		events = append(events, pipelineEvent{
			Raw:    line,
			Key:    fields["key"],
			Phase:  firstNonEmpty(fields["strategist.phase"], fields["phase"]),
			Status: firstNonEmpty(fields["strategist.status"], fields["status"]),
		})
	}
	return events
}

func parseLogFields(line string) map[string]string {
	fields := make(map[string]string)
	for _, tok := range strings.Fields(line) {
		k, v, ok := strings.Cut(tok, "=")
		if !ok {
			continue
		}
		fields[k] = v
	}
	return fields
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func rolePrefix(slot string) string {
	if p, ok := slotRolePrefix[slot]; ok {
		return p
	}
	return slot
}

func slotObserved(slot string, events []pipelineEvent) bool {
	prefix := rolePrefix(slot)
	for _, e := range events {
		if e.Phase == prefix || strings.HasPrefix(e.Key, prefix+"_") {
			return true
		}
	}
	return false
}

func checkSlotsInvoked(slots []string, events []pipelineEvent) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	for _, slot := range slots {
		found := slotObserved(slot, events)
		items = append(items, newItem(fmt.Sprintf("pipeline slots_invoked %s", slot), found,
			fmt.Sprintf("slot %q was not invoked (no %s events in emit.log)", slot, rolePrefix(slot))))
	}
	return items
}

func checkSlotsNotInvoked(slots []string, events []pipelineEvent) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	for _, slot := range slots {
		found := slotObserved(slot, events)
		items = append(items, newItem(fmt.Sprintf("pipeline slots_not_invoked %s", slot), !found,
			fmt.Sprintf("slot %q must not be invoked but %s events were found in emit.log", slot, rolePrefix(slot))))
	}
	return items
}

// checkMustStopAt asserts the last phase-tagged event matches want. An empty want means
// no assertion is made (not every route stops at a named phase).
func checkMustStopAt(want string, events []pipelineEvent) []domain.DojoCheckItem {
	if want == "" {
		return nil
	}
	got := ""
	if last := lastPhaseEvent(events); last != nil {
		got = last.Phase
	}
	passed := got == want
	return []domain.DojoCheckItem{newItem("pipeline must_stop_at "+want, passed,
		fmt.Sprintf("pipeline last stopped at phase %q, expected %q", got, want))}
}

func lastPhaseEvent(events []pipelineEvent) *pipelineEvent {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Phase != "" {
			return &events[i]
		}
	}
	return nil
}

// checkAutoStopAtGate asserts an approval-gate auto-stop event occurred when want is true.
// When want is false, no assertion is made — routes that never reach a gate (e.g. Critical
// Hit's inline gate) have nothing to prove here.
func checkAutoStopAtGate(want bool, events []pipelineEvent) []domain.DojoCheckItem {
	if !want {
		return nil
	}
	found := false
	for _, e := range events {
		if e.Phase == "approval_gate" && e.Status == "auto_stopped" {
			found = true
			break
		}
	}
	return []domain.DojoCheckItem{newItem("pipeline auto_stop_at_gate", found,
		"expected an approval_gate auto-stop event (strategist.phase=approval_gate strategist.status=auto_stopped) in emit.log")}
}
