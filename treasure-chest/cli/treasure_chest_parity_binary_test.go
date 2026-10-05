package treasurecli

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestBinary_EveryCommandHasMigrationSmokeCoverage(t *testing.T) {
	bin := buildStrategistForParityTest(t)
	tests := []struct {
		name    string
		fixture func(*testing.T) string
		args    func(string) []string
		check   func(*testing.T, string, parityProcessResult)
	}{
		{
			name: "bare status json",
			args: func(root string) []string { return []string{"treasure-chest", "--root", root, "--format", "json"} },
			check: func(t *testing.T, _ string, result parityProcessResult) {
				assertParitySuccess(t, result)
				var decoded map[string]any
				require.NoError(t, json.Unmarshal([]byte(result.stdout), &decoded))
				assert.Contains(t, decoded, "chests")
			},
		},
		{
			name: "explicit list",
			args: func(root string) []string {
				return []string{"treasure-chest", "list", "--root", root, "--format", "json"}
			},
			check: func(t *testing.T, _ string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "chests")
			},
		},
		{
			name: "add",
			args: func(root string) []string {
				return []string{"treasure-chest", "add", "docs/new-source", "--root", root, "--id", "extra", "--scope", "discovery", "--tags", "migration,source"}
			},
			check: func(t *testing.T, root string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "id=extra")
				assert.Contains(t, readParityFile(t, root, "active.yaml"), "id: extra")
			},
		},
		{
			name: "remove",
			args: func(root string) []string {
				return []string{"treasure-chest", "remove", "--root", root, "--id", "source"}
			},
			check: func(t *testing.T, root string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "id=source")
				assert.Contains(t, readParityFile(t, root, "treasure-chests.yaml"), "status: inactive")
				assert.Contains(t, readParityFile(t, root, "jewels.yaml"), "status: deprecated")
			},
		},
		{
			name: "doctor",
			args: func(root string) []string { return []string{"treasure-chest", "doctor", "--root", root} },
			check: func(t *testing.T, _ string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "no consistency drift")
			},
		},
		{
			name: "index",
			args: func(root string) []string { return []string{"treasure-chest", "index", "--root", root} },
			check: func(t *testing.T, root string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "compiled artifact refreshed")
				assert.FileExists(t, filepath.Join(root, ".compiled", ".index.gz"))
			},
		},
		{
			name:    "hidden scan dry run",
			fixture: parityScanFixture,
			args: func(root string) []string {
				return []string{"treasure-chest", "scan", "--root", root, "--dry-run"}
			},
			check: func(t *testing.T, root string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "dry-run")
				assert.NoDirExists(t, filepath.Join(root, "treasure", "gaps"))
			},
		},
		{
			name: "items list",
			args: func(root string) []string {
				return []string{"treasure-chest", "items", "list", "--root", root, "--format", "json"}
			},
			check: func(t *testing.T, _ string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "jewel-1")
				assert.Contains(t, result.stdout, "potion-1")
			},
		},
		{
			name: "items show",
			args: func(root string) []string {
				return []string{"treasure-chest", "items", "show", "jewel-1", "--root", root}
			},
			check: func(t *testing.T, _ string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "Widgets require explicit teardown.")
			},
		},
		{
			name: "items accept",
			args: func(root string) []string {
				return []string{"treasure-chest", "items", "accept", "jewel-1", "--root", root}
			},
			check: func(t *testing.T, root string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "status: accepted")
				assert.Contains(t, readParityFile(t, root, "jewels.yaml"), "status: accepted")
			},
		},
		{
			name: "items verify",
			args: func(root string) []string {
				return []string{"treasure-chest", "items", "verify", "jewel-1", "--root", root, "--evidence", "source#widgets"}
			},
			check: func(t *testing.T, root string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "status: verified")
				assert.Contains(t, readParityFile(t, root, "jewels.yaml"), "source#widgets")
			},
		},
		{
			name: "items deprecate",
			args: func(root string) []string {
				return []string{"treasure-chest", "items", "deprecate", "jewel-1", "--root", root}
			},
			check: func(t *testing.T, root string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "status: deprecated")
				assert.Contains(t, readParityFile(t, root, "jewels.yaml"), "status: deprecated")
			},
		},
		{
			name:    "items migrate-status",
			fixture: parityMigrationFixture,
			args: func(root string) []string {
				return []string{"treasure-chest", "items", "migrate-status", "--root", root}
			},
			check: func(t *testing.T, root string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "migrated active -> accepted")
				assert.Contains(t, readParityFile(t, root, "jewels.yaml"), "status: accepted")
			},
		},
		{
			name: "items check-evidence",
			args: func(root string) []string {
				return []string{"treasure-chest", "items", "check-evidence", "source", "--root", root}
			},
			check: func(t *testing.T, _ string, result parityProcessResult) {
				assertParitySuccess(t, result)
				assert.Contains(t, result.stdout, "no findings")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := parityFixture
			if tt.fixture != nil {
				fixture = tt.fixture
			}
			root := fixture(t)
			tt.check(t, root, runParityProcess(t, bin, tt.args(root)...))
		})
	}
}
