package credential

import "strings"

// ParseDotenv reads KEY=VALUE lines without shell expansion. Comments, blank
// lines and malformed lines are skipped; matching quotes are removed.
func ParseDotenv(content string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		if key, value, ok := parseDotenvLine(line); ok {
			values[key] = value
		}
	}
	return values
}

func parseDotenvLine(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")
	key, raw, found := strings.Cut(line, "=")
	key = strings.TrimSpace(key)
	if !found || key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false
	}
	return key, unquote(strings.TrimSpace(raw)), true
}

func unquote(value string) string {
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
		if end := strings.IndexByte(value[1:], value[0]); end >= 0 {
			return value[1 : 1+end]
		}
	}
	if comment := strings.Index(value, " #"); comment >= 0 {
		return strings.TrimSpace(value[:comment])
	}
	return value
}
