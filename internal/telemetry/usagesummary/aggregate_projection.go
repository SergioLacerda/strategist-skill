package usagesummary

import "github.com/SergioLacerda/strategist-skill/internal/domain"

func aggregateContributions(contribs []contribution) (map[string]*accumulator, map[string]*accumulator, *accumulator, *accumulator) {
	byPhase, byRun := map[string]*accumulator{}, map[string]*accumulator{}
	unattributed, total := &accumulator{}, &accumulator{}
	for _, c := range contribs {
		phaseBucket(byPhase, unattributed, c).add(c)
		runBucket(byRun, c).add(c)
		total.add(c)
	}
	return byPhase, byRun, unattributed, total
}

func phaseBucket(byPhase map[string]*accumulator, unattributed *accumulator, c contribution) *accumulator {
	if c.phase == domain.UsagePhaseUnattributed {
		return unattributed
	}
	key := usageRowKey(c.phase)
	if byPhase[key] == nil {
		byPhase[key] = &accumulator{}
	}
	return byPhase[key]
}

func runBucket(byRun map[string]*accumulator, c contribution) *accumulator {
	if c.phaseRunID == "" {
		return &accumulator{}
	}
	key := phaseRunKey(usageRowKey(c.phase), c.phaseRunID)
	if byRun[key] == nil {
		byRun[key] = &accumulator{}
	}
	return byRun[key]
}

func fillBuckets(dst map[string]Bucket, src map[string]*accumulator) {
	for key, acc := range src {
		dst[key] = acc.bucket()
	}
}

// phaseRunKey is the stable projection key shared by usage aggregation and
// the lifecycle read model. The phase is normalized exactly as ByPhase does.
func phaseRunKey(phase, runID string) string { return phase + "|" + runID }
