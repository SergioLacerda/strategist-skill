package calibration

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// Boundaries a catalog can target.
const (
	BoundaryPackage = "package"
	BoundaryRanger  = "ranger"
)

// Context is what an operator may use besides the fields: the run seed, a stable
// key for deterministic choices, the graded level in (0,1] and another package's
// fields for cross-package operators.
type Context struct {
	Seed  int64
	Key   string
	Level float64
	Other Fields
}

// Operator injects one known defect (or one neutral change) into a projection.
// Applies is its precondition: an operator that does not apply is skipped, never
// forced. Apply must not mutate its input.
type Operator struct {
	Name     string
	Stage    int
	Boundary string
	Band     Band
	Expect   Expect
	Criteria []string
	// Levels, when set, makes the operator graded: one case per level.
	Levels  []float64
	Neutral bool
	Applies func(Fields) bool
	Apply   func(Fields, Context) Fields
}

// Catalog returns the operators of one boundary, ordered by name.
func Catalog(boundary string) []Operator {
	var out []Operator
	for _, op := range allOperators() {
		if op.Boundary == boundary || op.Boundary == BoundaryAny {
			out = append(out, op)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func allOperators() []Operator {
	var all []Operator
	all = append(all, packageOperators()...)
	all = append(all, semanticPackageOperators()...)
	all = append(all, rangerOperators()...)
	all = append(all, neutralOperators()...)
	return all
}

// CatalogVersion identifies the operator set: a digest of every operator's
// name, stage, boundary, expectation and levels, so a changed catalog is a new version.
func CatalogVersion() string {
	var text strings.Builder
	ops := allOperators()
	sort.Slice(ops, func(i, j int) bool { return ops[i].Name < ops[j].Name })
	for _, op := range ops {
		fmt.Fprintf(&text, "%s|%d|%s|%s|%s|%v|%v\n", op.Name, op.Stage, op.Boundary, op.Band, op.Expect, op.Levels, op.Criteria)
	}
	sum := sha256.Sum256([]byte(text.String()))
	return fmt.Sprintf("catalog-%x", sum[:4])
}

// ByName finds an operator in the full catalog.
func ByName(name string) (Operator, bool) {
	for _, op := range allOperators() {
		if op.Name == name {
			return op, true
		}
	}
	return Operator{}, false
}
