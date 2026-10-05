package domain

import "strings"

// UsageQuality states how far a token number may be trusted. It is part of
// every number: an unobserved value is `unavailable`, never zero.
type UsageQuality string

// Usage measurement qualities, from strongest to weakest.
const (
	UsageProviderMeasured   UsageQuality = "provider_measured"
	UsageHostMeasured       UsageQuality = "host_measured"
	UsageReportedUnverified UsageQuality = "reported_unverified"
	UsageTokenizerEstimated UsageQuality = "tokenizer_estimated" //nolint:gosec // G101: a quality label, not a credential
	UsageHeuristicEstimated UsageQuality = "heuristic_estimated"
	UsageUnavailable        UsageQuality = "unavailable"
)

// UsageSource is the declared origin (transport) of a usage observation. It
// bounds the strongest quality the observation may claim.
type UsageSource string

// Usage sources. The CLI only accepts agent_report and estimator: the binary
// cannot verify a provider or host counter, so measured sources are reserved
// for adapters that observe them directly.
const (
	UsageSourceAgentReport      UsageSource = "agent_report"
	UsageSourceEstimator        UsageSource = "estimator"
	UsageSourceHostCounter      UsageSource = "host_counter"
	UsageSourceProviderResponse UsageSource = "provider_response"
)

// UsageRecordKind states the semantics of a reported count.
type UsageRecordKind string

// Usage record kinds.
const (
	UsageKindIncremental        UsageRecordKind = "incremental"
	UsageKindCumulativeSnapshot UsageRecordKind = "cumulative_snapshot"
)

// UsagePhaseUnattributed is the usage phase for consumption outside any phase.
const UsagePhaseUnattributed = "UNATTRIBUTED"

var usageQualityRanks = map[UsageQuality]int{
	UsageProviderMeasured: 5, UsageHostMeasured: 4, UsageReportedUnverified: 3,
	UsageTokenizerEstimated: 2, UsageHeuristicEstimated: 1, UsageUnavailable: 0,
}

var usageSourceCeilings = map[UsageSource]UsageQuality{
	UsageSourceAgentReport:      UsageReportedUnverified,
	UsageSourceEstimator:        UsageTokenizerEstimated,
	UsageSourceHostCounter:      UsageHostMeasured,
	UsageSourceProviderResponse: UsageProviderMeasured,
}

var usageSourceDefaults = map[UsageSource]UsageQuality{
	UsageSourceAgentReport:      UsageReportedUnverified,
	UsageSourceEstimator:        UsageHeuristicEstimated,
	UsageSourceHostCounter:      UsageHostMeasured,
	UsageSourceProviderResponse: UsageProviderMeasured,
}

// usagePhaseAliases maps CLI spellings to canonical usage phases. The Approval
// Gate is deliberately absent: it carries duration and never tokens.
var usagePhaseAliases = map[string]string{
	"bootstrap": string(PhaseBootstrap), "intake": string(PhaseIntake),
	"discovery": string(PhaseDiscovery), "refinement": string(PhaseRefinement),
	"refining": string(PhaseRefinement), "execution": string(PhaseExecution),
	"delivery": string(PhaseExecution), "unattributed": UsagePhaseUnattributed,
}

// UsageQualityRank orders qualities; an unknown value ranks -1.
func UsageQualityRank(q UsageQuality) int {
	if rank, ok := usageQualityRanks[q]; ok {
		return rank
	}
	return -1
}

// UsageQualityAllowedForSource reports whether source may claim quality: never
// stronger than the source's ceiling.
func UsageQualityAllowedForSource(source UsageSource, quality UsageQuality) bool {
	ceiling, ok := usageSourceCeilings[source]
	if !ok || UsageQualityRank(quality) < 0 {
		return false
	}
	return UsageQualityRank(quality) <= UsageQualityRank(ceiling)
}

// DefaultUsageQuality is the quality a source claims when none is given.
func DefaultUsageQuality(source UsageSource) UsageQuality {
	if q, ok := usageSourceDefaults[source]; ok {
		return q
	}
	return UsageUnavailable
}

// ValidUsagePhase reports whether phase is a canonical usage phase.
func ValidUsagePhase(phase string) bool {
	for _, canonical := range usagePhaseAliases {
		if canonical == phase {
			return true
		}
	}
	return false
}

// ParseUsagePhase maps a CLI spelling or canonical name to a canonical usage
// phase. The Approval Gate and the terminal phases are rejected.
func ParseUsagePhase(input string) (string, bool) {
	canonical, ok := usagePhaseAliases[strings.ToLower(strings.TrimSpace(input))]
	return canonical, ok
}

// ValidUsageRecordKind reports whether kind is a declared record kind.
func ValidUsageRecordKind(kind string) bool {
	return kind == string(UsageKindIncremental) || kind == string(UsageKindCumulativeSnapshot)
}
