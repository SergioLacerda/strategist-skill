package main

import (
	"go/parser"
	"go/token"
	"testing"
)

// TestCmdThinness verifies that the cmd/strategist files thinned out by the
// architecture-refactoring extraction (governance sync and validate) do not
// reintroduce direct YAML/JSON parsing — that logic now lives in
// internal/governance and internal/validate. A cmd file that starts importing
// these again is a sign business logic is creeping back into the CLI adapter
// layer. The treasure-chest index/add pipelines' equivalent of this check
// moved with them to internal/treasurecli (see its own cmd_isolation_test.go)
// when the treasure-chest/runbook command cluster was extracted into that
// package (20260806-treasure-chest-cmd-consolidation).
func TestCmdThinness(t *testing.T) {
	t.Parallel()

	files := []string{
		"sync_governance.go",
		"sync_governance_report.go",
		"validate.go",
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
			t.Errorf("%s imports %s — YAML/JSON parsing belongs in internal/, not cmd/strategist", file, imp.Path.Value)
		}
	}
}
