package calibration

import (
	"math"
	"sort"
)

// AUC is the probability that a random positive scores above a random negative
// (ties count half). It is 0.5 when there is nothing to compare.
func AUC(positives, negatives []float64) float64 {
	if len(positives) == 0 || len(negatives) == 0 {
		return 0.5
	}
	wins := 0.0
	for _, p := range positives {
		for _, n := range negatives {
			wins += beats(p, n)
		}
	}
	return wins / float64(len(positives)*len(negatives))
}

func beats(p, n float64) float64 {
	switch {
	case p > n:
		return 1
	case p == n:
		return 0.5
	}
	return 0
}

// Scored is a case-level probability of accept with its true label (1 positive, 0 defect).
type Scored struct {
	Score float64
	Label float64
}

// Brier is the mean squared error of the probabilities.
func Brier(items []Scored) float64 {
	if len(items) == 0 {
		return 0
	}
	sum := 0.0
	for _, item := range items {
		sum += (item.Score - item.Label) * (item.Score - item.Label)
	}
	return sum / float64(len(items))
}

// ECE is the expected calibration error over ten equal-width bins.
func ECE(items []Scored) float64 {
	const bins = 10
	var count [bins]int
	var score, label [bins]float64
	for _, item := range items {
		bin := min(int(item.Score*bins), bins-1)
		count[bin]++
		score[bin] += item.Score
		label[bin] += item.Label
	}
	total := 0.0
	for i := 0; i < bins; i++ {
		if count[i] > 0 {
			total += float64(count[i]) / float64(len(items)) * math.Abs(score[i]-label[i]) / float64(count[i])
		}
	}
	return total
}

// StdDev is the population standard deviation.
func StdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := Mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return math.Sqrt(sum / float64(len(values)))
}

// Mean is the arithmetic mean, 0 for an empty slice.
func Mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// Percentile is the nearest-rank percentile of a copy of values.
func Percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	rank := int(math.Ceil(p*float64(len(sorted)))) - 1
	return sorted[max(rank, 0)]
}
