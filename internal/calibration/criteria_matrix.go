package calibration

import "sort"

// GoldenRow labels the row of the intact packages in a criteria matrix.
const GoldenRow = "golden"

// CriterionRow is the mean answer of every criterion over the runs of one group of
// cases: the intact packages or the cases of one operator.
type CriterionRow struct {
	Group string             `json:"group"`
	Runs  int                `json:"runs"`
	Means map[string]float64 `json:"means"`
}

// CriteriaMatrix says which criterion of an atomic profile moved for which defect,
// so a defect the profile misses can be traced to the question that should have caught it.
type CriteriaMatrix struct {
	Criteria []string       `json:"criteria"`
	Rows     []CriterionRow `json:"rows"`
}

type matrixCell struct {
	sums map[string]float64
	runs int
}

func groupOf(stat CaseStat) string {
	if stat.Operator == "" {
		return GoldenRow
	}
	return stat.Operator
}

// BuildCriteriaMatrix averages the per-criterion answers of the observations that
// carry them; nil when none does (a profile with an aggregate answer).
func BuildCriteriaMatrix(stats []CaseStat) *CriteriaMatrix {
	cells, names := map[string]*matrixCell{}, map[string]bool{}
	for _, stat := range stats {
		for _, obs := range stat.Observations {
			addScores(cells, names, groupOf(stat), obs.Scores)
		}
	}
	if len(cells) == 0 {
		return nil
	}
	matrix := &CriteriaMatrix{Criteria: sortedNames(names)}
	for group, cell := range cells {
		row := CriterionRow{Group: group, Runs: cell.runs, Means: map[string]float64{}}
		for name, sum := range cell.sums {
			row.Means[name] = sum / float64(cell.runs)
		}
		matrix.Rows = append(matrix.Rows, row)
	}
	sort.Slice(matrix.Rows, func(i, j int) bool { return rowLess(matrix.Rows[i].Group, matrix.Rows[j].Group) })
	return matrix
}

func addScores(cells map[string]*matrixCell, names map[string]bool, group string, scores map[string]float64) {
	if len(scores) == 0 {
		return
	}
	cell := cells[group]
	if cell == nil {
		cell = &matrixCell{sums: map[string]float64{}}
		cells[group] = cell
	}
	cell.runs++
	for name, score := range scores {
		cell.sums[name] += score
		names[name] = true
	}
}

func sortedNames(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// rowLess puts the golden row first and the operators after it by name.
func rowLess(a, b string) bool {
	if a == GoldenRow || b == GoldenRow {
		return a == GoldenRow && b != GoldenRow
	}
	return a < b
}
