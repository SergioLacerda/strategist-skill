package application

import (
	"regexp"
	"strings"
)

var (
	releaseVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	aheadVersionRe   = regexp.MustCompile(`^(\d+\.\d+\.\d+)(-\d+-g[0-9a-f]+)?(-dirty)?$`)
)

// DisplayVersion renders a raw build version for the public CLI. Release
// builds display their semver; local builds retain the base release and a plus
// marker; unknown values are explicit development builds.
func DisplayVersion(raw string) string {
	v := strings.TrimLeft(strings.TrimSpace(raw), "vV")
	if releaseVersionRe.MatchString(v) {
		return "V" + v
	}
	if m := aheadVersionRe.FindStringSubmatch(v); m != nil {
		return "V" + m[1] + "+"
	}
	return "Vdev"
}
