package application

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// HostObservedUsageInput is one time window of usage a host recorded in its
// own session files, already projected to numbers by an observer. The values
// are counters the host wrote, not numbers an agent declared.
type HostObservedUsageInput struct {
	MissionID, Phase, Role, InvocationID, PhaseRunID string
	Host, SessionID, Provider                        string
	Models                                           []string
	WindowStart, WindowEnd                           time.Time
	Calls                                            int
	Input, Output, Cache, Reasoning                  *int64
}

// RecordHostObservedUsage persists one observed window as an incremental
// record with source host_counter and quality host_measured. The idempotency
// key is the host session and the window (not the phase), so the same window
// reported twice, even under another phase, is never counted twice; a re-run
// that sees different counts in the same window is a conflict, because the
// host file can lag the live conversation. The artifact requirement applies.
func RecordHostObservedUsage(root, basePath string, in HostObservedUsageInput, now time.Time, ports MissionUsagePorts) (MissionUsageResult, error) {
	return recordMissionUsage(root, basePath, in.MissionID, in.Phase, in.PhaseRunID, ports, "mission observe-usage: usage ports are required", func(resolved string) (MissionUsageRecord, error) {
		in.PhaseRunID = resolved
		return hostObservationRecord(in, now)
	})
}

func hostObservationRecord(in HostObservedUsageInput, now time.Time) (MissionUsageRecord, error) {
	phase, ok := domain.ParseUsagePhase(in.Phase)
	if !ok {
		return MissionUsageRecord{}, fmt.Errorf("--phase %q is not a usage phase (use bootstrap, intake, discovery, refinement, execution or unattributed)", in.Phase)
	}
	if err := validateHostObservation(in); err != nil {
		return MissionUsageRecord{}, err
	}
	rec := MissionUsageRecord{
		EventID:   fmt.Sprintf("host|%s|%s|%s|%s", in.Host, in.SessionID, in.WindowStart.UTC().Format(time.RFC3339), in.WindowEnd.UTC().Format(time.RFC3339)),
		MissionID: in.MissionID, Phase: phase, Role: in.Role, InvocationID: in.InvocationID, PhaseRunID: in.PhaseRunID,
		Attempt: 1, UsageScope: "window", RecordKind: string(domain.UsageKindIncremental),
		Provider: defaultString(in.Provider, "unknown"), Model: modelLabel(in.Models),
		MeasurementSource: string(domain.UsageSourceHostCounter), MeasurementQuality: string(domain.UsageHostMeasured),
		TokensInput: in.Input, TokensOutput: in.Output, TokensCache: in.Cache, TokensReasoning: in.Reasoning,
		Calls: in.Calls, ObservedAt: now.UTC().Format(time.RFC3339Nano),
	}
	return rec, nil
}

func validateHostObservation(in HostObservedUsageInput) error {
	problems := append(hostIdentityProblems(in), hostCountProblems(in)...)
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func hostIdentityProblems(in HostObservedUsageInput) []string {
	var problems []string
	if strings.TrimSpace(in.MissionID) == "" {
		problems = append(problems, "mission id is required")
	}
	if strings.TrimSpace(in.Role) == "" {
		problems = append(problems, "--role is required")
	}
	if in.Host != "claude" && in.Host != "codex" {
		problems = append(problems, fmt.Sprintf("--host %q must be claude or codex", in.Host))
	}
	if strings.TrimSpace(in.SessionID) == "" {
		problems = append(problems, "the host session id is unknown: pass --session-id")
	}
	return problems
}

func hostCountProblems(in HostObservedUsageInput) []string {
	var problems []string
	if in.Calls < 1 || (in.Input == nil && in.Output == nil) {
		problems = append(problems, "the host recorded no usage inside the window: nothing is observed, and nothing is recorded as zero")
	}
	if !in.WindowEnd.After(in.WindowStart) {
		problems = append(problems, "the window end must be after its start")
	}
	return problems
}

func modelLabel(models []string) string {
	switch len(models) {
	case 0:
		return "unknown"
	case 1:
		return models[0]
	}
	return "mixed"
}
