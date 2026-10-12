package eval

import "strings"

func normalizeHarvestBlock(block string) string {
	if contentStart := firstFenceLine(block); contentStart >= 0 {
		block = strings.Join([]string{strings.Split(block, "\n")[0], fencedBody(block, contentStart)}, "\n")
	}
	block = strings.ReplaceAll(block, "`", "")
	return normalizeLegacyHarvestFields(block)
}

func normalizeLegacyHarvestFields(block string) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = normalizeHarvestLine(line)
	}
	return strings.Join(lines, "\n")
}

func normalizeHarvestLine(line string) string {
	trimmed := strings.TrimSpace(line)
	indent := len(line) - len(strings.TrimLeft(line, " \t"))
	if normalized, ok := normalizeHarvestListItem(line, trimmed, indent); ok {
		line = normalized
	}
	key, value, ok := strings.Cut(trimmed, ":")
	if ok && strings.TrimSpace(key) == "dependencies" && legacyDependencyIsEmpty(value) {
		return line[:indent] + "dependencies: []"
	}
	return line
}

func normalizeHarvestListItem(line, trimmed string, indent int) (string, bool) {
	if indent == 0 || !strings.HasPrefix(trimmed, "- ") {
		return line, false
	}
	key, _, ok := strings.Cut(strings.TrimPrefix(trimmed, "- "), ":")
	if !ok || key == "" {
		return line, false
	}
	return line[:indent] + strings.TrimPrefix(trimmed, "- "), true
}

func legacyDependencyIsEmpty(value string) bool {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(value), "`\"'")) {
	case "none", "null", "n/a":
		return true
	default:
		return false
	}
}
