package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// commandWord is the shape of a subcommand name; anything else in command
	// position (`a|b` alternatives, `(prose,`) marks an illustrative line.
	commandWord       = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	shellStopTokens   = map[string]bool{"|": true, "||": true, "<": true, ">": true, ">>": true, "&&": true, ";": true, "2>": true, "2>&1": true}
	ellipsisTokens    = map[string]bool{"...": true, "â€¦": true}
	alwaysKnownFlags  = map[string]bool{"help": true}
	requiredUsageMark = "(required)"
)

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
		token, stop := commandToken(field)
		if token != "" {
			tokens = append(tokens, token)
		}
		if stop {
			break
		}
	}
	return tokens
}

func commandToken(field string) (string, bool) {
	if shellStopTokens[field] {
		return "", true
	}
	if trimmed, ok := strings.CutSuffix(field, ";"); ok {
		return trimmed, true
	}
	return field, false
}

// resolveCommand walks subcommand names. It returns ok=false for an
// illustrative line, and a nil command with the attempted path for an
// unknown subcommand of a command group.
func resolveCommand(tree *cobra.Command, tokens []string) (*cobra.Command, []string, []string, bool) {
	cur := tree
	var path []string
	for i, token := range tokens {
		resolution, next, done := resolveCommandToken(cur, path, tokens, i, token)
		if done {
			return resolution.command, resolution.path, resolution.rest, resolution.ok
		}
		cur = next
		path = append(path, token)
	}
	return cur, path, nil, len(path) > 0
}

type commandResolution struct {
	command *cobra.Command
	path    []string
	rest    []string
	ok      bool
}

func resolveCommandToken(cur *cobra.Command, path, tokens []string, index int, token string) (*commandResolution, *cobra.Command, bool) {
	if isFlagToken(token) || isPlaceholder(token) {
		return &commandResolution{command: cur, path: path, rest: tokens[index:], ok: len(path) > 0}, nil, true
	}
	if !commandWord.MatchString(token) {
		return &commandResolution{}, nil, true
	}
	next := findSubcommand(cur, token)
	if next != nil {
		return nil, next, false
	}
	if cur.HasSubCommands() {
		return &commandResolution{path: append(path, token), ok: true}, nil, true
	}
	return &commandResolution{command: cur, path: path, rest: tokens[index:], ok: len(path) > 0}, nil, true
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
		problems = append(problems, flagProblem(cmd, used, token)...)
	}
	if len(used) == 0 {
		return problems
	}
	return append(problems, missingRequiredFlagProblems(cmd, used)...)
}

func flagProblem(cmd *cobra.Command, used map[string]bool, token string) []string {
	name, ok := longFlagName(token)
	if !ok {
		return nil
	}
	used[name] = true
	if alwaysKnownFlags[name] || cmd.Flags().Lookup(name) != nil || cmd.InheritedFlags().Lookup(name) != nil {
		return nil
	}
	return []string{"unknown flag --" + name}
}

func missingRequiredFlagProblems(cmd *cobra.Command, used map[string]bool) []string {
	var problems []string
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
