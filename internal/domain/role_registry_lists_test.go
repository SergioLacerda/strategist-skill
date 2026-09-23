package domain_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	roleLiteral      = regexp.MustCompile(`"(scout|ranger|archivist|sniper)"`)
	removedRoleTable = regexp.MustCompile(`\b(LevelingRoles|rolePhase|RoleHandoffSchema)\b`)
)

// The role registry is the only place that lists the roles. A second list of
// role names (or the removed per-feature tables) is how the vocabulary drifted
// into five copies; this guard keeps it from coming back.
func TestNoHardcodedRoleListsOutsideTheRegistry(t *testing.T) {
	root := filepath.Join("..", "..")
	registry := filepath.Join(root, "internal", "domain", "role_registry.go")
	for _, path := range productionGoFiles(t, root) {
		if path != registry {
			assertNoRoleLists(t, path)
		}
	}
}

// productionGoFiles lists the non-test Go sources of the repository, skipping
// vendored, generated and runtime trees.
func productionGoFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return skipTree(d.Name())
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)
	return files
}

func skipTree(name string) error {
	switch name {
	case ".git", "node_modules", "defaults", ".strategist", ".analysis":
		return fs.SkipDir
	}
	if strings.HasPrefix(name, ".tmp-") {
		return fs.SkipDir
	}
	return nil
}

func assertNoRoleLists(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // repository source walked by a test
	require.NoError(t, err)
	for i, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		assert.False(t, removedRoleTable.MatchString(line), "%s:%d references a removed role table", path, i+1)
		assert.Less(t, len(roleLiteral.FindAllString(line, -1)), 3, "%s:%d lists several roles; read them from domain.RoleRegistry", path, i+1)
	}
}
