package treasurecli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTreasureChestBinary_OldMutationNamespaceIsNotCompatiblyReintroduced(t *testing.T) {
	bin := buildStrategistForParityTest(t)
	result := runParityProcess(t, bin, "treasure-chest", "items", "migrate-status")

	assert.Equal(t, 1, result.exitCode)
	assert.Empty(t, result.stdout)
	assert.Contains(t, result.stderr, `unknown command "treasure-chest"`)
}
