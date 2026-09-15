package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// pluginLockFileName is the on-disk artifact persisting the resolved
// discovery/refinement role→weapon bindings across `strategist install` runs
// (docs/adr/0037-wizard-role-binding-persistence.md). Like active.yaml's
// .config.lock seal, it is machine-written/read only — never hand-edited —
// but it is a distinct concept: .config.lock is a content-hash tamper seal
// over active.yaml, this file is the actual activation record (analogous to
// domain.PluginLock/SlotBinding, not to the seal).
const pluginLockFileName = "plugins.lock"

// readPluginLockFile reads and parses plugins.lock under strategistDir. A
// missing file is not an error — it means no binding has ever been
// persisted for this installation (fresh install, or one that predates
// ADR-0037) — and returns a zero-value PluginLockFile for the caller to
// seed from scratch.
func readPluginLockFile(strategistDir string) (domain.PluginLockFile, error) {
	path := filepath.Join(strategistDir, pluginLockFileName)
	data, err := os.ReadFile(path) //nolint:gosec // G304: fixed filename under a caller-owned root, no injection surface
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.PluginLockFile{SchemaVersion: domain.PluginLockFileSchemaVersion}, nil
		}
		return domain.PluginLockFile{}, fmt.Errorf("read %s: %w", pluginLockFileName, err)
	}
	var f domain.PluginLockFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("%s: %w", pluginLockFileName, err)
	}
	return f, nil
}

// writePluginLockFile atomically writes f as plugins.lock under strategistDir.
func writePluginLockFile(strategistDir string, f domain.PluginLockFile) error {
	f.SchemaVersion = domain.PluginLockFileSchemaVersion
	data, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", pluginLockFileName, err)
	}
	path := filepath.Join(strategistDir, pluginLockFileName)
	if err := atomicWriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", pluginLockFileName, err)
	}
	return nil
}
