//go:build spec

package spec_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandTreeDocumentationUsesPlatformNeutralBinaryLabel(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	generator := readFile(t, filepath.Join(root, "scripts", "generate-command-tree.sh"))
	if !strings.Contains(generator, `BIN_DISPLAY="${BIN%.exe}"`) {
		t.Fatal("command-tree generator must normalize the Windows executable suffix")
	}
	if !strings.Contains(generator, "walked from \\`$BIN_DISPLAY --help\\`") {
		t.Fatal("command-tree generator must use the normalized binary label")
	}

	commandTree := readFile(t, filepath.Join(root, "docs", "generated", "command-tree.md"))
	if strings.Contains(commandTree, "walked from `bin/strategist.exe --help`") {
		t.Fatal("generated command-tree documentation must not contain a platform-specific executable suffix")
	}
	if !strings.Contains(commandTree, "walked from `bin/strategist --help`") {
		t.Fatal("generated command-tree documentation must name the canonical binary")
	}
}
