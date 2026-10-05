package treasurecli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestBinary_DoctorSeparatesErrorStreamAndExitCode(t *testing.T) {
	bin := buildStrategistForParityTest(t)
	root := parityFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "knowledge.index.yaml"), []byte("sources: []\n"), 0o644))

	result := runParityProcess(t, bin, "treasure-chest", "doctor", "--root", root)
	assert.Equal(t, 1, result.exitCode)
	assert.Empty(t, result.stdout)
	assert.Contains(t, result.stderr, "runbooks")
}
