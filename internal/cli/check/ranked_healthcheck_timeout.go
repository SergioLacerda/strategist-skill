package check

import (
	"os"
	"time"
)

const (
	// rankedHealthcheckTimeoutEnv lets an operator raise the ranked runtime
	// healthcheck limit (for example on a machine whose antivirus scans a freshly
	// extracted node.exe on first launch).
	rankedHealthcheckTimeoutEnv = "STRATEGIST_RANKED_HEALTHCHECK_TIMEOUT"

	// defaultRankedHealthcheckTimeout is generous on purpose: the Node cold path
	// measured about 0.8 s on Linux, and the Windows first launch is unmeasured.
	defaultRankedHealthcheckTimeout = 30 * time.Second

	maxRankedHealthcheckTimeout = 10 * time.Minute
)

// rankedHealthcheckTimeout returns the configured limit, falling back to the
// default for an unset, unparseable, non-positive or absurdly large value.
func rankedHealthcheckTimeout() time.Duration {
	raw := os.Getenv(rankedHealthcheckTimeoutEnv)
	if raw == "" {
		return defaultRankedHealthcheckTimeout
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 || d > maxRankedHealthcheckTimeout {
		return defaultRankedHealthcheckTimeout
	}
	return d
}
