package check

import (
	"regexp"
	"strings"
)

var (
	codeSpanCommand = regexp.MustCompile("`(strategist [^`]+)`")
	quotedCommand   = regexp.MustCompile(`"(strategist [^"]+)"`)
	bareYAMLCommand = regexp.MustCompile(`^\s*(?:-\s+|[\w.-]+:\s+)(strategist\s.+?)(?:\s+#.*)?$`)
)

type commandLine struct {
	number  int
	command string
}

// commandLines extracts candidate command lines: code spans and quoted strings
// anywhere, every line of a fenced code block, and bare YAML scalar values.
func commandLines(rel, text string) []commandLine {
	isYAML := !strings.HasSuffix(strings.ToLower(rel), ".md")
	var out []commandLine
	inFence := false
	for i, line := range strings.Split(text, "\n") {
		out = append(out, commandLineCandidates(i+1, line, isYAML, &inFence)...)
	}
	return out
}

func commandLineCandidates(number int, line string, isYAML bool, inFence *bool) []commandLine {
	line = strings.TrimRight(line, "\r")
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") {
		*inFence = !*inFence
		return nil
	}
	if *inFence {
		return fencedCommandLine(number, trimmed)
	}
	return inlineCommandLines(number, line, isYAML)
}

func fencedCommandLine(number int, line string) []commandLine {
	if !strings.HasPrefix(line, "strategist ") {
		return nil
	}
	return []commandLine{{number: number, command: line}}
}

func inlineCommandLines(number int, line string, isYAML bool) []commandLine {
	var out []commandLine
	found := false
	for _, re := range []*regexp.Regexp{codeSpanCommand, quotedCommand} {
		for _, m := range re.FindAllStringSubmatch(line, -1) {
			out = append(out, commandLine{number: number, command: strings.TrimSpace(m[1])})
			found = true
		}
	}
	if found || !isYAML {
		return out
	}
	if m := bareYAMLCommand.FindStringSubmatch(line); m != nil {
		out = append(out, commandLine{number: number, command: strings.TrimSpace(m[1])})
	}
	return out
}
