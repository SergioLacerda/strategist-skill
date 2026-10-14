package check

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// conformanceTree is the command tree of the running binary, set by runCheck
// from cmd.Root() so the invocation-conformance advisory checks agent-facing
// command lines against the commands this binary actually ships.
var conformanceTree *cobra.Command

// conformanceTopLevelFiles and conformanceDirs are the agent-facing runtime
// texts an agent copies command lines from.
var (
	conformanceTopLevelFiles = []string{"SKILL.md", "agent-protocol.md", "protocol.md"}
	conformanceDirs          = []string{"contracts", "roles"}
)

var (
	codeSpanCommand = regexp.MustCompile("`(strategist [^`]+)`")
	quotedCommand   = regexp.MustCompile(`"(strategist [^"]+)"`)
	bareYAMLCommand = regexp.MustCompile(`^\s*(?:-\s+|[\w.-]+:\s+)(strategist\s.+?)(?:\s+#.*)?$`)
	// commandWord is the shape of a subcommand name; anything else in command
	// position (`a|b` alternatives, `(prose,`) marks an illustrative line.
	commandWord       = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	shellStopTokens   = map[string]bool{"|": true, "||": true, "<": true, ">": true, ">>": true, "&&": true, ";": true, "2>": true, "2>&1": true}
	ellipsisTokens    = map[string]bool{"...": true, "…": true}
	alwaysKnownFlags  = map[string]bool{"help": true}
	requiredUsageMark = "(required)"
)

// invocationConformanceAdvisories reports every `strategist …` command line in
// the agent-facing runtime texts that the installed binary would reject: an
// unknown command, an unknown flag, or a missing required flag. A line with no
// flag is a mention, not an invocation, and is never checked for required
// flags. The result is advisory: it never changes readiness or exit status.
func invocationConformanceAdvisories(root string, tree *cobra.Command) []string {
	if tree == nil {
		return nil
	}
	var advisories []string
	seen := make(map[string]bool)
	for _, rel := range conformanceFiles(root) {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) //nolint:gosec // G304: fixed runtime texts under the selected root
		if err != nil {
			continue
		}
		for _, line := range commandLines(rel, string(raw)) {
			for _, problem := range conformanceProblems(tree, line.command) {
				advisory := fmt.Sprintf(
					"[Strategist] phase=preflight status=warn reason=invocation_nonconformant file=%s line=%d command=`%s` detail=%s (the installed binary would reject this documented command line)",
					rel, line.number, line.command, problem)
				if !seen[advisory] {
					seen[advisory] = true
					advisories = append(advisories, advisory)
				}
			}
		}
	}
	return advisories
}

// conformanceFiles lists the top-level texts first, then the walked
// directories in sorted order, so advisories are reported deterministically.
func conformanceFiles(root string) []string {
	var files []string
	for _, name := range conformanceTopLevelFiles {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			files = append(files, name)
		}
	}
	var walked []string
	for _, dir := range conformanceDirs {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error { //nolint:errcheck // a missing directory simply contributes no files
			if err != nil || d.IsDir() || !isConformanceText(path) {
				return nil //nolint:nilerr // unreadable entries are skipped, not fatal for an advisory
			}
			if rel, relErr := filepath.Rel(root, path); relErr == nil {
				walked = append(walked, filepath.ToSlash(rel))
			}
			return nil
		})
	}
	sort.Strings(walked)
	return append(files, walked...)
}

func isConformanceText(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".yaml", ".yml":
		return true
	}
	return false
}

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
		if strings.HasPrefix(trimmed, "strategist ") {
			return []commandLine{{number: number, command: trimmed}}
		}
		return nil
	}

	var out []commandLine
	found := false
	for _, re := range []*regexp.Regexp{codeSpanCommand, quotedCommand} {
		for _, m := range re.FindAllStringSubmatch(line, -1) {
			out = append(out, commandLine{number: number, command: strings.TrimSpace(m[1])})
			found = true
		}
	}
	if !found && isYAML {
		if m := bareYAMLCommand.FindStringSubmatch(line); m != nil {
			out = append(out, commandLine{number: number, command: strings.TrimSpace(m[1])})
		}
	}
	return out
}

// conformanceProblems resolves one command line against tree and lists what
// the binary would reject. Illustrative lines (alternatives such as
// `start|status|submit`, or a placeholder in command position) are skipped.
func conformanceProblems(tree *cobra.Command, command string) []string {
	tokens := commandTokens(command)
	if len(tokens) < 2 || tokens[0] != "strategist" {
		return nil
	}
	cmd, path, rest, ok := resolveCommand(tree, tokens[1:])
	if !ok {
		return nil
	}
	if cmd == nil {
		return []string{fmt.Sprintf("unknown command %q", strings.Join(path, " "))}
	}
	return flagProblems(cmd, rest)
}

// commandTokens splits a command line and stops at the first shell operator,
// including a `;` glued to the end of a token (`check-stale;`).
func commandTokens(command string) []string {
	var tokens []string
	for _, field := range strings.Fields(command) {
		if shellStopTokens[field] {
			break
		}
		if trimmed, ok := strings.CutSuffix(field, ";"); ok {
			if trimmed != "" {
				tokens = append(tokens, trimmed)
			}
			break
		}
		tokens = append(tokens, field)
	}
	return tokens
}

// resolveCommand walks subcommand names. It returns ok=false for an
// illustrative line, and a nil command with the attempted path for an
// unknown subcommand of a command group.
func resolveCommand(tree *cobra.Command, tokens []string) (*cobra.Command, []string, []string, bool) {
	cur := tree
	var path []string
	for i, token := range tokens {
		if isFlagToken(token) || isPlaceholder(token) {
			return cur, path, tokens[i:], len(path) > 0
		}
		if !commandWord.MatchString(token) {
			return nil, nil, nil, false
		}
		next := findSubcommand(cur, token)
		if next == nil {
			if cur.HasSubCommands() {
				return nil, append(path, token), nil, true
			}
			return cur, path, tokens[i:], len(path) > 0
		}
		cur = next
		path = append(path, token)
	}
	return cur, path, nil, len(path) > 0
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if child.Name() == name || child.HasAlias(name) {
			return child
		}
	}
	return nil
}

func isFlagToken(token string) bool {
	return strings.HasPrefix(strings.TrimLeft(token, "["), "-")
}

func isPlaceholder(token string) bool {
	return strings.HasPrefix(token, "<") || strings.HasPrefix(token, "[") && !isFlagToken(token)
}

func flagProblems(cmd *cobra.Command, rest []string) []string {
	used := make(map[string]bool)
	var problems []string
	for _, token := range rest {
		name, ok := longFlagName(token)
		if !ok {
			continue
		}
		used[name] = true
		if !alwaysKnownFlags[name] && cmd.Flags().Lookup(name) == nil && cmd.InheritedFlags().Lookup(name) == nil {
			problems = append(problems, "unknown flag --"+name)
		}
	}
	if len(used) == 0 {
		return problems
	}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if isRequiredFlag(f) && !used[f.Name] {
			problems = append(problems, "missing required flag --"+f.Name)
		}
	})
	return problems
}

func longFlagName(token string) (string, bool) {
	token = strings.Trim(token, "[]")
	if ellipsisTokens[token] || !strings.HasPrefix(token, "--") {
		return "", false
	}
	name := strings.TrimPrefix(token, "--")
	if eq := strings.IndexByte(name, '='); eq >= 0 {
		name = name[:eq]
	}
	return name, name != ""
}

func isRequiredFlag(f *pflag.Flag) bool {
	if values, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok && len(values) > 0 && values[0] == "true" {
		return true
	}
	return strings.Contains(f.Usage, requiredUsageMark)
}
