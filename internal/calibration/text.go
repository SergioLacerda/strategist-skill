package calibration

import (
	"regexp"
	"strings"
)

var (
	taskLine    = regexp.MustCompile(`^(\s*- \[[ xX]\] \d+(?:\.\d+)* )\[([a-z_]+)\]( .*)$`)
	verifyTail  = regexp.MustCompile(`(?i)[;.,]?\s*\bverify\b.*$`)
	pathToken   = regexp.MustCompile("`[^`]+`|\\b[\\w./-]+\\.(?:go|md|yaml|yml|json|tsv)(?::\\d+)?\\b")
	manySpaces  = regexp.MustCompile(`[ \t]{2,}`)
	manyBreaks  = regexp.MustCompile(`\n{3,}`)
	trailingWS  = regexp.MustCompile(`(?m)[ \t]+$`)
	mustWord    = regexp.MustCompile(`\bMUST\b`)
	checkedTask = regexp.MustCompile(`- \[ \]`)
)

func lines(text string) []string { return strings.Split(text, "\n") }

func join(parts []string) string { return strings.Join(parts, "\n") }

// taskIndexes returns the line indexes that are task lines.
func taskIndexes(text string) []int {
	var idx []int
	for i, line := range lines(text) {
		if taskLine.MatchString(line) {
			idx = append(idx, i)
		}
	}
	return idx
}

// rewriteTasks applies edit to a deterministic fraction of the task lines.
func rewriteTasks(text string, ctx Context, edit func(string) string) string {
	all := lines(text)
	tasks := taskIndexes(text)
	chosen := pickIndexes(ctx.Seed, ctx.Key, len(tasks), ctx.Level)
	for position, index := range tasks {
		if chosen[position] {
			all[index] = edit(all[index])
		}
	}
	return join(all)
}

func hasField(fields Fields, name string) bool {
	return strings.TrimSpace(fields[name]) != ""
}

func with(fields Fields, name, value string) Fields {
	out := fields.clone()
	if strings.TrimSpace(value) == "" {
		delete(out, name)
		return out
	}
	out[name] = value
	return out
}
