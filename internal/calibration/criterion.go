package calibration

import "sort"

// Criterion calibration judges each question of an atomic profile on its own: does its
// answer separate the golden parents from the defects that target it? The aggregate (the
// lowest answer) cannot be certified when real packages have mediocre baselines on some
// criterion, yet a single criterion can be reliable enough to raise a hint. The rule is
// pre-registered here (rule criteria-1, in criterion_rule.go) and is read out of fold, like the
// aggregate under gates-3. It does not replace the aggregate: it says which hints are worth raising.
const (
	// CriterionFolds is the grouped cross-validation used to judge each criterion.
	CriterionFolds = 5
	// MinCriterionDetection is the least one-sided lower bound of the rate at which a
	// criterion's threshold must catch the defects that target it.
	MinCriterionDetection = 0.50
)

// Criterion statuses.
const (
	CriterionReliable    = "reliable"
	CriterionUnreliable  = "unreliable"
	CriterionNotMeasured = "not_measured"
)

// CriterionResult is the reading of one criterion.
type CriterionResult struct {
	Criterion string   `json:"criterion"`
	Operators []string `json:"operators"`
	Golden    int      `json:"golden_parents"`
	Defects   int      `json:"defects"`
	AUC       float64  `json:"auc"`
	// Threshold flags a package when the criterion's answer is below it: the lowest golden
	// answer over the whole corpus, so that no golden parent is flagged in the data it was read from.
	Threshold float64 `json:"threshold"`
	// FalseFlag and Detection are read out of fold: the golden parents flagged and the targeted
	// defects caught by a threshold chosen without their fold.
	FalseFlag Rate   `json:"false_flag"`
	Detection Rate   `json:"detection"`
	Status    string `json:"status"`
	Note      string `json:"note,omitempty"`
}

type criterionData struct {
	golden, defects []CaseStat
}

// criterionScore is the mean answer of a criterion over a case's runs.
func criterionScore(stat CaseStat, criterion string) (float64, bool) {
	var scores []float64
	for _, obs := range stat.Observations {
		if score, ok := obs.Scores[criterion]; ok {
			scores = append(scores, score)
		}
	}
	if len(scores) == 0 {
		return 0, false
	}
	return Mean(scores), true
}

func scoresFor(stats []CaseStat, criterion string) []float64 {
	var out []float64
	for _, stat := range stats {
		if score, ok := criterionScore(stat, criterion); ok {
			out = append(out, score)
		}
	}
	return out
}

// targets groups the defects by the criteria their operator targets.
func targets(stats []CaseStat) (map[string]*criterionData, []string) {
	data := map[string]*criterionData{}
	golden := positives(stats)
	for _, stat := range defects(stats) {
		addTargets(data, golden, stat)
	}
	names := make([]string, 0, len(data))
	for name := range data {
		names = append(names, name)
	}
	sort.Strings(names)
	return data, names
}

func addTargets(data map[string]*criterionData, golden []CaseStat, stat CaseStat) {
	op, ok := ByName(stat.Operator)
	if !ok {
		return
	}
	for _, name := range op.Criteria {
		if data[name] == nil {
			data[name] = &criterionData{golden: golden}
		}
		data[name].defects = append(data[name].defects, stat)
	}
}

// CriterionCalibration reads every criterion that some operator targets; nil when no
// observation carries per-criterion answers (a profile with an aggregate answer).
func CriterionCalibration(stats []CaseStat, policy Policy, limits Limits) []CriterionResult {
	data, names := targets(stats)
	var out []CriterionResult
	for _, name := range names {
		if len(scoresFor(data[name].golden, name)) == 0 {
			continue
		}
		out = append(out, readCriterion(name, *data[name], policy, limits))
	}
	return out
}

type flagTally struct{ flagged, golden, caught, defects int }

func (t *flagTally) add(goldenTest, defectTest []CaseStat, criterion string, threshold float64) {
	for _, score := range scoresFor(goldenTest, criterion) {
		t.golden++
		if score < threshold {
			t.flagged++
		}
	}
	for _, score := range scoresFor(defectTest, criterion) {
		t.defects++
		if score < threshold {
			t.caught++
		}
	}
}

func splitFold(stats []CaseStat, fold int) (train, test []CaseStat) {
	return foldSplit(stats, fold, CriterionFolds)
}

func readCriterion(name string, d criterionData, policy Policy, limits Limits) CriterionResult {
	goldenScores, defectScores := scoresFor(d.golden, name), scoresFor(d.defects, name)
	res := CriterionResult{Criterion: name, Operators: operatorsOf(d.defects), Golden: len(goldenScores), Defects: len(defectScores)}
	res.AUC = AUC(goldenScores, defectScores)
	res.Threshold = lowest(goldenScores)
	var tally flagTally
	for fold := 0; fold < CriterionFolds; fold++ {
		trainGolden, testGolden := splitFold(d.golden, fold)
		_, testDefects := splitFold(d.defects, fold)
		tally.add(testGolden, testDefects, name, lowest(scoresFor(trainGolden, name)))
	}
	res.FalseFlag = NewRate(tally.flagged, tally.golden)
	res.Detection = NewClusterRate(tally.caught, tally.defects, EffectiveSample(tally.defects, parentCount(d.defects), policy.ClusterRho))
	res.Status, res.Note = judgeCriterion(res, policy, limits)
	return res
}

func lowest(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	low := values[0]
	for _, v := range values[1:] {
		low = min(low, v)
	}
	return low
}

func operatorsOf(stats []CaseStat) []string {
	seen := map[string]bool{}
	var names []string
	for _, stat := range stats {
		if !seen[stat.Operator] {
			seen[stat.Operator] = true
			names = append(names, stat.Operator)
		}
	}
	sort.Strings(names)
	return names
}
