package calibration

import (
	"regexp"
	"strings"
)

// Real refined packages write their checklists in more than one grammar: numbered or not
// (`1.1`, `A.1`), a bare tag (`[implementation_handoff]`) or `[task_type: implementation_handoff]`,
// and an item may wrap onto indented continuation lines. A task is one checklist item with all its lines.
var (
	taskHeader = regexp.MustCompile(`^(\s*- \[[ xX]\] (?:(?:\d+(?:\.\d+)*|[A-Z]\.\d+(?:\.\d+)*) )?)(.*)$`)
	taskTag    = regexp.MustCompile(`^\[(?:task_type: ?)?([a-z_]+)\]( |$)`)
	// stripTag removes the tag of an item whose header is already known to carry one, keeping the layout of the old grammar.
	stripTag    = regexp.MustCompile(`(?s)^(\s*- \[[ xX]\] (?:(?:\d+(?:\.\d+)*|[A-Z]\.\d+(?:\.\d+)*) )?)\[(?:task_type: ?)?[a-z_]+\]( .*)$`)
	relabelTag  = regexp.MustCompile(`\[((?:task_type: ?)?)implementation_handoff\]`)
	verifyAfter = regexp.MustCompile(`(?is)[;.,]?\s*\bverify\b.*$`)
)

// taskItem is one checklist item: the line range it spans (inclusive) and its type tag, empty when untyped.
type taskItem struct {
	Start, End int
	Type       string
}

// text is the item's lines joined, as they appear in the plan.
func (t taskItem) text(all []string) string { return join(all[t.Start : t.End+1]) }

// parseTasks finds every checklist item of a plan, with its continuation lines.
func parseTasks(plan string) []taskItem {
	all := lines(plan)
	var items []taskItem
	for i := 0; i < len(all); i++ {
		item, ok := itemAt(all, i)
		if !ok {
			continue
		}
		items = append(items, item)
		i = item.End
	}
	return items
}

// itemAt reads the item that starts on line i, if one does.
func itemAt(all []string, i int) (taskItem, bool) {
	head := taskHeader.FindStringSubmatch(all[i])
	if head == nil {
		return taskItem{}, false
	}
	item := taskItem{Start: i, End: i}
	if tag := taskTag.FindStringSubmatch(head[2]); tag != nil {
		item.Type = tag[1]
	}
	for item.End+1 < len(all) && continues(all[item.End+1]) {
		item.End++
	}
	return item, true
}

// continues reports whether a line continues the item above it: indented, not blank, not an item itself.
func continues(line string) bool {
	if strings.TrimSpace(line) == "" || taskHeader.MatchString(line) {
		return false
	}
	return line[0] == ' ' || line[0] == '\t'
}

// typedTasks are the items that carry a type tag; they are the ones the operators rewrite.
func typedTasks(plan string) []taskItem {
	var typed []taskItem
	for _, item := range parseTasks(plan) {
		if item.Type != "" {
			typed = append(typed, item)
		}
	}
	return typed
}

// taskIndexes returns the first line of every typed task.
func taskIndexes(text string) []int {
	var idx []int
	for _, item := range typedTasks(text) {
		idx = append(idx, item.Start)
	}
	return idx
}

// rewriteTasks applies edit to a deterministic fraction of the typed tasks; edit gets the whole item.
func rewriteTasks(text string, ctx Context, edit func(string) string) string {
	all := lines(text)
	tasks := typedTasks(text)
	chosen := pickIndexes(ctx.Seed, ctx.Key, len(tasks), ctx.Level)
	var out []string
	next := 0
	for position, item := range tasks {
		if !chosen[position] {
			continue
		}
		out = append(out, all[next:item.Start]...)
		out = append(out, lines(edit(item.text(all)))...)
		next = item.End + 1
	}
	return join(append(out, all[next:]...))
}
