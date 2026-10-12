package dojo

import (
	"cmp"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// caseSet collects cases with unique ids: a second package with identical
// content adds nothing.
type caseSet struct {
	cases []calibration.Case
	seen  map[string]bool
}

func (s *caseSet) add(batch ...calibration.Case) {
	for _, c := range batch {
		if !s.seen[c.ID] {
			s.seen[c.ID] = true
			s.cases = append(s.cases, c)
		}
	}
}

// BuildCases generates every case of a corpus: for each package its base case
// and its mutations, and for each amendment the before snapshot as a natural
// negative paired with the package.
func BuildCases(corpus Corpus, cal domain.DojoCalibration) []calibration.Case {
	opts := calibration.Options{Seed: cal.Seed, MaxStage: max(cal.Stage, 1), Operators: cal.Operators, Others: corpus.Sources}
	set := caseSet{seen: map[string]bool{}}
	bases := map[string]calibration.Case{}
	for _, src := range corpus.Sources {
		set.add(calibration.Generate(src, opts)...)
		bases[src.Name] = calibration.BaseCase(src)
	}
	for _, before := range corpus.Befores {
		if parent, ok := bases[before.Entry]; ok {
			set.add(calibration.NaturalNegative(parent, before.Seq, before.Fields))
		}
	}
	return set.cases
}

// Summary counts cases by kind, for the dry run.
func Summary(cases []calibration.Case) map[calibration.Kind]int {
	out := map[calibration.Kind]int{}
	for _, c := range cases {
		out[c.Kind]++
	}
	return out
}

// PlannedBytes is the exact size of the projected state the run would send, per
// call, summed over every case and repetition. It uses the same projection the
// provider call uses, supplied by the caller.
func PlannedBytes(cases []calibration.Case, repetitions int, size func(calibration.Case) int) int {
	total := 0
	for _, c := range cases {
		total += size(c)
	}
	return total * max(repetitions, 1)
}

// isDefectCase is a known defect: a synthetic, natural or graded negative that must be rejected.
func isDefectCase(c calibration.Case) bool {
	if c.Expect != calibration.ExpectReject {
		return false
	}
	return c.Kind == calibration.KindSynthetic || c.Kind == calibration.KindNatural || c.Kind == calibration.KindBoundary
}

// CountDefects counts the cases that are known defects.
func CountDefects(cases []calibration.Case) int {
	n := 0
	for _, c := range cases {
		if isDefectCase(c) {
			n++
		}
	}
	return n
}

// DefectParents counts the distinct parents of the defect cases.
func DefectParents(cases []calibration.Case) int {
	parents := map[string]bool{}
	for _, c := range cases {
		if isDefectCase(c) {
			parents[cmp.Or(c.ParentID, c.ID)] = true
		}
	}
	return len(parents)
}
