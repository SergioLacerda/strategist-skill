package rankedbinding

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/filesystem"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"gopkg.in/yaml.v3"
)

// PrepareRankedProviderRuntimes materializes the runtime contract of each
// selected Ranked slot after active.yaml/plugins.lock have been persisted.
// It intentionally reads the installed catalog, not the build checkout, so
// the installed binding and runtime cannot drift silently.
func PrepareRankedProviderRuntimes(ctx context.Context, strategistDir string) error {
	rankedBindings, err := LoadRankedBindings(strategistDir)
	if err != nil {
		return err
	}
	if len(rankedBindings) == 0 {
		return nil
	}
	roles, catalog, err := LoadRankedRuntimeInputs(strategistDir)
	if err != nil {
		return err
	}
	state, err := prepareRankedRuntimeState(ctx, strategistDir, roles, catalog, rankedBindings)
	if err != nil {
		return err
	}
	return writeRankedRuntimeState(strategistDir, state)
}

func prepareRankedRuntimeState(ctx context.Context, strategistDir string, roles domain.RoleSlotMap, catalog plugincatalog.PluginCatalog, bindings []domain.SlotBinding) (domain.RankedRuntimeState, error) {
	state := domain.RankedRuntimeState{SchemaVersion: domain.RankedRuntimeStateSchemaVersion}
	for _, binding := range bindings {
		entry, required, err := PrepareRankedBinding(ctx, strategistDir, roles, catalog, binding)
		if err != nil {
			return domain.RankedRuntimeState{}, err
		}
		if required {
			state.Entries = append(state.Entries, entry)
		}
	}
	return state, nil
}

// LoadRankedBindings reads the persisted slot bindings from plugins.lock under strategistDir; a missing lock yields none.
func LoadRankedBindings(strategistDir string) ([]domain.SlotBinding, error) {
	lockRaw, err := os.ReadFile(filepath.Join(strategistDir, names.PluginLockFileName)) //nolint:gosec // fixed path under the install root
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugins.lock for ranked runtime: %w", err)
	}
	var lock domain.PluginLockFile
	if err := yaml.Unmarshal(lockRaw, &lock); err != nil {
		return nil, fmt.Errorf("parse plugins.lock for ranked runtime: %w", err)
	}
	bindings := make([]domain.SlotBinding, 0, len(lock.Bindings))
	for _, binding := range lock.Bindings {
		if binding.EffectiveMode() == domain.SlotBindingModeRanked {
			bindings = append(bindings, binding)
		}
	}
	return bindings, nil
}

// LoadRankedRuntimeInputs reads the role map and the plugin catalog under strategistDir that Ranked runtime
// preparation needs.
func LoadRankedRuntimeInputs(strategistDir string) (domain.RoleSlotMap, plugincatalog.PluginCatalog, error) {
	roleRaw, err := os.ReadFile(filepath.Join(strategistDir, "roles", "default.yaml")) //nolint:gosec // fixed path under the install root
	if err != nil {
		return nil, plugincatalog.PluginCatalog{}, fmt.Errorf("read role map for ranked runtime: %w", err)
	}
	var roles domain.RoleSlotMap
	if err := yaml.Unmarshal(roleRaw, &roles); err != nil {
		return nil, plugincatalog.PluginCatalog{}, fmt.Errorf("parse role map for ranked runtime: %w", err)
	}
	catalogRaw, err := os.ReadFile(filepath.Join(strategistDir, names.PluginCatalogPath)) //nolint:gosec // fixed path under the install root
	if err != nil {
		return nil, plugincatalog.PluginCatalog{}, fmt.Errorf("read catalog for ranked runtime: %w", err)
	}
	catalog, err := plugincatalog.ParseCatalogBytes(catalogRaw)
	if err != nil {
		return nil, plugincatalog.PluginCatalog{}, fmt.Errorf("parse catalog for ranked runtime: %w", err)
	}
	return roles, catalog, nil
}

// PrepareRankedBinding resolves and bootstraps the private OpenSpec runtime of one binding and returns its state
// entry; the bool is false when the binding needs no private runtime.
func PrepareRankedBinding(ctx context.Context, strategistDir string, roles domain.RoleSlotMap, catalog plugincatalog.PluginCatalog, binding domain.SlotBinding) (domain.RankedRuntimeStateEntry, bool, error) {
	provider, runtime, err := ResolveRankedProvider(catalog, binding)
	if err != nil || runtime.Kind == domain.RankedRuntimeNone || runtime.Kind != domain.RankedRuntimeOpenSpecRoot {
		return domain.RankedRuntimeStateEntry{}, false, err
	}
	private, err := BootstrapRankedProvider(ctx, strategistDir, provider, runtime)
	if err != nil {
		return domain.RankedRuntimeStateEntry{}, false, err
	}
	return domain.RankedRuntimeStateEntry{
		Runtime:        private,
		Role:           roles[binding.Slot],
		Slot:           binding.Slot,
		Provider:       provider.ID,
		ContractDigest: provider.CertificationDigest,
		Root:           runtime.Root,
		Kind:           runtime.Kind,
	}, true, nil
}

// BootstrapOpenSpecRuntimeWith bootstraps the private OpenSpec runtime at root with exe, or validates it when it
// already exists.
func BootstrapOpenSpecRuntimeWith(ctx context.Context, root string, runtime domain.WeaponRuntime, exe RankedExecutable) error {
	bootstrapArgs, err := openSpecCommandArgs(runtime.Bootstrap, "init")
	if err != nil {
		return fmt.Errorf("invalid bootstrap command: %w", err)
	}
	healthcheckArgs, err := openSpecCommandArgs(runtime.Healthcheck, "context")
	if err != nil {
		return fmt.Errorf("invalid healthcheck command: %w", err)
	}
	if filesystem.Exists(filepath.Join(root, names.RuntimeConfigName)) {
		return validateExistingOpenSpecRuntime(ctx, root, exe, healthcheckArgs)
	}
	return InitializeOpenSpecRuntime(ctx, root, exe, bootstrapArgs, healthcheckArgs)
}

func validateExistingOpenSpecRuntime(ctx context.Context, root string, exe RankedExecutable, healthcheckArgs []string) error {
	output, err := RunRankedRuntimeCommand(ctx, root, exe.Name, exe.Args(healthcheckArgs)...)
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}
	if err := domain.ValidateOpenSpecHealthcheck(output, root, filesystem.PathResolver{}); err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}
	return RemoveLegacyNestedOpenSpecRoot(root)
}

// InitializeOpenSpecRuntime runs the bootstrap and healthcheck commands of the OpenSpec runtime at root.
func InitializeOpenSpecRuntime(ctx context.Context, root string, exe RankedExecutable, bootstrapArgs, healthcheckArgs []string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create root %s: %w", root, err)
	}
	if _, err := RunRankedRuntimeCommand(ctx, filepath.Dir(root), exe.Name, exe.Args(bootstrapArgs)...); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	if !filesystem.Exists(filepath.Join(root, names.RuntimeConfigName)) {
		return fmt.Errorf("bootstrap completed without %s/config.yaml", root)
	}
	return validateExistingOpenSpecRuntime(ctx, root, exe, healthcheckArgs)
}
