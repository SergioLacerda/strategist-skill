package dojo

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/stretchr/testify/assert"
)

func TestWhatIfLabelsIncludesEveryOverride(t *testing.T) {
	labels := whatIfLabels(CheckInput{
		WhatIfLocalScope: []string{"lint", "tests"},
		JudgeOnly:        true,
		ExcludeCriteria:  []string{"clarity"},
	})
	assert.Equal(t, []string{"local_scope:lint", "local_scope:tests", "local_scope:none", "criterion:clarity"}, labels)
	assert.Empty(t, whatIfLabels(CheckInput{}))
}

func TestCalibrationItemsRenderCriteriaAndCrossValidation(t *testing.T) {
	threshold := 0.8
	report := calibration.Report{
		Criteria: &calibration.CriteriaMatrix{Rows: []calibration.CriterionRow{{
			Group: "golden", Runs: 2, Means: map[string]float64{"z": 0.4, "a": 0.9},
		}}},
		CrossValidation: &calibration.CrossValidation{
			Folds: 3, Thresholds: []float64{threshold}, Infeasible: 1,
		},
		CriterionCalibration: []calibration.CriterionResult{{
			Criterion: "clarity", Status: calibration.CriterionReliable, AUC: 0.9,
			Golden: 2, Defects: 1, Operators: []string{"mutation"}, Threshold: 0.7,
			Note: "small sample",
		}},
	}

	items := extraItems(report)
	if assert.Len(t, items, 3) {
		assert.Contains(t, items[0].Detail, "3 folds")
		assert.Contains(t, items[1].Detail, "small sample")
		assert.Contains(t, items[2].Detail, "a 0.90, z 0.40")
	}
	assert.Nil(t, criteriaItems(calibration.Report{}))
	assert.Nil(t, crossValidationItems(calibration.Report{}))
}
