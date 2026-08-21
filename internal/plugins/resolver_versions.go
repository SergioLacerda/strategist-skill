package plugins

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func sortedDependencies(dependencies []Dependency) []Dependency {
	out := append([]Dependency(nil), dependencies...)
	sort.Slice(out, func(i, j int) bool {
		left := out[i].Kind + "/" + out[i].ID + "/" + out[i].Constraint
		right := out[j].Kind + "/" + out[j].ID + "/" + out[j].Constraint
		return left < right
	})
	return out
}

func compareCandidate(left, right Candidate) int {
	if cmp := compareVersions(right.Version, left.Version); cmp != 0 {
		return cmp
	}
	if left.Digest < right.Digest {
		return -1
	}
	if left.Digest > right.Digest {
		return 1
	}
	return 0
}

func candidateSatisfiesAll(candidate Candidate, constraints []string) bool {
	for _, constraint := range constraints {
		if !versionSatisfies(candidate.Version, constraint) {
			return false
		}
	}
	return true
}

func versionSatisfies(version, constraint string) bool {
	constraint = normalizeConstraint(constraint)
	if constraint == "" || constraint == "*" {
		return true
	}
	for _, part := range strings.Fields(constraint) {
		if !versionSatisfiesPart(version, part) {
			return false
		}
	}
	return true
}

func versionSatisfiesPart(version, part string) bool {
	switch {
	case strings.HasPrefix(part, ">="):
		return compareVersions(version, strings.TrimPrefix(part, ">=")) >= 0
	case strings.HasPrefix(part, "<"):
		return compareVersions(version, strings.TrimPrefix(part, "<")) < 0
	case strings.HasPrefix(part, "="):
		return compareVersions(version, strings.TrimPrefix(part, "=")) == 0
	default:
		return compareVersions(version, part) == 0
	}
}

func compareVersions(left, right string) int {
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	for i := 0; i < len(leftParts) || i < len(rightParts); i++ {
		var l, r int
		if i < len(leftParts) {
			l = leftParts[i]
		}
		if i < len(rightParts) {
			r = rightParts[i]
		}
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
	}
	return strings.Compare(left, right)
}

func versionParts(version string) []int {
	raw := strings.Split(version, ".")
	parts := make([]int, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			parts = append(parts, 0)
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, n)
	}
	return parts
}

func normalizeConstraint(constraint string) string {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		return "*"
	}
	return constraint
}

func conflictError(key string, constraints []string) error {
	unique := append([]string(nil), constraints...)
	sort.Strings(unique)
	return fmt.Errorf("dependency_conflict: %s constraints=%s", key, strings.Join(unique, ","))
}

func requirementKey(req Requirement) string {
	return req.Kind + ":" + req.ID
}

func candidateKey(candidate Candidate) string {
	return candidate.Kind + ":" + candidate.ID
}

func nodeKey(node domain.PluginLockNode) string {
	return node.Kind + ":" + node.ID
}
