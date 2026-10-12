package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
)

func TestRankedRuntimeTestDefaultsExist(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "embed", "defaults", "skills", installtest.OpenSpecProposePayloadDir, "runtime", "runtime.build.yaml")); err != nil {
		t.Fatal(err)
	}
}
