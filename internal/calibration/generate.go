package calibration

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Source is one package of the corpus with its projected fields.
type Source struct {
	Name     string
	Digest   string
	Boundary string
	Fields   Fields
	Golden   bool
	HeldOut  bool
}

// Options selects what Generate produces.
type Options struct {
	Seed int64
	// MaxStage is the highest operator stage included (default 1).
	MaxStage int
	// Operators, when set, restricts generation to these names.
	Operators []string
	// Others are the other sources of the corpus, used by cross-package operators.
	Others []Source
}

func shortID(digest string) string {
	digest = strings.TrimPrefix(digest, "sha256:")
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

// BaseCase is the unmodified package as a positive case: golden when promoted,
// a candidate otherwise.
func BaseCase(src Source) Case {
	kind := KindCandidate
	if src.Golden {
		kind = KindGolden
	}
	return Case{
		ID: shortID(src.Digest), Kind: kind, Source: src.Name, Boundary: src.Boundary, Expect: ExpectAccept,
		Band: BandClean, Golden: src.Golden, HeldOut: src.HeldOut, Fields: src.Fields, Digest: src.Digest,
	}
}

// Generate builds the base case and every applicable child of one source. A
// child is skipped when its operator's precondition fails, when it produces no
// change, or when it would no longer project any field. The result is
// deterministic for the same source, options and catalog.
func Generate(src Source, opts Options) []Case {
	base := BaseCase(src)
	cases := []Case{base}
	for _, op := range Catalog(src.Boundary) {
		if wanted(op, opts) && op.Applies(src.Fields) {
			cases = append(cases, expand(base, src, op, opts)...)
		}
	}
	return cases
}

func expand(base Case, src Source, op Operator, opts Options) []Case {
	var out []Case
	for _, level := range levelsOf(op) {
		if child, ok := derive(base, src, op, level, opts); ok {
			out = append(out, child)
		}
	}
	return out
}

func wanted(op Operator, opts Options) bool {
	maxStage := opts.MaxStage
	if maxStage == 0 {
		maxStage = 1
	}
	if op.Stage > maxStage {
		return false
	}
	if len(opts.Operators) == 0 {
		return true
	}
	for _, name := range opts.Operators {
		if name == op.Name {
			return true
		}
	}
	return false
}

func levelsOf(op Operator) []float64 {
	if len(op.Levels) == 0 {
		return []float64{1}
	}
	return op.Levels
}

func derive(base Case, src Source, op Operator, level float64, opts Options) (Case, bool) {
	key := base.ID + "~" + op.Name
	id := key
	if len(op.Levels) > 0 {
		id += fmt.Sprintf("@%.2f", level)
	}
	// The key excludes the level so a lower level is a subset of a higher one.
	ctx := Context{Seed: opts.Seed, Key: key, Level: level, Other: otherFields(src, opts.Others)}
	out := op.Apply(src.Fields, ctx)
	if reflect.DeepEqual(out, src.Fields) || !projects(out) {
		return Case{}, false
	}
	child := Case{
		ID: id, ParentID: base.ID, Kind: kindOf(op), Operator: op.Name, Stage: op.Stage, Source: src.Name, Boundary: src.Boundary,
		Expect: expectAt(op, level), Criteria: op.Criteria, Band: op.Band, Level: level, Seed: opts.Seed,
		Golden: src.Golden, HeldOut: src.HeldOut, Fields: out, Digest: Digest(out),
	}
	return child, true
}

func kindOf(op Operator) Kind {
	switch {
	case op.Neutral:
		return KindNeutral
	case len(op.Levels) > 0:
		return KindBoundary
	default:
		return KindSynthetic
	}
}

// expectAt is the expectation of a graded level: only the full defect must be
// rejected; a partial one must merely score lower than its parent.
func expectAt(op Operator, level float64) Expect {
	if len(op.Levels) > 0 && level < 1 && op.Expect == ExpectReject {
		return ExpectDrop
	}
	return op.Expect
}

func projects(fields Fields) bool {
	for _, value := range fields {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

// otherFields picks the fields of the next other source by digest order, so the
// choice is stable and never the source itself.
func otherFields(src Source, others []Source) Fields {
	var candidates []Source
	for _, other := range others {
		if other.Digest != src.Digest && other.Boundary == src.Boundary {
			candidates = append(candidates, other)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Digest < candidates[j].Digest })
	index := sort.Search(len(candidates), func(i int) bool { return candidates[i].Digest > src.Digest })
	return candidates[index%len(candidates)].Fields
}

// NaturalNegative builds the case of an amendment's before snapshot: a package a
// human asked to revise. It is paired with the current package as its parent.
func NaturalNegative(parent Case, sequence int, fields Fields) Case {
	return Case{
		ID: fmt.Sprintf("%s~before-%d", parent.ID, sequence), ParentID: parent.ID, Kind: KindNatural, Operator: "amendment_before",
		Source: parent.Source, Boundary: parent.Boundary, Expect: ExpectReject, Band: BandMajor, Golden: parent.Golden, HeldOut: parent.HeldOut,
		Fields: fields, Digest: Digest(fields),
	}
}
