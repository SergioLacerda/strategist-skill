package initiative

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

func conflictingOutcomeEvidence(outcomes []OutcomeCorrelation) int {
	statuses := make(map[string]string)
	conflicts := make(map[string]bool)
	for _, outcome := range outcomes {
		for _, ref := range outcome.EvidenceRefs {
			if prior, ok := statuses[ref.ID]; ok && prior != outcome.Status {
				conflicts[ref.ID] = true
			}
			statuses[ref.ID] = outcome.Status
		}
	}
	return len(conflicts)
}

func assessmentIdentity(advice Advice, result Result) (string, string) {
	material := struct {
		Advice        Advice
		Result        Result
		Algorithm     string
		PolicyVersion string
		PolicyDigest  string
	}{advice, result, PreciseShotAlgorithmVersion, advice.PolicyVersion, advice.PolicyDigest}
	inputDigest := digest(material)
	return inputDigest, "psa-" + inputDigest[:24]
}

func digest(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func canonicalReasons(reasons []string) []string {
	seen := make(map[string]struct{}, len(reasons))
	result := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if strings.TrimSpace(reason) == "" {
			continue
		}
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		result = append(result, reason)
	}
	sort.Strings(result)
	return result
}

func lowerConfidence(left, right ConfidenceTier) ConfidenceTier {
	if confidenceRank(left) <= confidenceRank(right) {
		return left
	}
	return right
}

func minConfidence(left, right ConfidenceTier) ConfidenceTier { return lowerConfidence(left, right) }

func confidenceRank(tier ConfidenceTier) int {
	switch tier {
	case ConfidenceLow:
		return 1
	case ConfidenceMedium:
		return 2
	case ConfidenceHigh:
		return 3
	default:
		return 0
	}
}
