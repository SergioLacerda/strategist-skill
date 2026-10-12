package calibration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRate_ZeroErrorsMatchesTheRuleOfThree(t *testing.T) {
	rate := NewRate(0, 59)
	assert.InDelta(t, 3.0/59, rate.Upper, 0.003)
	assert.LessOrEqual(t, rate.Upper, 0.05)
	assert.Greater(t, NewRate(0, 58).Upper, 0.05)
}

func TestMinSample_ForTheDefaultLimits(t *testing.T) {
	assert.Equal(t, 59, MinSample(0.05))
	assert.Equal(t, 19, MinSample(0.15))
}

func TestRate_KnownExactInterval(t *testing.T) {
	rate := NewRate(5, 20)
	assert.InDelta(t, 0.25, rate.Value, 1e-9)
	assert.InDelta(t, 0.0866, rate.Low, 0.001)
	assert.InDelta(t, 0.4910, rate.High, 0.001)
}

func TestRate_EmptyAndFullSamples(t *testing.T) {
	empty := NewRate(0, 0)
	assert.Equal(t, 0, empty.N)
	assert.InDelta(t, 1.0, empty.Upper, 1e-9)
	full := NewRate(10, 10)
	assert.InDelta(t, 1.0, full.Value, 1e-9)
	assert.InDelta(t, 1.0, full.High, 1e-9)
}

func TestAUC(t *testing.T) {
	assert.InDelta(t, 1.0, AUC([]float64{0.9, 0.8}, []float64{0.1, 0.2}), 1e-9)
	assert.InDelta(t, 0.0, AUC([]float64{0.1}, []float64{0.9}), 1e-9)
	assert.InDelta(t, 0.5, AUC([]float64{0.5}, []float64{0.5}), 1e-9)
	assert.InDelta(t, 0.5, AUC(nil, []float64{0.5}), 1e-9)
}

func TestBrierAndECE(t *testing.T) {
	items := []Scored{{0.9, 1}, {0.1, 0}}
	assert.InDelta(t, 0.01, Brier(items), 1e-9)
	assert.InDelta(t, 0.1, ECE(items), 1e-9)
	assert.InDelta(t, 0.0, Brier(nil), 1e-9)
	assert.InDelta(t, 0.0, ECE(nil), 1e-9)
}

func TestStdDevMeanPercentile(t *testing.T) {
	assert.InDelta(t, 1.0, StdDev([]float64{1, 3}), 1e-9)
	assert.InDelta(t, 0.0, StdDev([]float64{4}), 1e-9)
	assert.InDelta(t, 2.0, Mean([]float64{1, 3}), 1e-9)
	assert.InDelta(t, 4.0, Percentile([]float64{1, 2, 3, 4, 5}, 0.75), 1e-9)
	assert.InDelta(t, 0.0, Percentile(nil, 0.5), 1e-9)
}
