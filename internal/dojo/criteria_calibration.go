package dojo

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Defaults a calibration case may leave out. The risk policy has none: it is
// stated in the case, never implied.
const (
	DefaultRepetitions    = 3
	DefaultHeldOutPercent = 25
	DefaultMaxAgeDays     = 90
)

func validateCalibration(c domain.DojoCriteria) []string {
	switch {
	case c.Kind == "" && c.Calibration == nil:
		return nil
	case c.Kind != domain.KindJudgeCalibration:
		return []string{fmt.Sprintf("kind %q is not supported (only %q)", c.Kind, domain.KindJudgeCalibration)}
	case c.Calibration == nil:
		return []string{"calibration: block is required for kind judge_calibration"}
	}
	cal := *c.Calibration
	var errs []string
	errs = append(errs, validateCalibrationScope(cal)...)
	errs = append(errs, validateCalibrationLimits(cal)...)
	errs = append(errs, validateCalibrationNames(cal)...)
	return errs
}

func validateCalibrationScope(cal domain.DojoCalibration) []string {
	var errs []string
	if cal.Boundary != calibration.BoundaryPackage && cal.Boundary != calibration.BoundaryRanger && cal.Boundary != calibration.BoundaryPairName {
		errs = append(errs, fmt.Sprintf("calibration: boundary %q must be package, ranger or %s", cal.Boundary, calibration.BoundaryPairName))
	}
	if cal.Layer != "" && cal.Layer != "judge" && cal.Layer != "integration" && cal.Layer != "live" && cal.Layer != "pair" {
		errs = append(errs, fmt.Sprintf("calibration: layer %q must be judge, integration, live or pair", cal.Layer))
	}
	if !calibration.KnownProfile(cal.Profile) {
		errs = append(errs, fmt.Sprintf("calibration: profile %q must be %s, %s or %s", cal.Profile, calibration.ProfileV1, calibration.ProfileV2, calibration.ProfileAtomicV1))
	}
	if cal.Provider != "" && cal.Provider != "jev" {
		errs = append(errs, fmt.Sprintf("calibration: provider %q is not supported", cal.Provider))
	}
	return errs
}

func validateCalibrationLimits(cal domain.DojoCalibration) []string {
	var errs []string
	policy := cal.RiskPolicy
	if policy.MaxFalseApproval <= 0 || policy.MaxFalseApproval >= 1 || policy.MaxFalseRejection <= 0 || policy.MaxFalseRejection >= 1 {
		errs = append(errs, "calibration: risk_policy max_false_approval and max_false_rejection must be stated, each between 0 and 1")
	}
	if cal.Stage < 0 || cal.Stage > 2 || cal.Repetitions < 0 || cal.Repetitions > 20 || cal.Budget < 0 || cal.MaxAgeDays < 0 {
		errs = append(errs, "calibration: stage must be 0 to 2, repetitions 0 to 20, budget and max_age_days not negative")
	}
	if cal.Corpus.HeldOutPercent < 0 || cal.Corpus.HeldOutPercent > 90 {
		errs = append(errs, "calibration: corpus.held_out_percent must be 0 to 90")
	}
	return errs
}

func validateCalibrationNames(cal domain.DojoCalibration) []string {
	var errs []string
	for _, name := range append(append([]string(nil), cal.Operators...), cal.LocalScope...) {
		if _, ok := calibration.ByName(name); !ok {
			errs = append(errs, fmt.Sprintf("calibration: unknown operator %q", name))
		}
	}
	if !calibration.KnownGateRule(cal.GateRule) {
		errs = append(errs, fmt.Sprintf("calibration: gate_rule %q must be %s, %s or %s", cal.GateRule, calibration.RuleGates1, calibration.RuleGates2, calibration.RuleGates3))
	}
	errs = append(errs, validateIncludes(cal)...)
	errs = append(errs, validateSynthetic(cal)...)
	return append(errs, validateCalibrationPaths(cal)...)
}

func validateIncludes(cal domain.DojoCalibration) []string {
	var errs []string
	for _, name := range cal.Corpus.Include {
		if !validScenarioName(name) {
			errs = append(errs, fmt.Sprintf("calibration: corpus.include %q is not a package name", name))
		}
	}
	return errs
}

func validateSynthetic(cal domain.DojoCalibration) []string {
	if !cal.Corpus.Synthetic || cal.Layer == "pair" {
		return nil
	}
	if len(cal.Corpus.Roots) == 0 {
		return []string{"calibration: corpus.synthetic needs corpus.roots"}
	}
	var errs []string
	for _, root := range cal.Corpus.Roots {
		if !strings.HasPrefix(filepath.ToSlash(filepath.Clean(root)), "dojo/") {
			errs = append(errs, fmt.Sprintf("calibration: synthetic root %q must be under dojo/", root))
		}
	}
	return errs
}

func validateCalibrationPaths(cal domain.DojoCalibration) []string {
	var errs []string
	for _, root := range append([]string{cal.Output}, cal.Corpus.Roots...) {
		if root != "" && (isUnsafeRelPath(root) || strings.Contains(filepath.ToSlash(root), "..")) {
			errs = append(errs, fmt.Sprintf("calibration: path %q must stay inside the workspace", root))
		}
	}
	return errs
}

// Repetitions resolves the repetitions of a case; 0 means the default.
func Repetitions(cal domain.DojoCalibration) int {
	return defaultInt(cal.Repetitions, DefaultRepetitions)
}

// HeldOutPercent resolves the held-out share; 0 means the default.
func HeldOutPercent(cal domain.DojoCalibration) int {
	return defaultInt(cal.Corpus.HeldOutPercent, DefaultHeldOutPercent)
}

// MaxAgeDays resolves the calibration age limit; 0 means the default.
func MaxAgeDays(cal domain.DojoCalibration) int { return defaultInt(cal.MaxAgeDays, DefaultMaxAgeDays) }

func defaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}
