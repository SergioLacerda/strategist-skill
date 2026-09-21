package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ReasonRankedRuntimeVersionSkew is the cataloged, non-blocking reason code for
// a host executable whose version differs from the provider contract's pin.
const ReasonRankedRuntimeVersionSkew = "ranked_runtime_version_skew"

// ReasonRankedRuntimeHealthcheckTimeout is the blocking reason code for a ranked
// runtime that did not answer its healthcheck within the configured limit.
const ReasonRankedRuntimeHealthcheckTimeout = "ranked_runtime_healthcheck_timeout"

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

// VersionAtLeast reports whether an exact MAJOR.MINOR.PATCH version satisfies
// an exact minimum. Invalid versions fail closed.
func VersionAtLeast(observed, minimum string) bool {
	got, ok := parseExactVersion(observed)
	if !ok {
		return false
	}
	want, ok := parseExactVersion(minimum)
	if !ok {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return got[i] > want[i]
		}
	}
	return true
}

func parseExactVersion(version string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(version, ".")
	if len(parts) != len(out) {
		return out, false
	}
	for i, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return out, false
		}
		out[i] = value
	}
	return out, true
}

// RankedRuntimeNodeVersionSkewMessage explains that a host Node satisfies the
// embedded bundle's minimum but is not the version the provider contract
// certified. It is advisory: host_node mode deliberately runs on the client's
// own Node, so the difference is reported rather than blocked.
func RankedRuntimeNodeVersionSkewMessage(provider, pin, observed string) string {
	if observed == "" {
		observed = "unreadable"
	}
	return fmt.Sprintf("reason=ranked_runtime_version_skew: Ranked provider %q was certified against Node %s but this host runs Node %s; "+
		"behavior may differ from what was certified — install Node %s, or use a release binary whose embedded payload carries the certified runtime",
		provider, pin, observed, pin)
}
