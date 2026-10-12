package application

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// MissionUsageSourceAgentReport identifies usage reported by the invoking agent.
const MissionUsageSourceAgentReport = "agent_report"

// MissionTokenUsageRecord is the adapter-neutral view of a legacy
// mission-token-usage record, kept only so the read side can filter it.
type MissionTokenUsageRecord struct {
	MissionID  string
	TokensIn   int64
	TokensOut  int64
	Source     string
	ReportedAt string
}

// MissionUsageInput is one usage observation declared by the invoking agent.
// A nil count means "not observed"; it is never turned into zero.
type MissionUsageInput struct {
	MissionID, Phase, Role            string
	InvocationID, PhaseRunID, EventID string
	UsageScope, SessionID, RecordKind string
	Source, Quality, Provider, Model  string
	Attempt                           int
	TokensInput, TokensOutput         *int64
	TokensCache, TokensReasoning      *int64
}

// MissionUsageRecord is the adapter-neutral durable usage record. The adapter
// derives RecordID because the identity rule lives with the ledger.
type MissionUsageRecord struct {
	RecordID, EventID, MissionID, Phase, PhaseRunID, Role   string
	InvocationID, UsageScope, SessionID, RecordKind         string
	Provider, Model, MeasurementSource, MeasurementQuality  string
	ObservedAt                                              string
	Attempt                                                 int
	TokensInput, TokensOutput, TokensCache, TokensReasoning *int64
	DedupUnavailable                                        bool
	Calls                                                   int
}

// MissionUsagePorts provides mission lookup and usage persistence. Append
// reports false for a duplicate delivery.
type MissionUsagePorts struct {
	MissionKnown func(basePath, missionID string) bool
	// ResolvePhaseRunID supplies the only open lifecycle run when the caller
	// omitted --phase-run-id. Nil preserves compatibility for library callers
	// that do not have a lifecycle ledger.
	ResolvePhaseRunID func(root, missionID, phase string) (string, error)
	Append            func(root string, record MissionUsageRecord) (bool, error)
}

// MissionUsageResult reports the persisted record and whether it was new.
type MissionUsageResult struct {
	Record   MissionUsageRecord
	Appended bool
}

// RecordMissionUsage validates one declared observation, normalizes its
// defaults and persists it. The CLI path accepts only the sources it cannot
// verify (agent_report, estimator): a measured quality needs an adapter that
// observes the provider or host counter itself. The artifact requirement is
// kept: usage is refused for a mission with no artifact yet.
func RecordMissionUsage(root, basePath string, in MissionUsageInput, now time.Time, ports MissionUsagePorts) (MissionUsageResult, error) {
	return recordMissionUsage(root, basePath, in.MissionID, in.Phase, in.PhaseRunID, ports, "mission report-usage: usage ports are required", func(resolved string) (MissionUsageRecord, error) {
		in.PhaseRunID = resolved
		return normalizeMissionUsage(in, now)
	})
}

func recordMissionUsage(root, basePath, missionID, phase, phaseRunID string, ports MissionUsagePorts, missingPortsMessage string, build func(string) (MissionUsageRecord, error)) (MissionUsageResult, error) {
	if ports.MissionKnown == nil || ports.Append == nil {
		return MissionUsageResult{}, errors.New(missingPortsMessage)
	}
	resolved, err := resolveUsagePhaseRun(root, missionID, phase, phaseRunID, ports.ResolvePhaseRunID)
	if err != nil {
		return MissionUsageResult{}, err
	}
	rec, err := build(resolved)
	if err != nil {
		return MissionUsageResult{}, err
	}
	return persistMissionUsage(root, basePath, rec, ports)
}

// persistMissionUsage applies the artifact requirement and appends the record.
func persistMissionUsage(root, basePath string, rec MissionUsageRecord, ports MissionUsagePorts) (MissionUsageResult, error) {
	if !ports.MissionKnown(basePath, rec.MissionID) {
		return MissionUsageResult{}, fmt.Errorf("unknown mission_id %q (no pending/refined/archived artifact found under %s)", rec.MissionID, basePath)
	}
	appended, err := ports.Append(root, rec)
	if err != nil {
		return MissionUsageResult{}, fmt.Errorf("persist usage: %w", err)
	}
	return MissionUsageResult{Record: rec, Appended: appended}, nil
}

func normalizeMissionUsage(in MissionUsageInput, now time.Time) (MissionUsageRecord, error) {
	if strings.TrimSpace(in.MissionID) == "" {
		return MissionUsageRecord{}, errors.New("mission id is required")
	}
	phase, ok := domain.ParseUsagePhase(in.Phase)
	if !ok {
		return MissionUsageRecord{}, fmt.Errorf("--phase %q is not a usage phase (use bootstrap, intake, discovery, refinement, execution or unattributed; the Approval Gate carries no tokens)", in.Phase)
	}
	rec := MissionUsageRecord{
		EventID: in.EventID, MissionID: in.MissionID, Phase: phase, PhaseRunID: in.PhaseRunID, Role: in.Role,
		InvocationID: in.InvocationID, SessionID: in.SessionID,
		Attempt:    defaultInt(in.Attempt, 1),
		UsageScope: defaultString(in.UsageScope, "invocation"),
		RecordKind: defaultString(in.RecordKind, string(domain.UsageKindIncremental)),
		Provider:   defaultString(in.Provider, "unknown"), Model: defaultString(in.Model, "unknown"),
		TokensInput: in.TokensInput, TokensOutput: in.TokensOutput,
		TokensCache: in.TokensCache, TokensReasoning: in.TokensReasoning,
		ObservedAt:       now.UTC().Format(time.RFC3339Nano),
		DedupUnavailable: in.EventID == "" && in.InvocationID == "",
	}
	if err := applyUsageQuality(&rec, in); err != nil {
		return MissionUsageRecord{}, err
	}
	return rec, validateMissionUsage(rec)
}

func applyUsageQuality(rec *MissionUsageRecord, in MissionUsageInput) error {
	source := domain.UsageSource(defaultString(in.Source, string(domain.UsageSourceAgentReport)))
	if source != domain.UsageSourceAgentReport && source != domain.UsageSourceEstimator {
		return fmt.Errorf("--source %q is not accepted by the CLI: only agent_report and estimator are, because the binary cannot verify a provider or host counter", source)
	}
	quality := domain.UsageQuality(in.Quality)
	if in.Quality == "" {
		quality = domain.DefaultUsageQuality(source)
	}
	if !domain.UsageQualityAllowedForSource(source, quality) {
		return fmt.Errorf("--quality %q is stronger than --source %q allows", quality, source)
	}
	rec.MeasurementSource, rec.MeasurementQuality = string(source), string(quality)
	return nil
}

func validateMissionUsage(rec MissionUsageRecord) error {
	problems := append(usageIdentityProblems(rec), negativeUsageCounts(rec)...)
	if rec.TokensInput == nil && rec.TokensOutput == nil {
		problems = append(problems, "at least one of --tokens-in or --tokens-out is required")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func usageIdentityProblems(rec MissionUsageRecord) []string {
	var problems []string
	if strings.TrimSpace(rec.Role) == "" {
		problems = append(problems, "--role is required")
	}
	if !domain.ValidUsageRecordKind(rec.RecordKind) {
		problems = append(problems, fmt.Sprintf("--record-kind %q must be incremental or cumulative_snapshot", rec.RecordKind))
	}
	if rec.RecordKind == string(domain.UsageKindCumulativeSnapshot) && rec.SessionID == "" {
		problems = append(problems, "a cumulative_snapshot requires --session-id")
	}
	if rec.Attempt < 1 {
		problems = append(problems, fmt.Sprintf("--attempt must be >= 1, got %d", rec.Attempt))
	}
	return problems
}

func negativeUsageCounts(rec MissionUsageRecord) []string {
	var out []string
	for name, p := range map[string]*int64{"--tokens-in": rec.TokensInput, "--tokens-out": rec.TokensOutput, "--tokens-cache": rec.TokensCache, "--tokens-reasoning": rec.TokensReasoning} {
		if p != nil && *p < 0 {
			out = append(out, fmt.Sprintf("%s must be >= 0, got %d", name, *p))
		}
	}
	return out
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func defaultInt(v, fallback int) int {
	if v == 0 {
		return fallback
	}
	return v
}
