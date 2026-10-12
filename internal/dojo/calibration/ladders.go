package calibration

import (
	"fmt"
	"sort"
	"strings"
)

type rung struct {
	level float64
	score float64
}

// ladder is one parent's graded operator with its rungs ordered by level, the
// parent being level 0.
type ladder struct {
	operator string
	parent   string
	rungs    []rung
}

// monotonicGate checks that, for each graded operator of one parent, the score
// never rises with the level beyond the effective tolerance: the larger of the
// configured tolerance and NoiseFactor times the measured repetition noise. The
// value is the number of inverted steps and the detail names the operators that
// have them. A ladder whose top level is not lower than its parent by that
// tolerance is flat: the judge does not scale with that defect, a limitation
// reported in the detail and not an inversion.
func monotonicGate(dev []CaseStat, limits Limits) Gate {
	groups := ladders(dev)
	if len(groups) == 0 {
		return unmeasured(GateMonotonic, 0, "needs graded cases with their parent")
	}
	tolerance := max(limits.MonotonicTol, limits.NoiseFactor*noiseP95(dev))
	tally := tallyLadders(groups, tolerance)
	detail := fmt.Sprintf("%d graded ladders, %d steps, tolerance %.3f, %d flat (the judge does not scale with that defect)%s", len(groups), tally.steps, tolerance, tally.flat, offenderText(tally.offenders))
	if limits.MaxInversionRate <= 0 {
		return decide(GateMonotonic, float64(tally.inversions), 0, tally.inversions == 0, detail)
	}
	effective := EffectiveSample(tally.steps, tally.parents, DefaultClusterRho)
	if need := MinSample(limits.MaxInversionRate); effective < float64(need) {
		return unmeasured(GateMonotonic, limits.MaxInversionRate, fmt.Sprintf("%d steps from %d parents are an effective %.0f; a rate bound of %.2f needs at least %d; %s", tally.steps, tally.parents, effective, limits.MaxInversionRate, need, detail))
	}
	rate := NewClusterRate(tally.inversions, tally.steps, effective)
	detail = fmt.Sprintf("%d of %d steps inverted, exact upper bound %.3f on an effective %d; ", tally.inversions, tally.steps, rate.Upper, effectiveOf(rate, tally.steps)) + detail
	return decide(GateMonotonic, rate.Upper, limits.MaxInversionRate, rate.Upper <= limits.MaxInversionRate, detail)
}

type ladderTally struct {
	inversions, steps, flat, parents int
	offenders                        map[string]int
}

func tallyLadders(groups []ladder, tolerance float64) ladderTally {
	t := ladderTally{offenders: map[string]int{}}
	parents := map[string]bool{}
	for _, l := range groups {
		n := l.invertedSteps(tolerance)
		t.inversions += n
		t.steps += max(len(l.rungs)-1, 0)
		parents[l.parent] = true
		if n > 0 {
			t.offenders[l.operator] += n
		}
		if l.isFlat(tolerance) {
			t.flat++
		}
	}
	t.parents = len(parents)
	return t
}

func offenderText(offenders map[string]int) string {
	if len(offenders) == 0 {
		return ""
	}
	names := make([]string, 0, len(offenders))
	for name := range offenders {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, len(names))
	for i, name := range names {
		parts[i] = fmt.Sprintf("%s (%d)", name, offenders[name])
	}
	return "; inverted in " + strings.Join(parts, ", ")
}

func (l ladder) isFlat(tolerance float64) bool {
	n := len(l.rungs)
	return n > 1 && l.rungs[n-1].score > l.rungs[0].score-tolerance
}

func (l ladder) invertedSteps(tolerance float64) int {
	n := 0
	for i := 1; i < len(l.rungs); i++ {
		if l.rungs[i].score > l.rungs[i-1].score+tolerance {
			n++
		}
	}
	return n
}

// ladders returns, per parent and operator, the parent and its graded children
// ordered by level.
func ladders(dev []CaseStat) []ladder {
	byID := map[string]CaseStat{}
	for _, stat := range dev {
		byID[stat.ID] = stat
	}
	grouped := map[string]*ladder{}
	for _, stat := range dev {
		parent, ok := ladderParent(stat, byID)
		if !ok {
			continue
		}
		key := stat.ParentID + "|" + stat.Operator
		if grouped[key] == nil {
			grouped[key] = &ladder{operator: stat.Operator, parent: stat.ParentID, rungs: []rung{{0, parent.Score()}}}
		}
		grouped[key].rungs = append(grouped[key].rungs, rung{stat.Level, stat.Score()})
	}
	return orderedLadders(grouped)
}

func ladderParent(stat CaseStat, byID map[string]CaseStat) (CaseStat, bool) {
	parent, ok := byID[stat.ParentID]
	if stat.Kind != KindBoundary || !ok || !parent.Golden {
		return CaseStat{}, false
	}
	return parent, stat.Answered() > 0 && parent.Answered() > 0
}

func orderedLadders(grouped map[string]*ladder) []ladder {
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ladder, 0, len(keys))
	for _, key := range keys {
		l := grouped[key]
		sort.Slice(l.rungs, func(i, j int) bool { return l.rungs[i].level < l.rungs[j].level })
		out = append(out, *l)
	}
	return out
}

// effectiveOf is the effective sample a rate stands on: its own, or n when cases count as independent.
func effectiveOf(rate Rate, n int) int {
	if rate.Effective > 0 {
		return rate.Effective
	}
	return n
}
