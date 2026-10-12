package calibration

import "sort"

// CaseStat aggregates every run of one case.
type CaseStat struct {
	ID       string
	ParentID string
	Kind     Kind
	Operator string
	Stage    int
	Band     Band
	Level    float64
	Expect   Expect
	Golden   bool
	HeldOut  bool
	// Observations are the answered runs; Failed counts runs that did not answer.
	Observations []Observation
	Failed       int
	// Local is true when the local checks were evaluated for this case, and Flags
	// are the ones that flagged it.
	Local bool
	Flags []string
}

// Filter narrows which observations count. A stage 2 case counts only when it
// is in Audited; any case in Excluded (an audit disagreement) never counts.
type Filter struct {
	Audited  map[string]bool
	Excluded map[string]bool
	// ExcludedOperators leaves every case of these operators out. It is for an
	// exploratory what-if only: the report says which operators were left out and a
	// calibration is never published from such a run.
	ExcludedOperators map[string]bool
}

func (f Filter) counts(obs Observation) bool {
	if f.Excluded[obs.CaseID] || f.ExcludedOperators[obs.Operator] || obs.Kind == KindCandidate {
		return false
	}
	return obs.Stage < 2 || f.Audited[obs.CaseID]
}

// Stats groups observations by case in a stable order.
func Stats(observations []Observation, filter Filter) []CaseStat {
	byID := map[string]*CaseStat{}
	for _, obs := range observations {
		if !filter.counts(obs) {
			continue
		}
		stat, ok := byID[obs.CaseID]
		if !ok {
			stat = newStat(obs)
			byID[obs.CaseID] = stat
		}
		stat.add(obs)
	}
	out := make([]CaseStat, 0, len(byID))
	for _, stat := range byID {
		out = append(out, *stat)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func newStat(obs Observation) *CaseStat {
	return &CaseStat{ID: obs.CaseID, ParentID: obs.ParentID, Kind: obs.Kind, Operator: obs.Operator, Stage: obs.Stage,
		Band: obs.Band, Level: obs.Level, Expect: obs.Expect, Golden: obs.Golden, HeldOut: obs.HeldOut}
}

func (c *CaseStat) add(obs Observation) {
	if obs.Local {
		c.Local, c.Flags = true, obs.LocalFlags
	}
	if obs.Answered() {
		c.Observations = append(c.Observations, obs)
		return
	}
	c.Failed++
}

// Answered is the number of usable runs.
func (c CaseStat) Answered() int { return len(c.Observations) }

// Scores are the probability of accept of every answered run.
func (c CaseStat) Scores() []float64 {
	scores := make([]float64, len(c.Observations))
	for i, obs := range c.Observations {
		scores[i] = obs.PAccept
	}
	return scores
}

// Score is the mean probability of accept across runs.
func (c CaseStat) Score() float64 { return Mean(c.Scores()) }

// ApprovedAt reports whether the case is approved at a threshold: at least half
// of its answered runs approve. A case with no answered run is not approved.
func (c CaseStat) ApprovedAt(threshold float64) bool {
	if len(c.Observations) == 0 {
		return false
	}
	approved := 0
	for _, obs := range c.Observations {
		if obs.Approved(threshold) {
			approved++
		}
	}
	return approved*2 >= len(c.Observations)
}

// isPositive is a golden parent: a neutral variant of it is the same package again,
// so it is measured as invariance and is not a second independent positive.
func (c CaseStat) isPositive() bool {
	return c.Kind == KindGolden && c.Expect == ExpectAccept && c.Golden && c.Answered() > 0
}

func (c CaseStat) isDefect() bool {
	if c.Expect != ExpectReject || c.Answered() == 0 {
		return false
	}
	return c.Kind == KindSynthetic || c.Kind == KindNatural || c.Kind == KindBoundary
}

func split(stats []CaseStat, heldOut bool) []CaseStat {
	var out []CaseStat
	for _, stat := range stats {
		if stat.HeldOut == heldOut {
			out = append(out, stat)
		}
	}
	return out
}

func positives(stats []CaseStat) []CaseStat { return pick(stats, CaseStat.isPositive) }

func defects(stats []CaseStat) []CaseStat { return pick(stats, CaseStat.isDefect) }

func pick(stats []CaseStat, keep func(CaseStat) bool) []CaseStat {
	var out []CaseStat
	for _, stat := range stats {
		if keep(stat) {
			out = append(out, stat)
		}
	}
	return out
}

func scoresOf(stats []CaseStat) []float64 {
	out := make([]float64, len(stats))
	for i, stat := range stats {
		out[i] = stat.Score()
	}
	return out
}

// falseApproval is the rate of defect cases approved at the threshold.
func falseApproval(defectCases []CaseStat, threshold, rho float64) Rate {
	n := len(defectCases)
	return NewClusterRate(countApproved(defectCases, threshold), n, EffectiveSample(n, parentCount(defectCases), rho))
}

// parentCount is the number of distinct parents the cases come from; a case with
// no parent is its own cluster.
func parentCount(cases []CaseStat) int {
	parents := map[string]bool{}
	for _, c := range cases {
		key := c.ParentID
		if key == "" {
			key = c.ID
		}
		parents[key] = true
	}
	return len(parents)
}

// falseRejection is the rate of positive cases not approved at the threshold.
func falseRejection(positiveCases []CaseStat, threshold float64) Rate {
	return NewRate(len(positiveCases)-countApproved(positiveCases, threshold), len(positiveCases))
}

func countApproved(stats []CaseStat, threshold float64) int {
	n := 0
	for _, stat := range stats {
		if stat.ApprovedAt(threshold) {
			n++
		}
	}
	return n
}
