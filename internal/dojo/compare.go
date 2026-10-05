package dojo

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
)

// CompareRow is the mean probability of accept of one operator level in two cases.
type CompareRow struct {
	Operator string
	Level    float64
	A, B     float64
	NA, NB   int
}

// SweepPoint is an exploratory operating point over every case of one side.
type SweepPoint struct {
	Threshold      float64
	FalseApproval  float64
	FalseRejection float64
}

// Comparison puts two cases' latest observations side by side. It is a laboratory
// reading: it uses every counted case, not the held-out split, publishes nothing
// and recommends nothing.
type Comparison struct {
	Rows         []CompareRow
	AUCA, AUCB   float64
	SweepA       []SweepPoint
	SweepB       []SweepPoint
	NameA, NameB string
}

// Compare computes the comparison of two sets of observations.
func Compare(nameA, nameB string, a, b []calibration.Observation) Comparison {
	statsA, statsB := calibration.Stats(a, calibration.Filter{}), calibration.Stats(b, calibration.Filter{})
	out := Comparison{NameA: nameA, NameB: nameB, Rows: compareRows(statsA, statsB)}
	out.AUCA, out.SweepA = labReading(statsA)
	out.AUCB, out.SweepB = labReading(statsB)
	return out
}

type groupKey struct {
	operator string
	level    float64
}

func groupMeans(stats []calibration.CaseStat) map[groupKey][]float64 {
	groups := map[groupKey][]float64{}
	for _, s := range stats {
		if s.Answered() == 0 {
			continue
		}
		key := groupKey{operator: s.Operator, level: s.Level}
		if s.Operator == "" && s.Golden {
			key.operator = "(intact golden package)"
		}
		groups[key] = append(groups[key], s.Score())
	}
	return groups
}

func compareRows(a, b []calibration.CaseStat) []CompareRow {
	ga, gb := groupMeans(a), groupMeans(b)
	var rows []CompareRow
	for key := range unionKeys(ga, gb) {
		rows = append(rows, CompareRow{Operator: key.operator, Level: key.level, A: calibration.Mean(ga[key]), B: calibration.Mean(gb[key]), NA: len(ga[key]), NB: len(gb[key])})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Operator != rows[j].Operator {
			return rows[i].Operator < rows[j].Operator
		}
		return rows[i].Level < rows[j].Level
	})
	return rows
}

func unionKeys(a, b map[groupKey][]float64) map[groupKey]bool {
	keys := map[groupKey]bool{}
	for key := range a {
		keys[key] = true
	}
	for key := range b {
		keys[key] = true
	}
	return keys
}

// labReading is the AUC and an operating sweep over every counted golden positive
// and defect negative of one side.
func labReading(stats []calibration.CaseStat) (float64, []SweepPoint) {
	var pos, neg []float64
	for _, s := range stats {
		if s.Answered() == 0 {
			continue
		}
		switch {
		case s.Expect == calibration.ExpectAccept && s.Golden && s.Kind == calibration.KindGolden:
			pos = append(pos, s.Score())
		case s.Expect == calibration.ExpectReject:
			neg = append(neg, s.Score())
		}
	}
	var sweep []SweepPoint
	for t := 0.50; t < 0.901; t += 0.05 {
		sweep = append(sweep, SweepPoint{Threshold: t, FalseApproval: share(neg, func(v float64) bool { return v > t }), FalseRejection: share(pos, func(v float64) bool { return v <= t })})
	}
	return calibration.AUC(pos, neg), sweep
}

func share(values []float64, hit func(float64) bool) float64 {
	if len(values) == 0 {
		return 0
	}
	n := 0
	for _, v := range values {
		if hit(v) {
			n++
		}
	}
	return float64(n) / float64(len(values))
}

// Render writes the comparison as plain text.
func (c Comparison) Render(w io.Writer) error {
	var b strings.Builder
	fmt.Fprintf(&b, "laboratory comparison (every counted case, not the held-out split; no calibration, no recommendation)\n")
	fmt.Fprintf(&b, "%-44s %5s  %8s %8s  %8s\n", "operator", "level", c.NameA, c.NameB, "delta")
	for _, r := range c.Rows {
		fmt.Fprintf(&b, "%-44s %5.2f  %8.2f %8.2f  %+8.2f\n", r.Operator, r.Level, r.A, r.B, r.B-r.A)
	}
	fmt.Fprintf(&b, "\nAUC over golden positives and defects: %s %.3f, %s %.3f\n", c.NameA, c.AUCA, c.NameB, c.AUCB)
	fmt.Fprintf(&b, "exploratory operating points (false approval / false rejection), %s then %s\n", c.NameA, c.NameB)
	for i := range c.SweepA {
		fmt.Fprintf(&b, "  t=%.2f  %5.1f%% / %5.1f%%   %5.1f%% / %5.1f%%\n", c.SweepA[i].Threshold, 100*c.SweepA[i].FalseApproval, 100*c.SweepA[i].FalseRejection, 100*c.SweepB[i].FalseApproval, 100*c.SweepB[i].FalseRejection)
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("dojo: write comparison: %w", err)
	}
	return nil
}
