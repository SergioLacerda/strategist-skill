package preparerunbook

import (
	"encoding/hex"
	"regexp"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
)

var headingPattern = regexp.MustCompile(`(?m)^##+\s+([^\n]+)\n([^#]*)`)

func matchingLines(text string, terms ...string) []string {
	result := []string{}
	for _, line := range strings.Split(text, "\n") {
		if clean, ok := matchingLine(line, terms); ok {
			result = append(result, clean)
		}
	}
	return unique(result, 8)
}

func matchingLine(line string, terms []string) (string, bool) {
	lower := strings.ToLower(line)
	for _, term := range terms {
		if strings.Contains(lower, term) {
			clean := strings.TrimSpace(strings.TrimLeft(line, "- *`"))
			return clean, clean != ""
		}
	}
	return "", false
}

func firstHeadingText(text string, names ...string) string {
	for _, match := range headingPattern.FindAllStringSubmatch(text, -1) {
		for _, name := range names {
			if strings.EqualFold(strings.TrimSpace(match[1]), name) {
				return strings.TrimSpace(strings.Split(match[2], "\n")[0])
			}
		}
	}
	return ""
}

func firstFrontString(artifacts []artifact, keys ...string) string {
	for _, a := range artifacts {
		for _, key := range keys {
			if value, ok := a.front[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func firstFrontStrings(artifacts []artifact, keys ...string) []string {
	for _, a := range artifacts {
		for _, key := range keys {
			if result := frontStrings(a.front[key]); len(result) > 0 {
				return result
			}
		}
	}
	return nil
}

func frontStrings(value any) []string {
	switch value := value.(type) {
	case []any:
		return nonEmptyStrings(value)
	case string:
		if strings.TrimSpace(value) != "" {
			return []string{strings.TrimSpace(value)}
		}
	}
	return nil
}

func nonEmptyStrings(values []any) []string {
	result := make([]string, 0, len(values))
	for _, item := range values {
		if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}

func signalNames(text string) []string {
	values := runbook.CanonicalSignalsIn(text)
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, string(value))
	}
	sort.Strings(result)
	return result
}

func missingQuestions(missing []string) []string {
	result := make([]string, len(missing))
	for i, v := range missing {
		result[i] = "Provide deterministic evidence for " + v
	}
	return result
}
func comparisonReason(kind ComparisonKind) string {
	switch kind {
	case ComparisonDuplicate:
		return "an equivalent Runbook was found"
	case ComparisonExtension:
		return "an existing Runbook shares signals and the candidate may extend it"
	case ComparisonNovel:
		return "no equivalent or overlapping Runbook was found"
	case ComparisonInsufficient:
		return "comparison evidence is incomplete"
	default:
		return "comparison evidence is incomplete"
	}
}
func hasAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}
func normalize(value string) string { return strings.Join(strings.Fields(strings.ToLower(value)), " ") }
func normalizeList(values []string) string {
	copyValues := append([]string(nil), values...)
	for i := range copyValues {
		copyValues[i] = normalize(copyValues[i])
	}
	sort.Strings(copyValues)
	return strings.Join(copyValues, "|")
}
func unique(values []string, limit int) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}
func slug(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
func shortDigest(value string) string {
	raw := strings.TrimPrefix(value, "sha256:")
	if _, err := hex.DecodeString(raw); err != nil {
		value = slug(value)
		return value[:min(12, len(value))]
	}
	return raw[:min(12, len(raw))]
}
