package authorization

import (
	"encoding/json"
	"fmt"
)

func finish(report Report) Report {
	for _, dimension := range report.Dimensions {
		if !dimension.Required {
			continue
		}
		if report, done := terminalDimensionDecision(report, dimension); done {
			return report
		}
	}
	return allowedReport(report)
}

func terminalDimensionDecision(report Report, dimension Dimension) (Report, bool) {
	switch dimension.Status {
	case "blocked":
		report.Decision, report.ExitClass = "blocked", "blocked"
	case "stale":
		report.Decision, report.ExitClass = "blocked", "stale"
	case "allowed", "ready":
		return report, false
	default:
		report.Decision, report.ExitClass = "denied", "denied"
	}
	report.ReasonCode = dimension.ReasonCode
	return report, true
}

func allowedReport(report Report) Report {
	report.Decision = "allowed"
	report.ReasonCode = "authorization_dimensions_satisfied"
	report.ExitClass = "success"
	return report
}

func authorizationError(report Report) error {
	if report.Decision == "allowed" {
		return nil
	}
	if report.ExitClass == "blocked" {
		return fmt.Errorf("%w: %s", ErrBlocked, report.ReasonCode)
	}
	if report.ExitClass == "stale" {
		return fmt.Errorf("%w: %s", ErrStale, report.ReasonCode)
	}
	return fmt.Errorf("%w: %s", ErrDenied, report.ReasonCode)
}

// JSON serializes the report as indented, stable JSON.
func (r Report) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal authorization report: %w", err)
	}
	return data, nil
}
