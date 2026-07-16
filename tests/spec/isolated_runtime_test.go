//go:build spec

package spec_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/install"
)

// isolatedStrategistDir builds a self-contained .strategist/ runtime tree from
// the embedded defaults, entirely inside a scratch directory owned by the test
// binary. It never reads or depends on a .strategist/ tree in the developer's
// working copy, so spec tests that need runtime contract files stay hermetic
// and reproducible in CI or any other machine with no prior `strategist
// install` step.
//
// Built once per test binary run and shared read-only across tests: the
// install is expensive (extract + compile) and none of these tests mutate it.
func isolatedStrategistDir(t *testing.T) string {
	t.Helper()

	dir, err := isolatedStrategistDirOnce()
	if err != nil {
		t.Fatalf("build isolated .strategist runtime: %v", err)
	}
	return dir
}

var isolatedStrategistDirOnce = sync.OnceValues(func() (string, error) {
	target, err := os.MkdirTemp("", "strategist-spec-runtime-")
	if err != nil {
		return "", err
	}

	svc := install.Service{
		Extractor: embedpkg.Extractor{},
		Compiler:  compile.Compiler{},
		AwarenessRefresher: func(strategistRoot, projectRoot, version string) bool {
			tplBytes, err := embedpkg.Extractor{}.ReadFile("templates/agent-protocol.md")
			if err != nil {
				tplBytes = nil
			}
			return compile.RefreshAgentAwareness(strategistRoot, projectRoot, version, tplBytes)
		},
		Version: "spec-test",
	}
	cfg := domain.InstallConfig{
		Target: target,
		Silent: true,
		NoShim: true,
	}
	if err := svc.Install(context.Background(), cfg); err != nil {
		os.RemoveAll(target)
		return "", err
	}

	return filepath.Join(target, ".strategist"), nil
})
