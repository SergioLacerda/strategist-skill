package calibration

import "math"

// Confidence levels used for every interval and bound.
const (
	// TwoSidedAlpha gives the displayed 95 percent two-sided interval.
	TwoSidedAlpha = 0.025
	// OneSidedAlpha gives the 95 percent one-sided bound used for risk decisions.
	OneSidedAlpha = 0.05
)

// Rate is an observed proportion with its exact (Clopper-Pearson) interval and
// the one-sided upper bound used to judge a risk limit.
type Rate struct {
	K     int     `json:"k"`
	N     int     `json:"n"`
	Value float64 `json:"value"`
	Low   float64 `json:"low"`
	High  float64 `json:"high"`
	Upper float64 `json:"upper_95"`
	// Effective is the sample size the bounds stand on when the cases are not
	// independent (mutants of one parent): 0 means the bounds use N as it is.
	Effective int `json:"effective_n,omitempty"`
}

// NewRate computes the rate k of n. An empty sample has no value: N is 0 and the
// bounds are the vacuous 0 and 1.
func NewRate(k, n int) Rate {
	if n == 0 {
		return Rate{High: 1, Upper: 1}
	}
	return Rate{
		K: k, N: n, Value: float64(k) / float64(n),
		Low: lowerBound(k, n, TwoSidedAlpha), High: upperBound(k, n, TwoSidedAlpha), Upper: upperBound(k, n, OneSidedAlpha),
	}
}

// binomialCDF is P(X <= k) for X ~ Binomial(n, p), through log-gamma so it is
// stable for the sample sizes a calibration uses.
func binomialCDF(k, n int, p float64) float64 {
	if p <= 0 {
		return 1
	}
	if p >= 1 {
		if k >= n {
			return 1
		}
		return 0
	}
	total := 0.0
	for i := 0; i <= k; i++ {
		total += math.Exp(logChoose(n, i) + float64(i)*math.Log(p) + float64(n-i)*math.Log1p(-p))
	}
	return math.Min(total, 1)
}

func logChoose(n, k int) float64 {
	a, _ := math.Lgamma(float64(n + 1))
	b, _ := math.Lgamma(float64(k + 1))
	c, _ := math.Lgamma(float64(n - k + 1))
	return a - b - c
}

// upperBound is the largest p still consistent with at most k successes at the
// given tail probability: P(X <= k | p) = alpha.
func upperBound(k, n int, alpha float64) float64 {
	if k >= n {
		return 1
	}
	return bisect(func(p float64) bool { return binomialCDF(k, n, p) > alpha })
}

// lowerBound is the smallest p consistent with at least k successes: P(X >= k | p) = alpha.
func lowerBound(k, n int, alpha float64) float64 {
	if k <= 0 {
		return 0
	}
	return bisect(func(p float64) bool { return 1-binomialCDF(k-1, n, p) < alpha })
}

// bisect finds the boundary of a predicate that is true below it and false above.
func bisect(below func(float64) bool) float64 {
	low, high := 0.0, 1.0
	for i := 0; i < 60; i++ {
		mid := (low + high) / 2
		if below(mid) {
			low = mid
		} else {
			high = mid
		}
	}
	return (low + high) / 2
}

// MinSample is the smallest sample whose one-sided upper bound at zero observed
// errors is within the limit: below it no result can certify the limit.
func MinSample(limit float64) int {
	for n := 1; n < 100000; n++ {
		if upperBound(0, n, OneSidedAlpha) <= limit {
			return n
		}
	}
	return 100000
}

// NewClusterRate is NewRate for cases that are not independent. The observed
// value stays k of n, but the interval and the upper bound are computed on the
// effective sample size neff (the design-effect reduction of n), so many mutants
// of few parents are not read as that many independent trials.
func NewClusterRate(k, n int, neff float64) Rate {
	if n == 0 || neff >= float64(n) {
		return NewRate(k, n)
	}
	effective := max(int(math.Round(neff)), 1)
	scaled := min(int(math.Ceil(float64(k)*neff/float64(n))), effective)
	bounds := NewRate(scaled, effective)
	rate := NewRate(k, n)
	rate.Low, rate.High, rate.Upper, rate.Effective = bounds.Low, bounds.High, bounds.Upper, effective
	return rate
}

// EffectiveSample is n reduced by the design effect 1 + (m-1)*rho for clusters of
// mean size m, where rho is the assumed intra-cluster correlation. With rho 0
// the cases count as independent; with rho 1 only the clusters count.
func EffectiveSample(n, clusters int, rho float64) float64 {
	if n == 0 || clusters == 0 {
		return 0
	}
	mean := float64(n) / float64(clusters)
	return float64(n) / (1 + (mean-1)*rho)
}
