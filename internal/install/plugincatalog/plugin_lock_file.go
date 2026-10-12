package plugincatalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/support"
	"gopkg.in/yaml.v3"
)

// ReadPluginLockFile reads and parses plugins.lock under strategistDir. A
// missing file is not an error — it means no binding has ever been
// persisted for this installation (fresh install, or one that predates
// ADR-0037) — and returns a zero-value PluginLockFile for the caller to
// seed from scratch.
func ReadPluginLockFile(strategistDir string) (domain.PluginLockFile, error) {
	path := filepath.Join(strategistDir, names.PluginLockFileName)
	data, err := os.ReadFile(path) //nolint:gosec // G304: fixed filename under a caller-owned root, no injection surface
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.PluginLockFile{SchemaVersion: domain.PluginLockFileSchemaVersion}, nil
		}
		return domain.PluginLockFile{}, fmt.Errorf("read %s: %w", names.PluginLockFileName, err)
	}
	var f domain.PluginLockFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("%s: %w", names.PluginLockFileName, err)
	}
	if err := domain.ValidateWeaponBindingArtifacts(f.Bindings, f.BindingArtifacts); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("%s: invalid binding artifacts: %w", names.PluginLockFileName, err)
	}
	return f, nil
}

// WritePluginLockFile atomically writes f as plugins.lock under strategistDir.
func WritePluginLockFile(strategistDir string, f domain.PluginLockFile) error {
	f.SchemaVersion = domain.PluginLockFileSchemaVersion
	if err := domain.ValidateWeaponBindingArtifacts(f.Bindings, f.BindingArtifacts); err != nil {
		return fmt.Errorf("validate %s: %w", names.PluginLockFileName, err)
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", names.PluginLockFileName, err)
	}
	path := filepath.Join(strategistDir, names.PluginLockFileName)
	if err := support.AtomicWriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", names.PluginLockFileName, err)
	}
	return nil
}
