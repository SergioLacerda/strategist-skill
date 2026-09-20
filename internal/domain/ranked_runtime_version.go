package domain

import (
	"fmt"
	"regexp"
)

// ReasonRankedRuntimeVersionSkew is the cataloged, non-blocking reason code for
// a host executable whose version differs from the provider contract's pin.
const ReasonRankedRuntimeVersionSkew = "ranked_runtime_version_skew"

// ReasonRankedRuntimePinMismatch is the blocking reason code for a private
// runtime whose component versions differ from the provider contract's pins: a
// build defect, never accepted.
const ReasonRankedRuntimePinMismatch = "ranked_runtime_pin_mismatch"

var reportedVersion = regexp.MustCompile(`\d+\.\d+\.\d+`)
var pinnedVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// ParseReportedVersion extracts MAJOR.MINOR.PATCH from a CLI's --version
// output (which may carry a "v" prefix, a program name, or trailing lines).
// It returns "" when no version is present.
func ParseReportedVersion(output []byte) string {
	return reportedVersion.FindString(string(output))
}

// VersionSkew reports whether the observed version fails to match the pin. An
// empty pin means the contract does not pin, so there is nothing to compare.
func VersionSkew(pin, observed string) bool {
	return pin != "" && pin != observed
}

// RankedRuntimeVersionSkewMessage explains a host/pin version difference.
func RankedRuntimeVersionSkewMessage(provider, pin, observed string) string {
	if observed == "" {
		observed = "unreadable"
	}
	return fmt.Sprintf("reason=ranked_runtime_version_skew: Ranked provider %q is pinned to version %s but the host executable reports %s; "+
		"install the pinned version so behavior matches what was certified",
		provider, pin, observed)
}
