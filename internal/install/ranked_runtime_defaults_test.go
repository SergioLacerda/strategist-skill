package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/runtimepayload"
)

func init() {
	// Production startup registers the embedded FS from cmd/strategist. Package
	// tests cannot import internal/embed because its white-box tests exercise
	// install; use the checked-in defaults tree as the equivalent source.
	runtimepayload.RegisterOpenSpec(os.DirFS(filepath.Join("..", "embed", "defaults")))
}

func TestRankedRuntimeTestDefaultsExist(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "embed", "defaults", "skills", "openspec-propose", "runtime", "runtime.build.yaml")); err != nil {
		t.Fatal(err)
	}
}
