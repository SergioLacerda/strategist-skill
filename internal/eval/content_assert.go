package eval

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// sourceCitationPattern matches file-path-looking tokens ending in a
// documentation/source extension, used by AssertSourceCitations.
var sourceCitationPattern = regexp.MustCompile(`[\w./-]+\.(?:md|go|yaml|yml|txt)\b`)

// evaluateContentAssertions applies the Phase 2 content-assertion types
// (see assertion.go) against content already read by the caller. Assertion
// types outside this switch (equal-state, required-event, artifact-exists,
// scope-includes, scope-excludes) are evaluated elsewhere and are silently
// skipped here.
func evaluateContentAssertions(content string, assertions []Assertion, res *ScenarioResult) {
	for _, a := range assertions {
		//exhaustive:ignore -- only content-assertion types apply here (see doc comment above)
		switch a.Type {
		case AssertContains:
			appendIfViolation(res, checkContains(content, a))
		case AssertNotContains:
			appendIfViolation(res, checkNotContains(content, a))
		case AssertRegex:
			appendIfViolation(res, checkRegex(content, a))
		case AssertMaxTokens:
			appendIfViolation(res, checkMaxTokens(content, a))
		case AssertForbiddenToolCall:
			appendIfViolation(res, checkForbiddenToolCall(content, a))
		case AssertRequiredSections:
			res.Violations = append(res.Violations, checkRequiredSections(content, a)...)
		case AssertSourceCitations:
			appendIfViolation(res, checkSourceCitations(content, a))
		}
	}
}

func appendIfViolation(res *ScenarioResult, v *Violation) {
	if v != nil {
		res.Violations = append(res.Violations, *v)
	}
}

func checkContains(content string, a Assertion) *Violation {
	if strings.Contains(content, a.Value) {
		return nil
	}
	return &Violation{
		AssertionType: a.Type,
		Message:       "content does not contain expected substring",
		Expected:      a.Value,
		Actual:        truncate(content),
	}
}

func checkNotContains(content string, a Assertion) *Violation {
	if !strings.Contains(content, a.Value) {
		return nil
	}
	return &Violation{
		AssertionType: a.Type,
		Message:       "content contains forbidden substring",
		Expected:      "absence of " + a.Value,
		Actual:        a.Value,
	}
}

func checkRegex(content string, a Assertion) *Violation {
	re, err := regexp.Compile(a.Value)
	if err != nil {
		return &Violation{
			AssertionType: a.Type,
			Message:       "invalid regex pattern",
			Expected:      a.Value,
			Actual:        err.Error(),
		}
	}
	if re.MatchString(content) {
		return nil
	}
	return &Violation{
		AssertionType: a.Type,
		Message:       "content does not match pattern",
		Expected:      a.Value,
		Actual:        truncate(content),
	}
}

func checkMaxTokens(content string, a Assertion) *Violation {
	budget, err := strconv.Atoi(a.Value)
	if err != nil {
		return &Violation{
			AssertionType: a.Type,
			Message:       "max-tokens value is not an integer",
			Expected:      a.Value,
			Actual:        err.Error(),
		}
	}
	count := len(strings.Fields(content))
	if count <= budget {
		return nil
	}
	return &Violation{
		AssertionType: a.Type,
		Message:       "content exceeds whitespace-token budget (approximate, not a real tokenizer)",
		Expected:      fmt.Sprintf("<= %d", budget),
		Actual:        strconv.Itoa(count),
	}
}

func checkForbiddenToolCall(content string, a Assertion) *Violation {
	if !strings.Contains(content, a.Value) {
		return nil
	}
	return &Violation{
		AssertionType: a.Type,
		Message:       "content references a forbidden tool/command",
		Expected:      "absence of " + a.Value,
		Actual:        a.Value,
	}
}

func checkRequiredSections(content string, a Assertion) []Violation {
	var out []Violation
	for _, section := range splitCSV(a.Value) {
		if !hasHeading(content, section) {
			out = append(out, Violation{
				AssertionType: a.Type,
				Message:       "required section heading not found",
				Expected:      section,
				Actual:        "absent",
			})
		}
	}
	return out
}

func checkSourceCitations(content string, a Assertion) *Violation {
	minCount, err := strconv.Atoi(a.Value)
	if err != nil {
		return &Violation{
			AssertionType: a.Type,
			Message:       "source-citations value is not an integer",
			Expected:      a.Value,
			Actual:        err.Error(),
		}
	}
	got := countSourceCitations(content)
	if got >= minCount {
		return nil
	}
	return &Violation{
		AssertionType: a.Type,
		Message:       "fewer distinct source citations than required",
		Expected:      fmt.Sprintf(">= %d", minCount),
		Actual:        strconv.Itoa(got),
	}
}

// hasHeading reports whether content contains a markdown heading line
// (starting with one or more '#') whose text contains section.
func hasHeading(content, section string) bool {
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "#") {
			continue
		}
		t = strings.TrimSpace(strings.TrimLeft(t, "#"))
		if strings.Contains(t, section) {
			return true
		}
	}
	return false
}

// countSourceCitations counts distinct source-path-looking tokens in content.
func countSourceCitations(content string) int {
	matches := sourceCitationPattern.FindAllString(content, -1)
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		seen[m] = true
	}
	return len(seen)
}

// splitCSV splits a comma-separated Value into trimmed, non-empty parts.
func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// truncate shortens s for readable violation messages.
func truncate(s string) string {
	const maxLen = 120
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
