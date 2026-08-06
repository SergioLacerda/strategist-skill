package treasurecli

import (
	"go/parser"
	"go/token"
	"testing"
)

// TestCmdThinness verifies that the treasure-chest index/add pipelines do not
// reintroduce direct YAML/JSON parsing — that logic lives in internal/treasure.
// A file here that starts importing these again is a sign business logic is
// creeping back into the CLI adapter layer. Continues the invariant
// cmd/strategist's own cmd_isolation_test.go enforced for these two files
// before they moved here (20260806-treasure-chest-cmd-consolidation).
func TestCmdThinness(t *testing.T) {
	t.Parallel()

	files := []string{
		"treasure_chest_index.go",
		"treasure_chest_mutate.go",
	}

	fset := token.NewFileSet()
	for _, file := range files {
		assertNoForbiddenImports(t, fset, file)
	}
}

var forbiddenCmdImports = map[string]bool{
	`"gopkg.in/yaml.v3"`: true,
	`"encoding/json"`:    true,
}

func assertNoForbiddenImports(t *testing.T, fset *token.FileSet, file string) {
	t.Helper()
	f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	for _, imp := range f.Imports {
		if forbiddenCmdImports[imp.Path.Value] {
			t.Errorf("%s imports %s — YAML/JSON parsing belongs in internal/treasure, not internal/treasurecli", file, imp.Path.Value)
		}
	}
}
