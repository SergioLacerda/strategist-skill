package opportunity_attack

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProductionPackageDoesNotImportTreasureChest(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	entries, err := filepath.Glob(filepath.Join(filepath.Dir(sourceFile), "*.go"))
	require.NoError(t, err)
	assertNoTreasureChestImports(t, entries)
}

func assertNoTreasureChestImports(t *testing.T, entries []string) {
	const treasureChestImport = "github.com/SergioLacerda/strategist-skill/treasure-chest"
	for _, path := range entries {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		assertFileImportsAllowed(t, path, treasureChestImport)
	}
}

func assertFileImportsAllowed(t *testing.T, path, forbiddenPrefix string) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	require.NoError(t, err)
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		require.NoError(t, err)
		if strings.HasPrefix(importPath, forbiddenPrefix) {
			t.Fatalf("Opportunity Attack production file %s imports Treasure Chest: %s", path, importPath)
		}
	}
}
