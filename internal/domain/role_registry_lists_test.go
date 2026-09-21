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

var roleLiteral = regexp.MustCompile(`"(scout|ranger|archivist|sniper)"`)

// The role registry is the only place that lists the roles. A second list of
// role names (or the removed per-feature tables) is how the vocabulary drifted
// into five copies; this guard keeps it from coming back.
func TestNoHardcodedRoleListsOutsideTheRegistry(t *testing.T) {
	root := filepath.Join("..", "..")
	removed := regexp.MustCompile(`\b(LevelingRoles|rolePhase|RoleHandoffSchema)\b`)
	allowed := map[string]bool{filepath.Join(root, "internal", "domain", "role_registry.go"): true}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "defaults", ".strategist", ".analysis":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || allowed[path] {
			return nil
		}
		raw, readErr := os.ReadFile(path) //nolint:gosec // repository source walked by a test
		require.NoError(t, readErr)
		for i, line := range strings.Split(string(raw), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			assert.False(t, removed.MatchString(line), "%s:%d references a removed role table", path, i+1)
			assert.Less(t, len(roleLiteral.FindAllString(line, -1)), 3, "%s:%d lists several roles; read them from domain.RoleRegistry", path, i+1)
		}
		return nil
	})
	require.NoError(t, err)
}
